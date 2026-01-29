package service

import (
	"codex-relay/internal/redis"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// UserRateLimiter 用户级 token 限流器配置
type UserRateLimiter struct {
	maxTokens      int           // 最大 token 数量
	windowDuration time.Duration // 时间窗口
}

var (
	defaultLimiter = &UserRateLimiter{
		maxTokens:      50000,
		windowDuration: 5 * time.Minute,
	}

	// 每个用户一个读写锁
	userLocks   = make(map[string]*sync.RWMutex)
	userLocksMu sync.Mutex
)

// getUserLock 获取或创建用户的读写锁
func getUserLock(key string) *sync.RWMutex {
	userLocksMu.Lock()
	defer userLocksMu.Unlock()

	if lock, exists := userLocks[key]; exists {
		return lock
	}

	lock := &sync.RWMutex{}
	userLocks[key] = lock
	return lock
}

// ConsumeTokens 消费 tokens（记录使用量）
func ConsumeTokens(token, sourceType string, tokens int) error {
	key := fmt.Sprintf("user:ratelimit:%s:%s", sourceType, token)
	lock := getUserLock(key)
	lock.Lock()
	defer lock.Unlock()

	rdb := redis.Client()
	if rdb == nil {
		log.Printf("[UserRateLimit] Redis 不可用，跳过记录")
		return nil
	}

	ctx := redis.Context()
	now := time.Now().UnixMilli()

	// 使用时间戳:tokens 作为 member，时间戳作为 score
	member := goredis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d:%d", now, tokens),
	}

	err := rdb.ZAdd(ctx, key, member).Err()
	if err != nil {
		log.Printf("[UserRateLimit] 记录 token 使用失败: %v", err)
		return err
	}

	// 设置 key 过期时间为 5 分钟
	// 不活跃用户的 key 会自动过期删除
	rdb.Expire(ctx, key, 5*time.Minute)

	log.Printf("[UserRateLimit] 已记录用户 %s 使用 %d tokens", token, tokens)
	return nil
}

// IsTokenLimitExceeded 检查是否超过限流
func IsTokenLimitExceeded(token, sourceType string) (bool, error) {
	key := fmt.Sprintf("user:ratelimit:%s:%s", sourceType, token)
	lock := getUserLock(key)
	lock.RLock()
	defer lock.RUnlock()

	rdb := redis.Client()
	if rdb == nil {
		log.Printf("[UserRateLimit] Redis 不可用，跳过限流检查")
		return false, nil
	}

	ctx := redis.Context()
	now := time.Now().UnixMilli()
	windowStart := now - defaultLimiter.windowDuration.Milliseconds()

	// 1. 删除时间窗口之外的旧记录
	_, err := rdb.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart)).Result()
	if err != nil {
		log.Printf("[UserRateLimit] 清理旧记录失败: %v", err)
		return false, nil
	}

	// 2. 统计当前窗口内的 token 总数
	members, err := rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		log.Printf("[UserRateLimit] 查询 token 使用量失败: %v", err)
		return false, nil
	}

	totalTokens := 0
	for _, member := range members {
		memberStr := member.Member.(string)
		parts := strings.Split(memberStr, ":")
		if len(parts) == 2 {
			tokens, _ := strconv.Atoi(parts[1])
			totalTokens += tokens
		}
	}

	// 3. 检查是否超限
	if totalTokens >= defaultLimiter.maxTokens {
		log.Printf("[UserRateLimit] 用户 %s 已超过限流: %d/%d tokens (窗口: %v)",
			token, totalTokens, defaultLimiter.maxTokens, defaultLimiter.windowDuration)
		return true, nil
	}

	log.Printf("[UserRateLimit] 用户 %s 当前使用: %d/%d tokens", token, totalTokens, defaultLimiter.maxTokens)
	return false, nil
}

// CleanExpiredRecords 清理过期记录（可选）
// 注意：由于 key 设置了 5 分钟过期时间，Redis 会自动清理，此函数可选
func CleanExpiredRecords() {
	log.Printf("[UserRateLimit] Redis key 已设置 5 分钟自动过期，无需手动清理")
}
