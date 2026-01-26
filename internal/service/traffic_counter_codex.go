package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"io"

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

	// 按方向缓存内容（用于更精确的 token 估算）
	inputBuffer  []byte
	outputBuffer []byte

	inputBufferMu  sync.Mutex
	outputBufferMu sync.Mutex
}

// NewTrafficCounter 初始化流量计数器
func NewTrafficCounter(customerToken string) *TrafficCounter {
	return &TrafficCounter{
		StartTime:     time.Now(),
		customerToken: customerToken,
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
		if err := SendTokenUsageToStreamWithIO(t.customerToken, newTokens, deltaIn, deltaOut); err != nil {
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
		if err := SendTokenUsageToStreamWithIO(t.customerToken, remaining, deltaIn, deltaOut); err != nil {
			return fmt.Errorf("刷新到 Stream 失败: %w", err)
		}
		log.Printf("[Token] 最后刷新: %s +%d (in=%d, out=%d)",
			maskKey(t.customerToken), remaining, deltaIn, deltaOut)
	}
	return nil
}

// SendTokenUsageToStreamWithIO 将 token 使用量发送到 Redis Stream，附带 in/out
func SendTokenUsageToStreamWithIO(customerToken string, tokens uint64, inTokens uint64, outTokens uint64) error {
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

	log.Printf("[消费] 处理消息: 用户=%s, tokens=%d (in=%d, out=%d)",
		maskKey(customerToken), tokens, inTokens, outTokens)

	// 写入 MySQL（按方向近似计费）
	if err := c.writeToMySQL(customerToken, tokens, inTokens, outTokens, ""); err != nil {
		log.Printf("✗ 写入 MySQL 失败: %v", err)
	} else {
		log.Printf("✓ 写入 MySQL 成功: %s +%d tokens (in=%d, out=%d)", maskKey(customerToken), tokens, inTokens, outTokens)
	}
}

// writeToMySQL 写入 MySQL 使用 UsageService
func (c *TokenUsageConsumer) writeToMySQL(customerToken string, tokens uint64, inTokens uint64, outTokens uint64, hash string) error {
	// 2. 将 tokens 转换为消费金额（方向区分）
	// 按官网每百万 tokens 单价近似计价，单位 USD
	// 实际项目中应根据产品定价/账户类型计算
	var consume float64
	if inTokens > 0 || outTokens > 0 {
		consume = TokensToConsumeByIO(inTokens, outTokens)
	} else {
		consume = TokensToConsume(tokens)
	}

	// 3. 使用 UsageService 记录使用量
	usageService := NewUsageService()
	if err := usageService.RecordTokenUsage(customerToken, tokens, consume); err != nil {
		return fmt.Errorf("failed to record token usage: %w", err)
	}

	log.Printf("[MySQL] 写入成功: token=%s, tokens=%d, consume=%.6f (USD), in=%d, out=%d, hash=%s",
		maskKey(customerToken), tokens, consume, inTokens, outTokens, hash)
	return nil
}

// TokensToConsume 将 tokens 数量转换为消费金额（单位：USD）
// 近似算法：按 Batch API 单价（每 100 万 tokens）估算：
// - 输入 $1.75 / 1M
// - 命中缓存的输入 $0.175 / 1M
// - 输出 $14.00 / 1M
// 假设总 tokens 中输入:输出≈70%:30%，其中输入的 10% 为缓存命中。
// 注意：这里只做近似估算，可按业务需求调整占比或按模型分类计价。
func TokensToConsume(tokens uint64) float64 {
	const (
		pricePerMInputUSD       = 1.75
		pricePerMCachedInputUSD = 0.175
		pricePerMOutputUSD      = 14.00

		inputRatio     = 0.70 // 总 tokens 中视作输入的占比
		outputRatio    = 0.30 // 总 tokens 中视作输出的占比（=1-inputRatio）
		cachedHitRatio = 0.10 // 输入 tokens 中命中缓存的占比
	)

	if tokens == 0 {
		return 0
	}

	// 有效的每 token 美元价格
	inputPerTokenUSD := ((1.0-cachedHitRatio)*pricePerMInputUSD + cachedHitRatio*pricePerMCachedInputUSD) / 1_000_000.0
	outputPerTokenUSD := pricePerMOutputUSD / 1_000_000.0
	blendedPerTokenUSD := inputRatio*inputPerTokenUSD + outputRatio*outputPerTokenUSD

	return float64(tokens) * blendedPerTokenUSD
}

// TokensToConsumeByIO 按输入/输出分别计价（单位：USD）
func TokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	const (
		pricePerMInputUSD       = 1.75
		pricePerMCachedInputUSD = 0.175
		pricePerMOutputUSD      = 14.00
		cachedHitRatio          = 0.10
	)
	if inTokens == 0 && outTokens == 0 {
		return 0
	}
	inputPerTokenUSD := ((1.0-cachedHitRatio)*pricePerMInputUSD + cachedHitRatio*pricePerMCachedInputUSD) / 1_000_000.0
	outputPerTokenUSD := pricePerMOutputUSD / 1_000_000.0
	return float64(inTokens)*inputPerTokenUSD + float64(outTokens)*outputPerTokenUSD
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
