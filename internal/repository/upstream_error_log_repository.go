package repository

import (
	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"
	"fmt"
	"time"
)

// UpstreamErrorLogRepository 处理上游错误日志的存储和读取
type UpstreamErrorLogRepository struct{}

// 单个 source 保留的最大错误日志条数
const upstreamErrorLogPerSourceLimit = 1000

// NewUpstreamErrorLogRepository 创建新的 UpstreamErrorLogRepository 实例
func NewUpstreamErrorLogRepository() *UpstreamErrorLogRepository {
	return &UpstreamErrorLogRepository{}
}

// Create 创建上游错误日志记录（并维护每个 source 最多 1000 条，超出则删除最旧记录）
func (r *UpstreamErrorLogRepository) Create(log *entity.UpstreamErrorLog) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	// 使用事务：写入新记录后，基于阈值一次性裁剪（按 created_at,id 排序）
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	if err := tx.Create(log).Error; err != nil {
		_ = tx.Rollback().Error
		return fmt.Errorf("failed to create upstream error log: %w", err)
	}

	// 找到该 source 的第 1000 条最新记录（作为阈值）
	var threshold struct {
		CreatedAt time.Time
		ID        int64
	}
	if err := tx.Raw(`
		SELECT created_at, id FROM (
			SELECT created_at, id
			FROM upstream_error_log
			WHERE source_id = ?
			ORDER BY created_at DESC, id DESC
			LIMIT 1 OFFSET ?
		) AS t
	`, log.SourceID, upstreamErrorLogPerSourceLimit-1).Scan(&threshold).Error; err != nil {
		_ = tx.Rollback().Error
		return fmt.Errorf("failed to find prune threshold for source_id %d: %w", log.SourceID, err)
	}

	// 仅当数量超过上限（阈值存在）时进行裁剪
	if !threshold.CreatedAt.IsZero() || threshold.ID != 0 {
		if err := tx.Exec(`
			DELETE FROM upstream_error_log
			WHERE source_id = ?
			  AND (created_at < ? OR (created_at = ? AND id < ?))
		`, log.SourceID, threshold.CreatedAt, threshold.CreatedAt, threshold.ID).Error; err != nil {
			_ = tx.Rollback().Error
			return fmt.Errorf("failed to prune old logs for source_id %d: %w", log.SourceID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit log create/prune: %w", err)
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
		Order("created_at DESC, id DESC").
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
