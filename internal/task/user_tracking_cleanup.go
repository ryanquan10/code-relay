package task

import (
	"codex-relay/internal/service"
	"context"
	"log"
	"time"
)

// UserTrackingCleanupTask 用户追踪数据清理任务
type UserTrackingCleanupTask struct {
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewUserTrackingCleanupTask 创建用户追踪清理任务
// interval: 清理间隔，建议 30 秒到 1 分钟
func NewUserTrackingCleanupTask(interval time.Duration) *UserTrackingCleanupTask {
	ctx, cancel := context.WithCancel(context.Background())
	return &UserTrackingCleanupTask{
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动清理任务
func (t *UserTrackingCleanupTask) Start() {
	log.Printf("[用户追踪清理] 任务启动，清理间隔: %v", t.interval)

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	// 立即执行一次清理
	t.cleanup()

	for {
		select {
		case <-t.ctx.Done():
			log.Printf("[用户追踪清理] 任务停止")
			return
		case <-ticker.C:
			t.cleanup()
		}
	}
}

// cleanup 执行清理操作
func (t *UserTrackingCleanupTask) cleanup() {
	trackingService := service.NewUserTrackingService()

	// 清理旧的追踪数据
	if err := trackingService.CleanupOldTracking(); err != nil {
		log.Printf("[用户追踪清理] 清理失败: %v", err)
		return
	}

	// 更新今天的最大用户数
	if err := trackingService.UpdateMaxUsersForToday(); err != nil {
		log.Printf("[用户追踪清理] 更新最大用户数失败: %v", err)
	}
}

// Stop 停止清理任务
func (t *UserTrackingCleanupTask) Stop() {
	t.cancel()
}
