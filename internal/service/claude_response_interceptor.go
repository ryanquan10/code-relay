package service

import (
	"io"
	"log"
	"sync"
)

// ClaudeUsageSession 为单次 Claude 请求/响应聚合用量（优先官方 usage）
type ClaudeUsageSession struct {
	customerToken string
	isSSE         bool

	mu sync.Mutex

	inBytes  uint64
	outBytes uint64

	inBuffer  []byte
	outBuffer []byte

	responseBuffer []byte // 缓存完整响应用于解析
	officialIn     uint64
	officialOut    uint64
	officialModel  *string
	officialFound  bool

	finalizeOnce sync.Once
	finalizeErr  error
}

// NewClaudeUsageSession 创建 Claude 用量会话
func NewClaudeUsageSession(customerToken string, isSSE bool) *ClaudeUsageSession {
	trackingService := NewUserTrackingService()
	if err := trackingService.TrackUserAccess(customerToken); err != nil {
		log.Printf("[警告] 记录用户访问失败: %v", err)
	}

	return &ClaudeUsageSession{
		customerToken:  customerToken,
		isSSE:          isSSE,
		inBuffer:       make([]byte, 0, 8192),
		outBuffer:      make([]byte, 0, 8192),
		responseBuffer: make([]byte, 0, 8192),
	}
}

// AddBytes 增加字节数统计
func (s *ClaudeUsageSession) AddBytes(n int, dir string) {
	if n <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch dir {
	case "in":
		s.inBytes += uint64(n)
	case "out":
		s.outBytes += uint64(n)
	}
}

// AppendContent 追加内容到缓存
func (s *ClaudeUsageSession) AppendContent(p []byte, dir string) {
	if len(p) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	switch dir {
	case "in":
		if len(s.inBuffer) < 32768 {
			remain := 32768 - len(s.inBuffer)
			if remain > len(p) {
				remain = len(p)
			}
			s.inBuffer = append(s.inBuffer, p[:remain]...)
		}
	case "out":
		if len(s.outBuffer) < 32768 {
			remain := 32768 - len(s.outBuffer)
			if remain > len(p) {
				remain = len(p)
			}
			s.outBuffer = append(s.outBuffer, p[:remain]...)
		}
	}
}

// FeedResponseChunk 喂入响应数据块，尝试提取官方 usage
func (s *ClaudeUsageSession) FeedResponseChunk(p []byte) {
	if len(p) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.officialFound {
		return
	}

	// 缓存响应内容（最多 1MB，避免内存溢出）
	if len(s.responseBuffer) < 1024*1024 {
		remain := 1024*1024 - len(s.responseBuffer)
		if remain > len(p) {
			remain = len(p)
		}
		s.responseBuffer = append(s.responseBuffer, p[:remain]...)
	}
}

// FinalizeAndSend 完成并发送用量数据（优先官方，回退估算）
func (s *ClaudeUsageSession) FinalizeAndSend() error {
	s.finalizeOnce.Do(func() {
		s.finalizeErr = s.finalizeAndSend()
	})
	return s.finalizeErr
}

func (s *ClaudeUsageSession) finalizeAndSend() error {
	s.mu.Lock()
	customerToken := s.customerToken
	isSSE := s.isSSE
	responseBuffer := append([]byte(nil), s.responseBuffer...)

	inBytes := s.inBytes
	outBytes := s.outBytes
	inBuf := append([]byte(nil), s.inBuffer...)
	outBuf := append([]byte(nil), s.outBuffer...)
	s.mu.Unlock()

	if customerToken == "" {
		return nil
	}

	// 尝试从响应中提取官方 token 信息
	if len(responseBuffer) > 0 {
		usageInfo, err := ParseClaudeResponse(responseBuffer, isSSE)
		if err == nil && usageInfo != nil {
			s.mu.Lock()
			s.officialIn = usageInfo.InputTokens
			s.officialOut = usageInfo.OutputTokens
			s.officialModel = usageInfo.Model
			s.officialFound = true
			s.mu.Unlock()

			totalTokens := usageInfo.InputTokens + usageInfo.OutputTokens
			modelStr := "unknown"
			if usageInfo.Model != nil {
				modelStr = *usageInfo.Model
			}

			log.Printf("[Claude] 提取到官方 token 使用量: in=%d, out=%d, total=%d, model=%s",
				usageInfo.InputTokens, usageInfo.OutputTokens, totalTokens, modelStr)

			// 发送到 Redis Stream（由消费者写入 MySQL）
			{
				var origin *string
				if len(inBuf) > 0 {
					tmp := string(inBuf)
					origin = &tmp
				}
				return SendClaudeTokenUsageToStreamWithIOAndMessage(customerToken, totalTokens, usageInfo.InputTokens, usageInfo.OutputTokens, usageInfo.Model, origin)
			}
		}
	}

	// 回退到估算
	log.Printf("[Claude] 未找到官方 token 信息，使用估算值")

	var inTokens, outTokens uint64
	if len(inBuf) > 0 {
		inTokens = EstimateTokensFromText(string(inBuf))
	} else {
		inTokens = BytesToTokens(inBytes)
	}
	if len(outBuf) > 0 {
		outTokens = EstimateTokensFromText(string(outBuf))
	} else {
		outTokens = BytesToTokens(outBytes)
	}

	total := inTokens + outTokens
	if total == 0 {
		return nil
	}

	{
		var origin *string
		if len(inBuf) > 0 {
			tmp := string(inBuf)
			origin = &tmp
		}
		return SendClaudeTokenUsageToStreamWithIOAndMessage(customerToken, total, inTokens, outTokens, nil, origin)
	}
}

// claudeCountingReadCloser 用于 Claude：抓取官方 usage，缺失回退估算
type claudeCountingReadCloser struct {
	R         io.ReadCloser
	Session   *ClaudeUsageSession
	Direction string
}

func NewClaudeCountingReadCloser(r io.ReadCloser, session *ClaudeUsageSession, dir string) io.ReadCloser {
	return &claudeCountingReadCloser{
		R:         r,
		Session:   session,
		Direction: dir,
	}
}

func (c *claudeCountingReadCloser) Read(p []byte) (int, error) {
	n, err := c.R.Read(p)
	if n > 0 && c.Session != nil {
		c.Session.AddBytes(n, c.Direction)
		c.Session.AppendContent(p[:n], c.Direction)
		if c.Direction == "out" {
			c.Session.FeedResponseChunk(p[:n])
		}
	}
	return n, err
}

func (c *claudeCountingReadCloser) Close() error {
	if c.Session != nil && c.Direction == "out" {
		if err := c.Session.FinalizeAndSend(); err != nil {
			log.Printf("[警告] Claude 最后刷新失败: %v", err)
		}
	}
	return c.R.Close()
}
