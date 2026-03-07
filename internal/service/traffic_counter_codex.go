package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	redisstore "codex-relay/internal/redis"
)

const (
	// TokensPerKB 每 KB 大约对应的 token 数
	TokensPerKB = 256

	// TokensPerKBChinese 中文文本每 KB 对应的 token 数
	TokensPerKBChinese = 600

	// FlushThreshold 达到多少 tokens 时触发写入 Stream（1k tokens）
	FlushThreshold = 1000

	// StreamKey Redis Stream 的 key
	StreamKey = "token_usage_stream"

	// StreamMaxLen Stream 最大长度（防止堆积）
	StreamMaxLen = 10000
)

// 正则表达式：用于从 JSON 中提取字段（兜底方案）
var (
	// gptModelRegex 匹配 "model":"gpt-xxxx" 格式
	gptModelRegex = regexp.MustCompile(`"model"\s*:\s*"(gpt-[^"]+)"`)
	// inputTokensRegex 匹配 "input_tokens":123 格式
	inputTokensRegex = regexp.MustCompile(`"input_tokens"\s*:\s*(\d+)`)
	// outputTokensRegex 匹配 "output_tokens":456 格式
	outputTokensRegex = regexp.MustCompile(`"output_tokens"\s*:\s*(\d+)`)
	// totalTokensRegex 匹配 "total_tokens":789 格式
	totalTokensRegex = regexp.MustCompile(`"total_tokens"\s*:\s*(\d+)`)
)

// TrafficCounter 流量计数器（按方向拆分）
type TrafficCounter struct {
	// 按方向累计的字节数
	InputBytes  uint64
	OutputBytes uint64

	// 按方向累计的已刷新 token（用于计算增量）
	PendingTokensIn  uint64
	PendingTokensOut uint64

	StartTime     time.Time
	customerToken string
	model         *string // 模型名称（可选）

	// 按方向缓存内容（用于更精确的 token 估算）
	inputBuffer  []byte
	outputBuffer []byte

	inputBufferMu  sync.Mutex
	outputBufferMu sync.Mutex
}

// NewTrafficCounter 初始化流量计数器
func NewTrafficCounter(customerToken string, model ...*string) *TrafficCounter {
	// 记录用户访问
	trackingService := NewUserTrackingService()
	if err := trackingService.TrackUserAccess(customerToken); err != nil {
		log.Printf("[警告] 记录用户访问失败: %v", err)
	}

	var modelPtr *string
	if len(model) > 0 {
		modelPtr = model[0]
	}

	return &TrafficCounter{
		StartTime:     time.Now(),
		customerToken: customerToken,
		model:         modelPtr,
		inputBuffer:   make([]byte, 0, 8192),
		outputBuffer:  make([]byte, 0, 8192),
	}
}

// AddBytes 增加字节数（第二个参数表示方向："in" 或 "out"）
func (t *TrafficCounter) AddBytes(n int64, dir string) {
	if n <= 0 {
		return
	}
	switch dir {
	case "in":
		atomic.AddUint64(&t.InputBytes, uint64(n))
	case "out":
		atomic.AddUint64(&t.OutputBytes, uint64(n))
	default:
		// 未知方向，忽略
	}
}

// GetTotalBytes 获取总字节数
func (t *TrafficCounter) GetTotalBytes() uint64 {
	return atomic.LoadUint64(&t.InputBytes) + atomic.LoadUint64(&t.OutputBytes)
}

// AppendContent 追加内容到缓存（按方向，便于更精确 token 估算）
func (t *TrafficCounter) AppendContent(data []byte, dir string) {
	if len(data) == 0 {
		return
	}
	switch dir {
	case "in":
		t.inputBufferMu.Lock()
		if len(t.inputBuffer) < 32768 { // 最多缓存 32KB
			t.inputBuffer = append(t.inputBuffer, data...)
		}
		t.inputBufferMu.Unlock()
	case "out":
		t.outputBufferMu.Lock()
		if len(t.outputBuffer) < 32768 { // 最多缓存 32KB
			t.outputBuffer = append(t.outputBuffer, data...)
		}
		t.outputBufferMu.Unlock()
	}
}

// Reset 重置计数器
func (t *TrafficCounter) Reset() {
	atomic.StoreUint64(&t.InputBytes, 0)
	atomic.StoreUint64(&t.OutputBytes, 0)
	atomic.StoreUint64(&t.PendingTokensIn, 0)
	atomic.StoreUint64(&t.PendingTokensOut, 0)

	t.inputBufferMu.Lock()
	t.inputBuffer = nil
	t.inputBufferMu.Unlock()

	t.outputBufferMu.Lock()
	t.outputBuffer = nil
	t.outputBufferMu.Unlock()

	t.StartTime = time.Now()
}

// EstimateTokensFromText 从文本内容估算 token 数
// 英文：约 4 个字符 = 1 token
// 中文：约 1.7 个汉字 = 1 token
func EstimateTokensFromText(text string) uint64 {
	if len(text) == 0 {
		return 0
	}

	runes := []rune(text)
	chineseCount := 0
	totalCount := len(runes)

	for _, r := range runes {
		// 判断是否为中文字符（CJK 统一汉字）
		if (r >= 0x4E00 && r <= 0x9FFF) || // 常用汉字
			(r >= 0x3400 && r <= 0x4DBF) || // 扩展A
			(r >= 0x20000 && r <= 0x2A6DF) { // 扩展B
			chineseCount++
		}
	}

	englishCount := totalCount - chineseCount

	// 英文：4 字符/token，中文：1.7 字符/token
	tokens := uint64(float64(englishCount)/4.0 + float64(chineseCount)/1.7)
	if tokens == 0 {
		tokens = 1
	}
	return tokens
}

// BytesToTokens 将字节数转换为大约的 token 数（快速估算）
func BytesToTokens(bytes uint64) uint64 {
	kb := bytes / 1024
	if kb == 0 {
		kb = 1
	}
	// 混合比例：30% 中文，70% 英文
	avgTokensPerKB := uint64((TokensPerKB*7 + TokensPerKBChinese*3 + 5) / 10)
	return kb * avgTokensPerKB
}

// CheckAndFlushToStream 检查是否需要将 tokens 写入 Redis Stream（带方向拆分）
func (t *TrafficCounter) CheckAndFlushToStream() error {
	// 计算 in/out 的 tokens
	var tokensIn, tokensOut uint64

	// in 优先使用缓存
	t.inputBufferMu.Lock()
	if len(t.inputBuffer) > 0 {
		tokensIn = EstimateTokensFromText(string(t.inputBuffer))
	} else {
		tokensIn = BytesToTokens(atomic.LoadUint64(&t.InputBytes))
	}
	t.inputBufferMu.Unlock()

	// out 优先使用缓存
	t.outputBufferMu.Lock()
	if len(t.outputBuffer) > 0 {
		tokensOut = EstimateTokensFromText(string(t.outputBuffer))
	} else {
		tokensOut = BytesToTokens(atomic.LoadUint64(&t.OutputBytes))
	}
	t.outputBufferMu.Unlock()

	// 计算新增 tokens（按方向）
	pendingIn := atomic.LoadUint64(&t.PendingTokensIn)
	pendingOut := atomic.LoadUint64(&t.PendingTokensOut)
	deltaIn := tokensIn - pendingIn
	deltaOut := tokensOut - pendingOut
	newTokens := deltaIn + deltaOut

	if newTokens >= FlushThreshold {
		if err := SendTokenUsageToStreamWithIOAndModel(t.customerToken, newTokens, deltaIn, deltaOut, t.model); err != nil {
			return fmt.Errorf("发送到 Stream 失败: %w", err)
		}
		log.Printf("[Token] 用户 %s 使用了 %d tokens (in=%d, out=%d, 已发送)",
			maskKey(t.customerToken), newTokens, deltaIn, deltaOut)

		// 更新 pending（按方向）
		atomic.StoreUint64(&t.PendingTokensIn, tokensIn)
		atomic.StoreUint64(&t.PendingTokensOut, tokensOut)
	}
	return nil
}

// Flush 强制刷新所有待处理的 tokens（在连接关闭时调用）
func (t *TrafficCounter) Flush() error {
	var tokensIn, tokensOut uint64

	t.inputBufferMu.Lock()
	inBufLen := len(t.inputBuffer)
	if inBufLen > 0 {
		tokensIn = EstimateTokensFromText(string(t.inputBuffer))
	} else {
		tokensIn = BytesToTokens(atomic.LoadUint64(&t.InputBytes))
	}
	t.inputBufferMu.Unlock()

	t.outputBufferMu.Lock()
	outBufLen := len(t.outputBuffer)
	if outBufLen > 0 {
		tokensOut = EstimateTokensFromText(string(t.outputBuffer))
	} else {
		tokensOut = BytesToTokens(atomic.LoadUint64(&t.OutputBytes))
	}
	t.outputBufferMu.Unlock()

	if inBufLen > 0 || outBufLen > 0 {
		log.Printf("[Token] 精确计算: %s in=%d (基于 %dB), out=%d (基于 %dB)",
			maskKey(t.customerToken), tokensIn, inBufLen, tokensOut, outBufLen)
	}

	pendingIn := atomic.LoadUint64(&t.PendingTokensIn)
	pendingOut := atomic.LoadUint64(&t.PendingTokensOut)
	deltaIn := tokensIn - pendingIn
	deltaOut := tokensOut - pendingOut
	remaining := deltaIn + deltaOut

	if remaining > 0 {
		if err := SendTokenUsageToStreamWithIOAndModel(t.customerToken, remaining, deltaIn, deltaOut, t.model); err != nil {
			return fmt.Errorf("刷新到 Stream 失败: %w", err)
		}
		log.Printf("[Token] 最后刷新: %s +%d (in=%d, out=%d)",
			maskKey(t.customerToken), remaining, deltaIn, deltaOut)

		// 记录 token 使用量到限流器
		ConsumeTokens(t.customerToken, "codex", int(remaining))
	}
	return nil
}

// SendTokenUsageToStreamWithIO 将 token 使用量发送到 Redis Stream，附带 in/out
func SendTokenUsageToStreamWithIO(customerToken string, tokens uint64, inTokens uint64, outTokens uint64) error {
	return SendTokenUsageToStreamWithIOAndModel(customerToken, tokens, inTokens, outTokens, nil)
}

// SendTokenUsageToStreamWithIOAndModel 将 token 使用量发送到 Redis Stream，附带 in/out 和 model（可选）
func SendTokenUsageToStreamWithIOAndModel(customerToken string, tokens uint64, inTokens uint64, outTokens uint64, model *string) error {
	client := redisstore.Client()
	if client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	ctx := redisstore.Context()

	data := map[string]interface{}{
		"customer_key":  customerToken,
		"tokens":        strconv.FormatUint(tokens, 10),
		"input_tokens":  strconv.FormatUint(inTokens, 10),
		"output_tokens": strconv.FormatUint(outTokens, 10),
	}
	if model != nil && *model != "" {
		data["model"] = *model
	}

	_, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		MaxLen: StreamMaxLen,
		Approx: true,
		Values: data,
	}).Result()
	if err != nil {
		return fmt.Errorf("写入 Stream 失败: %w", err)
	}
	return nil
}

// 保留原有的简单发送函数（兼容）
func SendTokenUsageToStream(customerToken string, tokens uint64) error {
	return SendTokenUsageToStreamWithIO(customerToken, tokens, tokens, 0)
}

// maskKey 隐藏 API key 的中间部分
func maskKey(key string) string {
	if len(key) <= 20 {
		return "***"
	}
	return key[:10] + "..." + key[len(key)-6:]
}

// ===== Codex: 官方 usage 优先（缺失回退估算）=====

type codexUsage struct {
	InputTokens  uint64 `json:"input_tokens"`
	OutputTokens uint64 `json:"output_tokens"`
	TotalTokens  uint64 `json:"total_tokens"`
}

type codexResponse struct {
	Usage *codexUsage `json:"usage"`
	Model string      `json:"model"`
}

type codexSSEEvent struct {
	Type     string         `json:"type"`
	Response *codexResponse `json:"response"`
}

type codexUsageEnvelope struct {
	Type     string         `json:"type"`
	Response *codexResponse `json:"response"`
}

func extractOfficialUsageFromJSON(payload []byte) (inTokens uint64, outTokens uint64, totalTokens uint64, model string, ok bool) {
	b := bytes.TrimSpace(payload)
	if len(b) == 0 || b[0] != '{' {
		return 0, 0, 0, "", false
	}

	// 记录使用的提取方式
	var extractMethod string
	var regexFields []string

	// 第一层：尝试 JSON 解析
	// 1) 直接响应体：{"usage":{...},"model":"..."}
	var direct codexResponse
	if err := json.Unmarshal(b, &direct); err == nil && direct.Usage != nil {
		u := direct.Usage
		inTokens = u.InputTokens
		outTokens = u.OutputTokens
		totalTokens = u.TotalTokens
		model = direct.Model
		extractMethod = "JSON解析(直接响应)"
	} else {
		// 2) 事件包裹：{"type":"...","response":{"usage":{...},"model":"..."}}
		var env codexUsageEnvelope
		if err := json.Unmarshal(b, &env); err != nil || env.Response == nil || env.Response.Usage == nil {
			return 0, 0, 0, "", false
		}
		u := env.Response.Usage
		inTokens = u.InputTokens
		outTokens = u.OutputTokens
		totalTokens = u.TotalTokens
		model = env.Response.Model
		extractMethod = "JSON解析(事件包裹)"
	}

	// 第二层：正则表达式兜底（当 JSON 解析失败或字段为空时）
	// 如果 input_tokens 为空，尝试正则提取
	if inTokens == 0 {
		if matches := inputTokensRegex.FindSubmatch(b); len(matches) > 1 {
			if val, err := strconv.ParseUint(string(matches[1]), 10, 64); err == nil {
				inTokens = val
				regexFields = append(regexFields, "input_tokens")
			}
		}
	}

	// 如果 output_tokens 为空，尝试正则提取
	if outTokens == 0 {
		if matches := outputTokensRegex.FindSubmatch(b); len(matches) > 1 {
			if val, err := strconv.ParseUint(string(matches[1]), 10, 64); err == nil {
				outTokens = val
				regexFields = append(regexFields, "output_tokens")
			}
		}
	}

	// 如果 total_tokens 为空，尝试正则提取
	if totalTokens == 0 {
		if matches := totalTokensRegex.FindSubmatch(b); len(matches) > 1 {
			if val, err := strconv.ParseUint(string(matches[1]), 10, 64); err == nil {
				totalTokens = val
				regexFields = append(regexFields, "total_tokens")
			}
		}
	}

	// 如果 model 为空，尝试正则提取
	if model == "" {
		if matches := gptModelRegex.FindSubmatch(b); len(matches) > 1 {
			model = string(matches[1])
			regexFields = append(regexFields, "model")
		}
	}

	// 第三层：计算兜底（补全缺失的 token 数据）
	calculatedFields := []string{}
	if totalTokens == 0 && (inTokens > 0 || outTokens > 0) {
		totalTokens = inTokens + outTokens
		calculatedFields = append(calculatedFields, "total_tokens=in+out")
	}
	if totalTokens > 0 && inTokens == 0 && outTokens == 0 {
		inTokens = totalTokens
		calculatedFields = append(calculatedFields, "input_tokens=total")
	}
	if totalTokens == 0 {
		return 0, 0, 0, "", false
	}

	// 输出日志：显示使用的提取方式
	logMsg := fmt.Sprintf("[提取方式] %s", extractMethod)
	if len(regexFields) > 0 {
		logMsg += fmt.Sprintf(" + 正则兜底[%v]", regexFields)
	}
	if len(calculatedFields) > 0 {
		logMsg += fmt.Sprintf(" + 计算补全[%v]", calculatedFields)
	}
	logMsg += fmt.Sprintf(" | 结果: in=%d, out=%d, total=%d, model=%s", inTokens, outTokens, totalTokens, model)
	log.Printf("%s", logMsg)

	return inTokens, outTokens, totalTokens, model, true
}

// CodexUsageSession 为单次请求/响应聚合用量（优先官方 usage）
type CodexUsageSession struct {
	customerToken string

	mu sync.Mutex

	inBytes  uint64
	outBytes uint64

	inBuffer  []byte
	outBuffer []byte

	sseRemainder  []byte
	officialIn    uint64
	officialOut   uint64
	officialTotal uint64
	officialModel string
	officialFound bool

	finalizeOnce sync.Once
	finalizeErr  error
}

func NewCodexUsageSession(customerToken string) *CodexUsageSession {
	// 记录用户访问（与通用计数器保持一致）
	trackingService := NewUserTrackingService()
	if err := trackingService.TrackUserAccess(customerToken); err != nil {
		log.Printf("[警告] 记录用户访问失败: %v", err)
	}

	return &CodexUsageSession{
		customerToken: customerToken,
		inBuffer:      make([]byte, 0, 8192),
		outBuffer:     make([]byte, 0, 8192),
		sseRemainder:  make([]byte, 0, 8192),
	}
}

func (s *CodexUsageSession) AddBytes(n int, dir string) {
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

func (s *CodexUsageSession) AppendContent(p []byte, dir string) {
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

func (s *CodexUsageSession) FeedResponseChunk(p []byte) {
	if len(p) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.officialFound {
		return
	}

	s.sseRemainder = append(s.sseRemainder, p...)
	// 防止非 SSE / 超大响应导致内存增长
	if len(s.sseRemainder) > 1024*1024 {
		s.sseRemainder = s.sseRemainder[len(s.sseRemainder)-1024*1024:]
	}

	for {
		sep := []byte("\n\n")
		sepLen := 2
		idx := bytes.Index(s.sseRemainder, sep)
		if idx < 0 {
			sep = []byte("\r\n\r\n")
			sepLen = 4
			idx = bytes.Index(s.sseRemainder, sep)
		}
		if idx < 0 {
			return
		}

		msg := s.sseRemainder[:idx]
		s.sseRemainder = s.sseRemainder[idx+sepLen:]

		var dataParts [][]byte
		for _, line := range bytes.Split(msg, []byte("\n")) {
			line = bytes.TrimSuffix(line, []byte("\r"))
			if !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}
			part := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
			if len(part) == 0 || bytes.Equal(part, []byte("[DONE]")) {
				continue
			}
			dataParts = append(dataParts, part)
		}
		if len(dataParts) == 0 {
			continue
		}

		payload := bytes.Join(dataParts, []byte("\n"))
		if len(payload) == 0 || payload[0] != '{' {
			continue
		}

		var ev codexSSEEvent
		if err := json.Unmarshal(payload, &ev); err != nil {
			continue
		}
		if ev.Type != "response.completed" || ev.Response == nil || ev.Response.Usage == nil {
			continue
		}

		u := ev.Response.Usage
		inTokens := u.InputTokens
		outTokens := u.OutputTokens
		totalTokens := u.TotalTokens
		if totalTokens == 0 {
			totalTokens = inTokens + outTokens
		}
		if totalTokens > 0 && inTokens == 0 && outTokens == 0 {
			inTokens = totalTokens
		}

		if totalTokens == 0 {
			continue
		}

		s.officialIn = inTokens
		s.officialOut = outTokens
		s.officialTotal = totalTokens
		s.officialModel = ev.Response.Model
		s.officialFound = true
		return
	}
}

func (s *CodexUsageSession) FinalizeAndSend() error {
	s.finalizeOnce.Do(func() {
		s.finalizeErr = s.finalizeAndSend()
	})
	return s.finalizeErr
}

func (s *CodexUsageSession) finalizeAndSend() error {
	s.mu.Lock()
	customerToken := s.customerToken
	officialFound := s.officialFound
	officialIn := s.officialIn
	officialOut := s.officialOut
	officialTotal := s.officialTotal
	officialModel := s.officialModel

	inBytes := s.inBytes
	outBytes := s.outBytes
	inBuf := append([]byte(nil), s.inBuffer...)
	outBuf := append([]byte(nil), s.outBuffer...)
	s.mu.Unlock()

	if customerToken == "" {
		return nil
	}

	if officialFound {
		var model *string
		if officialModel != "" {
			model = &officialModel
		}
		err := SendTokenUsageToStreamWithIOAndModel(customerToken, officialTotal, officialIn, officialOut, model)
		if err == nil {
			// 记录 token 使用量到限流器
			ConsumeTokens(customerToken, "codex", int(officialTotal))
		}
		return err
	}

	// 官方 usage 没拿到（SSE 解析失败/非 SSE）：再尝试从整段响应体 JSON 直接提取 usage
	if inT, outT, totalT, m, ok := extractOfficialUsageFromJSON(outBuf); ok {
		var model *string
		if m != "" {
			model = &m
		}
		err := SendTokenUsageToStreamWithIOAndModel(customerToken, totalT, inT, outT, model)
		if err == nil {
			// 记录 token 使用量到限流器
			ConsumeTokens(customerToken, "codex", int(totalT))
		}
		return err
	}

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

	// 使用 officialModel（如果有的话）
	var model *string
	if officialModel != "" {
		model = &officialModel
	}

	err := SendTokenUsageToStreamWithIOAndModel(customerToken, total, inTokens, outTokens, model)
	if err == nil {
		// 记录 token 使用量到限流器
		ConsumeTokens(customerToken, "codex", int(total))
	}
	return err
}

// CountingReadCloser 包装 io.ReadCloser，统计流量（带方向）
type CountingReadCloser struct {
	R         io.ReadCloser   // 底层 Reader
	Counter   *TrafficCounter // 流量计数器
	Direction string          // "in" 或 "out"
}

// NewCountingReadCloser 创建计数 Reader（第三个参数表示方向）
func NewCountingReadCloser(r io.ReadCloser, counter *TrafficCounter, dir string) io.ReadCloser {
	return &CountingReadCloser{
		R:         r,
		Counter:   counter,
		Direction: dir,
	}
}

// Read 实现 io.Reader
func (c *CountingReadCloser) Read(p []byte) (int, error) {
	n, err := c.R.Read(p)
	if n > 0 {
		// 增加字节数
		c.Counter.AddBytes(int64(n), c.Direction)
		// 追加到缓存（用于精确计算）
		c.Counter.AppendContent(p[:n], c.Direction)
		// 异步检查是否需要刷新到 Stream
		go func() {
			if flushErr := c.Counter.CheckAndFlushToStream(); flushErr != nil {
				log.Printf("[警告] 刷新 tokens 到 Stream 失败: %v", flushErr)
			}
		}()
	}
	return n, err
}

// Close 实现 io.Closer
func (c *CountingReadCloser) Close() error {
	// 关闭时强制刷新所有待处理的 tokens
	if c.Counter != nil {
		if err := c.Counter.Flush(); err != nil {
			log.Printf("[警告] 关闭时刷新 tokens 失败: %v", err)
		}
	}
	return c.R.Close()
}

// codexCountingReadCloser 用于 Codex：抓取官方 usage，缺失回退估算
type codexCountingReadCloser struct {
	R         io.ReadCloser
	Session   *CodexUsageSession
	Direction string // "in" 或 "out"
}

func NewCodexCountingReadCloser(r io.ReadCloser, session *CodexUsageSession, dir string) io.ReadCloser {
	return &codexCountingReadCloser{
		R:         r,
		Session:   session,
		Direction: dir,
	}
}

func (c *codexCountingReadCloser) Read(p []byte) (int, error) {
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

func (c *codexCountingReadCloser) Close() error {
	if c.Session != nil && c.Direction == "out" {
		if err := c.Session.FinalizeAndSend(); err != nil {
			log.Printf("[警告] Codex 最后刷新失败: %v", err)
		}
	}
	return c.R.Close()
}

// ===== 消费者端：从 Stream 读取并写入 MySQL =====

// TokenUsageConsumer Stream 消费者
type TokenUsageConsumer struct {
	GroupName    string
	ConsumerName string
}

// NewTokenUsageConsumer 创建消费者
func NewTokenUsageConsumer(groupName, consumerName string) *TokenUsageConsumer {
	return &TokenUsageConsumer{
		GroupName:    groupName,
		ConsumerName: consumerName,
	}
}

// InitConsumerGroup 初始化消费者组
func (c *TokenUsageConsumer) InitConsumerGroup() error {
	client := redisstore.Client()
	if client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	ctx := redisstore.Context()

	// 创建消费者组
	err := client.XGroupCreateMkStream(ctx, StreamKey, c.GroupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("创建消费者组失败: %w", err)
	}

	log.Printf("✓ 消费者组 %s 初始化成功", c.GroupName)
	return nil
}

// StartConsuming 开始消费 Stream 并写入 MySQL
func (c *TokenUsageConsumer) StartConsuming(ctx context.Context) {
	client := redisstore.Client()
	if client == nil {
		log.Printf("Redis 客户端未初始化")
		return
	}

	log.Printf("消费者 %s 启动...", c.ConsumerName)

	for {
		select {
		case <-ctx.Done():
			log.Printf("消费者 %s 停止", c.ConsumerName)
			return
		default:
			// 读取消息
			streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.GroupName,
				Consumer: c.ConsumerName,
				Streams:  []string{StreamKey, ">"},
				Count:    10,
				Block:    1 * time.Second,
			}).Result()

			if err != nil {
				if err.Error() != "redis: nil" {
					log.Printf("读取 Stream 失败: %v", err)
				}
				// 出错时（包括 client 已关闭），尝试获取最新客户端
				client = redisstore.Client()
				time.Sleep(1 * time.Second)
				continue
			}

			// 处理消息
			for _, stream := range streams {
				for _, message := range stream.Messages {
					c.processMessage(message)
					// ACK 消息
					client.XAck(ctx, StreamKey, c.GroupName, message.ID)
				}
			}
		}
	}
}

// processMessage 处理单条消息
func (c *TokenUsageConsumer) processMessage(msg redis.XMessage) {
	// 解析消息
	customerToken, _ := msg.Values["customer_key"].(string)
	tokensStr, _ := msg.Values["tokens"].(string)
	inStr, _ := msg.Values["input_tokens"].(string)
	outStr, _ := msg.Values["output_tokens"].(string)
	modelStr, _ := msg.Values["model"].(string)

	var inTokens, outTokens uint64
	if inStr != "" {
		if v, err := strconv.ParseUint(inStr, 10, 64); err == nil {
			inTokens = v
		}
	}
	if outStr != "" {
		if v, err := strconv.ParseUint(outStr, 10, 64); err == nil {
			outTokens = v
		}
	}

	// 总 tokens（若未提供 in/out，则回退使用 tokens 字段）
	var tokens uint64
	if inTokens > 0 || outTokens > 0 {
		tokens = inTokens + outTokens
	} else {
		if v, err := strconv.ParseUint(tokensStr, 10, 64); err == nil {
			tokens = v
		}
	}

	var model *string
	if modelStr != "" {
		model = &modelStr
	}
	// 可选的原始消息体（当 tokens < 500 或 > 5000 会入库）
	originStr, _ := msg.Values["origin_message"].(string)
	var origin *string
	if originStr != "" {
		origin = &originStr
	}

	log.Printf("[消费] 处理消息: 用户=%s, tokens=%d (in=%d, out=%d, model=%s)",
		maskKey(customerToken), tokens, inTokens, outTokens, modelStr)

	// 写入 MySQL（按方向近似计费）
	if err := c.writeToMySQL(customerToken, tokens, inTokens, outTokens, model, "", origin); err != nil {
		log.Printf("✗ 写入 MySQL 失败: %v", err)
	} else {
		log.Printf("✓ 写入 MySQL 成功: %s +%d tokens (in=%d, out=%d, model=%s)", maskKey(customerToken), tokens, inTokens, outTokens, modelStr)
	}
}

// writeToMySQL 写入 MySQL 使用 UsageService
func (c *TokenUsageConsumer) writeToMySQL(customerToken string, tokens uint64, inTokens uint64, outTokens uint64, model *string, hash string, originMessage *string) error {
	// 2. 按 account -> product.account_type -> pricing 计算消费
	pricingService := NewPricingService()
	consume, pricingRule, accountType, calcErr := pricingService.CalculateConsumeByCustomerToken(customerToken, tokens, inTokens, outTokens)
	if calcErr != nil {
		log.Printf("[计价] 查询 pricing 失败，回退默认值: token=%s, error=%v", maskKey(customerToken), calcErr)
		if inTokens > 0 || outTokens > 0 {
			consume = TokensToConsumeByIO(inTokens, outTokens)
		} else {
			consume = TokensToConsume(tokens)
		}
		pricingRule = nil
	}

	// 3. 使用 UsageService 记录使用量
	usageService := NewUsageService()
	if err := usageService.RecordTokenUsage(customerToken, tokens, consume, model, originMessage); err != nil {
		return fmt.Errorf("failed to record token usage: %w", err)
	}

	unit := DefaultPricingUnit
	inUnitPrice := DefaultInTokenUnitPrice
	outUnitPrice := DefaultOutTokenUnitPrice
	tokenUnit := DefaultTokenUnit
	if pricingRule != nil {
		if pricingRule.Unit != "" {
			unit = pricingRule.Unit
		}
		inUnitPrice = pricingRule.InTokenUnitPrice
		outUnitPrice = pricingRule.OutTokenUnitPrice
		tokenUnit = pricingRule.TokenUnit
		if tokenUnit <= 0 {
			tokenUnit = DefaultTokenUnit
		}
	}
	log.Printf("[MySQL] 写入成功: token=%s, tokens=%d, consume=%.6f (%s), account_type=%s, in=%d, out=%d, in_token_unit_price=%.8f, out_token_unit_price=%.8f, token_unit=%d, hash=%s",
		maskKey(customerToken), tokens, consume, unit, accountType, inTokens, outTokens, inUnitPrice, outUnitPrice, tokenUnit, hash)
	return nil
}

// TokensToConsume 将 tokens 数量转换为消费金额（基于默认 pricing）
func TokensToConsume(tokens uint64) float64 {
	return CalculateConsumeByPricing(tokens, 0, 0, DefaultPricingForAccountType(""))
}

// TokensToConsumeByIO 按输入/输出分别计价（基于默认 pricing）
func TokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	return CalculateConsumeByPricing(0, inTokens, outTokens, DefaultPricingForAccountType(""))
}

// GetUsageFromMySQL 从 MySQL 查询指定日期的使用量
// 返回消费金额（单位：USD）
func GetUsageFromMySQL(customerToken string, dates []string) (float64, error) {
	usageService := NewUsageService()

	// 如果 dates 为空，默认查询今天
	if len(dates) == 0 {
		totalConsume, err := usageService.GetTodayUsageByToken(customerToken)
		if err != nil {
			return 0, fmt.Errorf("failed to get today's usage: %w", err)
		}
		return totalConsume, nil
	}

	// 查询指定日期的使用量
	totalConsume, err := usageService.GetUsageByDates(customerToken, dates)
	if err != nil {
		return 0, fmt.Errorf("failed to get usage by dates: %w", err)
	}
	return totalConsume, nil
}

// GetUsageFromRedis 从 MySQL 查询使用量（为了保持函数名向后兼容）
// 返回消费金额（单位：USD）
func GetUsageFromRedis(customerToken string, dates []string) (float64, error) {
	return GetUsageFromMySQL(customerToken, dates)
}

// SendTokenUsageToStreamWithIOAndModelAndMessage 将 token 使用量发送到 Redis Stream，附带 in/out、model（可选）与原始消息（可选）
// originMessage 仅在 tokens < 500 或 > 5000 时随消息发送，避免 Stream 负载过大；与 UsageService 的入库策略保持一致。
func SendTokenUsageToStreamWithIOAndModelAndMessage(customerToken string, tokens uint64, inTokens uint64, outTokens uint64, model *string, originMessage *string) error {
	client := redisstore.Client()
	if client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	ctx := redisstore.Context()

	data := map[string]interface{}{
		"customer_key":  customerToken,
		"tokens":        strconv.FormatUint(tokens, 10),
		"input_tokens":  strconv.FormatUint(inTokens, 10),
		"output_tokens": strconv.FormatUint(outTokens, 10),
	}
	if model != nil && *model != "" {
		data["model"] = *model
	}
	// 与 usage_service 的保存策略保持一致，降低 Stream 压力
	if originMessage != nil && (tokens < 500 || tokens > 5000) {
		msg := *originMessage
		// 保护性截断，避免过长消息撑爆 Stream；UTF-8 字节层面限制
		const maxBytes = 8192
		if len(msg) > maxBytes {
			msg = msg[:maxBytes]
		}
		data["origin_message"] = msg
	}

	_, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		MaxLen: StreamMaxLen,
		Approx: true,
		Values: data,
	}).Result()
	if err != nil {
		return fmt.Errorf("写入 Stream 失败: %w", err)
	}
	return nil
}
