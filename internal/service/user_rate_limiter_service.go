package service

import (
	"codex-relay/internal/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	defaultTokenLimit = map[string]int64{
		"codex":  8_517_547,
		"claude": 13_333_333,
		// 未来新增模型可以直接在这里加一行
		// "gemini": 20_000_000,
	}

	// 每个用户一个读写锁
	userLocks   = make(map[string]*sync.RWMutex)
	userLocksMu sync.Mutex
)

const rollingWindow = 24 * time.Hour

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

func validateSourceType(sourceType string) error {
	if _, ok := defaultTokenLimit[sourceType]; ok {
		return nil
	}
	return fmt.Errorf("invalid sourceType %q (must be 'codex' or 'claude')", sourceType)
}

func usageKey(token, sourceType string, now time.Time) string {
	_ = now
	return fmt.Sprintf("user:ratelimit:%s:%s", sourceType, token)
}

func limitKey(sourceType string) string {
	return fmt.Sprintf("tokenLimit:%s", sourceType)
}

func defaultLimitFor(sourceType string) (int64, error) {
	limit, ok := defaultTokenLimit[sourceType]
	if !ok {
		return 0, fmt.Errorf("unknown source type: %q", sourceType)
	}
	return limit, nil
}

func getTokenLimit(ctx context.Context, rdb *goredis.Client, sourceType string) (int64, error) {
	def, err := defaultLimitFor(sourceType)
	if err != nil {
		return 0, err
	}
	if rdb == nil {
		return def, nil
	}

	// 允许通过 Redis 手动调节：tokenLimit:codex / tokenLimit:claude
	// 如果 key 不存在，写入默认值（不设置过期）
	_ = rdb.SetNX(ctx, limitKey(sourceType), def, 0).Err()

	val, err := rdb.Get(ctx, limitKey(sourceType)).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return def, nil
		}
		return def, err
	}
	parsed, parseErr := strconv.ParseInt(val, 10, 64)
	if parseErr != nil || parsed <= 0 {
		return def, nil
	}
	return parsed, nil
}

// ConsumeTokens 消费 tokens（记录使用量）
func ConsumeTokens(token, sourceType string, tokens int) error {
	rdb := redis.Client()
	if rdb == nil {
		return nil
	}
	if err := validateSourceType(sourceType); err != nil {
		return err
	}
	ctx := redis.Context()
	now := time.Now()
	key := usageKey(token, sourceType, now)

	lock := getUserLock(key)
	lock.Lock()
	defer lock.Unlock()

	ts := now.UnixMilli()
	member := fmt.Sprintf("%d:%d", ts, tokens)

	if err := rdb.ZAdd(ctx, key, goredis.Z{Score: float64(ts), Member: member}).Err(); err != nil {
		return err
	}
	// 设置 key 过期，滚动窗口 + 1h 缓冲
	_ = rdb.Expire(ctx, key, rollingWindow+time.Hour).Err()
	return nil
}

// IsTokenLimitExceeded 检查是否超过限流
func IsTokenLimitExceeded(token, sourceType string) (bool, error) {
	rdb := redis.Client()
	if rdb == nil {
		log.Printf("[UserRateLimit] Redis 不可用，跳过限流检查")
		return false, nil
	}

	ctx := redis.Context()
	now := time.Now()

	// 兼容旧调用：sourceType 为空时，检查 codex/claude 两种类型，只要任意一种超限就拦截
	if sourceType == "" {
		exceededCodex, err := isTokenLimitExceededForType(ctx, rdb, token, "codex", now)
		if err != nil {
			return false, err
		}
		exceededClaude, err := isTokenLimitExceededForType(ctx, rdb, token, "claude", now)
		if err != nil {
			return false, err
		}
		return exceededCodex || exceededClaude, nil
	}
	if err := validateSourceType(sourceType); err != nil {
		return false, err
	}
	return isTokenLimitExceededForType(ctx, rdb, token, sourceType, now)
}

func isTokenLimitExceededForType(ctx context.Context, rdb *goredis.Client, token, sourceType string, now time.Time) (bool, error) {
	key := usageKey(token, sourceType, now)
	lock := getUserLock(key)
	lock.RLock()
	defer lock.RUnlock()

	limit, err := getTokenLimit(ctx, rdb, sourceType)
	if err != nil {
		log.Printf("[UserRateLimit] 获取限额失败: %v", err)
		return false, nil
	}

	nowMs := now.UnixMilli()
	windowStartMs := now.Add(-rollingWindow).UnixMilli()

	// 1) 清理旧记录
	_, err = rdb.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStartMs)).Result()
	if err != nil {
		log.Printf("[UserRateLimit] 清理旧记录失败: %v", err)
		return false, nil
	}

	// 2) 统计当前窗口内 token 总数
	members, err := rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			log.Printf("[UserRateLimit] 用户 %s (%s) 24h 使用: 0/%d tokens", token, sourceType, limit)
			return false, nil
		}
		log.Printf("[UserRateLimit] 查询 token 使用量失败: %v", err)
		return false, nil
	}

	var used int64
	for _, m := range members {
		memberStr, ok := m.Member.(string)
		if !ok {
			continue
		}
		parts := strings.Split(memberStr, ":")
		if len(parts) != 2 {
			continue
		}
		ts, tsErr := strconv.ParseInt(parts[0], 10, 64)
		if tsErr != nil || ts < windowStartMs || ts > nowMs {
			continue
		}
		tok, tokErr := strconv.ParseInt(parts[1], 10, 64)
		if tokErr != nil {
			continue
		}
		used += tok
	}

	if used >= limit {
		log.Printf("[UserRateLimit] 用户 %s (%s) 已超过限流: %d/%d tokens (24h 滚动窗口)",
			token, sourceType, used, limit)
		return true, nil
	}

	log.Printf("[UserRateLimit] 用户 %s (%s) 24h 使用: %d/%d tokens", token, sourceType, used, limit)
	return false, nil
}

// CleanExpiredRecords 清理过期记录（可选）
// 注意：key 已设置滚动窗口过期（+缓冲），Redis 会自动清理，此函数可选
func CleanExpiredRecords() {
	log.Printf("[UserRateLimit] Redis key 已设置自动过期，无需手动清理")
}
