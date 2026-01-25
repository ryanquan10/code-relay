package repository

import (
	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"
	"fmt"
	"time"
)

// UpstreamErrorLogRepository 处理上游错误日志的存储和读取
type UpstreamErrorLogRepository struct{}

// NewUpstreamErrorLogRepository 创建新的 UpstreamErrorLogRepository 实例
func NewUpstreamErrorLogRepository() *UpstreamErrorLogRepository {
	return &UpstreamErrorLogRepository{}
}

// Create 创建上游错误日志记录
func (r *UpstreamErrorLogRepository) Create(log *entity.UpstreamErrorLog) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	if err := db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create upstream error log: %w", err)
	}

	return nil
}

// GetRecentErrorsBySourceID 获取指定 SourceID 最近的错误日志
func (r *UpstreamErrorLogRepository) GetRecentErrorsBySourceID(sourceID int64, limit int, since time.Time) ([]entity.UpstreamErrorLog, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var logs []entity.UpstreamErrorLog
	err := db.
		Where("source_id = ? AND created_at >= ?", sourceID, since).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent errors for source_id %d: %w", sourceID, err)
	}

	return logs, nil
}

// CountErrorsBySourceID 统计指定时间范围内某个 SourceID 的错误次数
func (r *UpstreamErrorLogRepository) CountErrorsBySourceID(sourceID int64, since time.Time) (int64, error) {
	db := mysql.DB()
	if db == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}

	var count int64
	err := db.
		Model(&entity.UpstreamErrorLog{}).
		Where("source_id = ? AND created_at >= ?", sourceID, since).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count errors for source_id %d: %w", sourceID, err)
	}

	return count, nil
}
