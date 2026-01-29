package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	redisstore "codex-relay/internal/redis"

	"github.com/redis/go-redis/v9"
)

const (
	// UserTrackingKey Redis Sorted Set key for tracking active users
	UserTrackingKey = "active_users_tracking"

	// MaxUsersKey Redis key for storing max users count for today
	MaxUsersKeyPrefix = "max_users:"
)

// UserTrackingService 用户追踪服务
type UserTrackingService struct {
	client *redis.Client
	ctx    context.Context
}

// NewUserTrackingService 创建用户追踪服务
func NewUserTrackingService() *UserTrackingService {
	return &UserTrackingService{
		client: redisstore.Client(),
		ctx:    redisstore.Context(),
	}
}

// TrackUserAccess 记录用户访问（使用 token 作为唯一标识）
// 将用户 token 添加到 Sorted Set，score 为当前时间戳
func (s *UserTrackingService) TrackUserAccess(customerToken string) error {
	if s.client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	now := time.Now().Unix()

	// 添加到 Sorted Set，score 为当前时间戳
	err := s.client.ZAdd(s.ctx, UserTrackingKey, redis.Z{
		Score:  float64(now),
		Member: customerToken,
	}).Err()

	if err != nil {
		return fmt.Errorf("记录用户访问失败: %w", err)
	}

	return nil
}

// GetActiveUsersInLastMinute 获取最近1分钟内的活跃用户数
func (s *UserTrackingService) GetActiveUsersInLastMinute() (int64, error) {
	if s.client == nil {
		return 0, fmt.Errorf("Redis 客户端未初始化")
	}

	now := time.Now().Unix()
	oneMinuteAgo := now - 60

	// 统计最近1分钟内的用户数
	count, err := s.client.ZCount(s.ctx, UserTrackingKey,
		strconv.FormatInt(oneMinuteAgo, 10),
		strconv.FormatInt(now, 10)).Result()

	if err != nil {
		return 0, fmt.Errorf("获取活跃用户数失败: %w", err)
	}

	return count, nil
}

// UpdateMaxUsersForToday 更新今天的最大用户数
func (s *UserTrackingService) UpdateMaxUsersForToday() error {
	if s.client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	// 获取当前活跃用户数
	currentUsers, err := s.GetActiveUsersInLastMinute()
	if err != nil {
		return err
	}

	// 获取今天的日期作为 key
	today := time.Now().Format("2006-01-02")
	maxUsersKey := MaxUsersKeyPrefix + today

	// 获取当前记录的最大值
	maxUsersStr, err := s.client.Get(s.ctx, maxUsersKey).Result()
	var maxUsers int64 = 0
	if err == nil {
		maxUsers, _ = strconv.ParseInt(maxUsersStr, 10, 64)
	}

	// 如果当前用户数大于记录的最大值，更新
	if currentUsers > maxUsers {
		err = s.client.Set(s.ctx, maxUsersKey, currentUsers, 0).Err()
		if err != nil {
			return fmt.Errorf("更新最大用户数失败: %w", err)
		}
		log.Printf("[用户追踪] 更新今日最大用户数: %d -> %d", maxUsers, currentUsers)
	}

	return nil
}

// GetMaxUsersForToday 获取今天的最大用户数
func (s *UserTrackingService) GetMaxUsersForToday() (int64, error) {
	if s.client == nil {
		return 0, fmt.Errorf("Redis 客户端未初始化")
	}

	today := time.Now().Format("2006-01-02")
	maxUsersKey := MaxUsersKeyPrefix + today

	maxUsersStr, err := s.client.Get(s.ctx, maxUsersKey).Result()
	if err == redis.Nil {
		return 0, nil // 今天还没有记录
	}
	if err != nil {
		return 0, fmt.Errorf("获取最大用户数失败: %w", err)
	}

	maxUsers, err := strconv.ParseInt(maxUsersStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析最大用户数失败: %w", err)
	}

	return maxUsers, nil
}

// CleanupOldTracking 清理旧的追踪数据（保留最近2分钟的数据）
func (s *UserTrackingService) CleanupOldTracking() error {
	if s.client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	// 删除2分钟前的数据
	twoMinutesAgo := time.Now().Unix() - 120

	removed, err := s.client.ZRemRangeByScore(s.ctx, UserTrackingKey,
		"-inf",
		strconv.FormatInt(twoMinutesAgo, 10)).Result()

	if err != nil {
		return fmt.Errorf("清理旧数据失败: %w", err)
	}

	if removed > 0 {
		log.Printf("[用户追踪] 清理了 %d 条旧记录", removed)
	}

	return nil
}
