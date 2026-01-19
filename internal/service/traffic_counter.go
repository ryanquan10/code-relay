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

// TrafficCounter 流量计数器（简化版，增强并发安全）
type TrafficCounter struct {
	TotalBytes    uint64 // 总字节数（上行+下行）
	PendingTokens uint64 // 待刷新的 tokens
	StartTime     time.Time
	customerToken string
	buffer        []byte     // 内容缓存（用于精确计算）
	bufferMu      sync.Mutex // buffer 并发保护
}

// NewTrafficCounter 初始化流量计数器
func NewTrafficCounter(customerToken string) *TrafficCounter {
	return &TrafficCounter{
		StartTime:     time.Now(),
		customerToken: customerToken,
		buffer:        make([]byte, 0, 8192),
	}
}

// AddBytes 增加字节数（上行或下行都调用这个）
func (t *TrafficCounter) AddBytes(n int64) {
	atomic.AddUint64(&t.TotalBytes, uint64(n))
}

// GetTotalBytes 获取总字节数
func (t *TrafficCounter) GetTotalBytes() uint64 {
	return atomic.LoadUint64(&t.TotalBytes)
}

// AppendContent 追加内容到缓存（用于精确计算 token）
// 使用 mutex 保护并发安全
func (t *TrafficCounter) AppendContent(data []byte) {
	t.bufferMu.Lock()
	defer t.bufferMu.Unlock()

	if len(t.buffer) < 32768 { // 最多缓存 32KB
		t.buffer = append(t.buffer, data...)
	}
}

// Reset 重置计数器
func (t *TrafficCounter) Reset() {
	atomic.StoreUint64(&t.TotalBytes, 0)
	atomic.StoreUint64(&t.PendingTokens, 0)

	t.bufferMu.Lock()
	t.buffer = nil
	t.bufferMu.Unlock()

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

// CheckAndFlushToStream 检查是否需要将 tokens 写入 Redis Stream
func (t *TrafficCounter) CheckAndFlushToStream() error {
	totalBytes := t.GetTotalBytes()

	// 优先使用缓存内容精确计算（需要加锁读取 buffer）
	var totalTokens uint64
	t.bufferMu.Lock()
	if len(t.buffer) > 0 {
		totalTokens = EstimateTokensFromText(string(t.buffer))
	} else {
		totalTokens = BytesToTokens(totalBytes)
	}
	t.bufferMu.Unlock()

	// 计算新增的 tokens
	currentPending := atomic.LoadUint64(&t.PendingTokens)
	newTokens := totalTokens - currentPending

	if newTokens >= FlushThreshold {
		// 发送到 Redis Stream
		if err := SendTokenUsageToStream(t.customerToken, newTokens); err != nil {
			return fmt.Errorf("发送到 Stream 失败: %w", err)
		}

		log.Printf("[Token] 用户 %s 使用了 %d tokens (已发送到 Stream)",
			maskKey(t.customerToken), newTokens)

		// 更新 pending tokens
		atomic.StoreUint64(&t.PendingTokens, totalTokens)
	}

	return nil
}

// Flush 强制刷新所有待处理的 tokens（在连接关闭时调用）
func (t *TrafficCounter) Flush() error {
	totalBytes := t.GetTotalBytes()
	currentPending := atomic.LoadUint64(&t.PendingTokens)

	// 使用缓存内容精确计算（需要加锁读取 buffer）
	var totalTokens uint64
	t.bufferMu.Lock()
	bufferLen := len(t.buffer)
	if bufferLen > 0 {
		totalTokens = EstimateTokensFromText(string(t.buffer))
	} else {
		totalTokens = BytesToTokens(totalBytes)
	}
	t.bufferMu.Unlock()

	if bufferLen > 0 {
		log.Printf("[Token] 精确计算: %s 使用了 %d tokens (基于 %d 字节内容分析)",
			maskKey(t.customerToken), totalTokens, bufferLen)
	}

	remainingTokens := totalTokens - currentPending

	if remainingTokens > 0 {
		// 发送剩余的 tokens
		if err := SendTokenUsageToStream(t.customerToken, remainingTokens); err != nil {
			return fmt.Errorf("刷新到 Stream 失败: %w", err)
		}

		log.Printf("[Token] 最后刷新: %s 使用了 %d tokens",
			maskKey(t.customerToken), remainingTokens)
	}

	return nil
}

// SendTokenUsageToStream 将 token 使用量发送到 Redis Stream（简化版）
// 仅传输核心字段：customer_key、tokens
func SendTokenUsageToStream(customerToken string, tokens uint64) error {
	client := redisstore.Client()
	if client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	ctx := redisstore.Context()

	// 构建 map，仅包含计数数据
	data := map[string]interface{}{
		"customer_key": customerToken,
		"tokens":       strconv.FormatUint(tokens, 10), // 仅tokens计数作为字符串
	}

	// 发送到 Stream（限制最大长度）
	_, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		MaxLen: StreamMaxLen, // 最多保留 10000 条
		Approx: true,         // 使用近似裁剪（性能更好）
		Values: data,
	}).Result()

	if err != nil {
		return fmt.Errorf("写入 Stream 失败: %w", err)
	}

	return nil
}

// maskKey 隐藏 API key 的中间部分
func maskKey(key string) string {
	if len(key) <= 20 {
		return "***"
	}
	return key[:10] + "..." + key[len(key)-6:]
}

// CountingReadCloser 包装 io.ReadCloser，统计流量
type CountingReadCloser struct {
	R       io.ReadCloser   // 底层 Reader
	Counter *TrafficCounter // 流量计数器
}

// NewCountingReadCloser 创建计数 Reader
func NewCountingReadCloser(r io.ReadCloser, counter *TrafficCounter) io.ReadCloser {
	return &CountingReadCloser{
		R:       r,
		Counter: counter,
	}
}

// Read 实现 io.Reader
func (c *CountingReadCloser) Read(p []byte) (int, error) {
	n, err := c.R.Read(p)
	if n > 0 {
		// 增加字节数
		c.Counter.AddBytes(int64(n))

		// 追加到缓存（用于精确计算）
		c.Counter.AppendContent(p[:n])

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

	tokens, err := strconv.ParseUint(tokensStr, 10, 64)
	if err != nil {
		log.Printf("✗ 解析 tokens 失败: %v", err)
		return
	}

	log.Printf("[消费] 处理消息: 用户=%s, tokens=%d",
		maskKey(customerToken), tokens)

	// TODO: 写入 MySQL
	if err := c.writeToMySQL(customerToken, tokens, ""); err != nil {
		log.Printf("✗ 写入 MySQL 失败: %v", err)
	} else {
		log.Printf("✓ 写入 MySQL 成功: %s +%d tokens", maskKey(customerToken), tokens)
	}
}

// writeToMySQL 写入 MySQL 使用 UsageService
func (c *TokenUsageConsumer) writeToMySQL(customerToken string, tokens uint64, hash string) error {
	// 2. 将 tokens 转换为消费金额
	// 这里使用一个简单的转换率：每 1000 tokens = 0.01 元
	// 实际项目中应该根据产品定价和账户类型计算
	consume := TokensToConsume(tokens)

	// 3. 使用 UsageService 记录使用量（假设 RecordTokenUsage 支持 hash 参数）
	usageService := NewUsageService()
	if err := usageService.RecordTokenUsage(customerToken, tokens, consume); err != nil {
		return fmt.Errorf("failed to record token usage: %w", err)
	}

	log.Printf("[MySQL] 写入成功: token=%s, tokens=%d, consume=%.4f, hash=%s",
		maskKey(customerToken), tokens, consume, hash)

	return nil
}

// TokensToConsume 将 tokens 数量转换为消费金额
// 转换率：每 1000 tokens = 0.01 元（即每个 token = 0.00001 元）
// 实际项目中应该根据产品定价和账户类型计算
func TokensToConsume(tokens uint64) float64 {
	return float64(tokens) * 0.00001
}

// GetUsageFromMySQL 从 MySQL 查询指定日期的使用量
// 返回消费金额（单位：元）
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
// 返回消费金额（单位：元）
func GetUsageFromRedis(customerToken string, dates []string) (float64, error) {
	return GetUsageFromMySQL(customerToken, dates)
}
