package repository

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UsageRepository 处理 Usage 数据的存储和读取
type UsageRepository struct{}

// NewUsageRepository 创建新的 UsageRepository 实例
func NewUsageRepository() *UsageRepository {
	return &UsageRepository{}
}

// Create 创建新的 Usage 记录
func (r *UsageRepository) Create(usage *entity.Usage) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	if err := db.Create(usage).Error; err != nil {
		return fmt.Errorf("failed to create usage record: %w", err)
	}

	return nil
}

// GetTodayUsageList 获取当天所有的 Usage 记录
// 根据 create_time 是当天的条件进行查询
func (r *UsageRepository) GetTodayUsageList() ([]entity.Usage, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	// 获取今天的开始和结束时间
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var usageList []entity.Usage
	err := db.Where("create_time >= ? AND create_time < ?", startOfDay, endOfDay).
		Find(&usageList).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get today's usage list: %w", err)
	}

	return usageList, nil
}

// GetUsageByDateRange 获取指定日期范围内的 Usage 记录
func (r *UsageRepository) GetUsageByDateRange(startTime, endTime time.Time) ([]entity.Usage, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var usageList []entity.Usage
	err := db.Where("create_time >= ? AND create_time < ?", startTime, endTime).
		Find(&usageList).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get usage list by date range: %w", err)
	}

	return usageList, nil
}

// GetTodayUsageByAccountId 获取指定账户当天的所有 Usage 记录
func (r *UsageRepository) GetTodayUsageByAccountId(accountId uint64) ([]entity.Usage, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	// 获取今天的开始和结束时间
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var usageList []entity.Usage
	err := db.Where("account_id = ? AND create_time >= ? AND create_time < ?",
		accountId, startOfDay, endOfDay).
		Find(&usageList).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get today's usage for account %d: %w", accountId, err)
	}

	return usageList, nil
}

// GetTodayTotalConsumeByAccountId 获取指定账户当天的总消费
func (r *UsageRepository) GetTodayTotalConsumeByAccountId(accountId uint64) (float64, error) {
	db := mysql.DB()
	if db == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}

	// 获取今天的开始和结束时间
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var totalConsume float64
	err := db.Model(&entity.Usage{}).
		Where("account_id = ? AND create_time >= ? AND create_time < ?",
			accountId, startOfDay, endOfDay).
		Select("COALESCE(SUM(consume), 0)").
		Scan(&totalConsume).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get today's total consume for account %d: %w", accountId, err)
	}

	return totalConsume, nil
}

// GetUsageByAccountIdAndDateRange 获取指定账户在指定日期范围内的 Usage 记录
func (r *UsageRepository) GetUsageByAccountIdAndDateRange(accountId uint64, startTime, endTime time.Time) ([]entity.Usage, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var usageList []entity.Usage
	err := db.Where("account_id = ? AND create_time >= ? AND create_time < ?",
		accountId, startTime, endTime).
		Find(&usageList).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get usage for account %d: %w", accountId, err)
	}

	return usageList, nil
}

// GetTotalConsumeByAccountIdAndDateRange 获取指定账户在指定日期范围内的总消费
func (r *UsageRepository) GetTotalConsumeByAccountIdAndDateRange(accountId uint64, startTime, endTime time.Time) (float64, error) {
	db := mysql.DB()
	if db == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}

	var totalConsume float64
	err := db.Model(&entity.Usage{}).
		Where("account_id = ? AND create_time >= ? AND create_time < ?",
			accountId, startTime, endTime).
		Select("COALESCE(SUM(consume), 0)").
		Scan(&totalConsume).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get total consume for account %d: %w", accountId, err)
	}

	return totalConsume, nil
}

// GetById 根据 ID 获取 Usage 记录
func (r *UsageRepository) GetById(id uint64) (*entity.Usage, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var usage entity.Usage
	err := db.First(&usage, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get usage by id %d: %w", id, err)
	}

	return &usage, nil
}

// GetDailyUsageByAccountId 获取指定账户按天分组的使用统计
// startTime: 开始日期
// endTime: 结束日期
// 返回每天的消费总额和记录数
func (r *UsageRepository) GetDailyUsageByAccountId(accountId uint64, startTime, endTime time.Time) ([]entity.DailyUsage, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var dailyUsages []entity.DailyUsage

	// 使用 DATE(create_time) 按天分组
	err := db.Model(&entity.Usage{}).
		Select("DATE(create_time) as date, SUM(consume) as total_consume, COUNT(*) as record_count").
		Where("account_id = ? AND create_time >= ? AND create_time < ?",
			accountId, startTime, endTime).
		Group("DATE(create_time)").
		Order("date DESC").
		Scan(&dailyUsages).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage for account %d: %w", accountId, err)
	}

	return dailyUsages, nil
}

// GetDailyUsageByToken 获取指定 token 按天分组的使用统计（需要先查询 account_id）
// 这个方法在 Service 层调用会更合适，这里仅提供基础查询
func (r *UsageRepository) GetDailyUsageGroupByDate(startTime, endTime time.Time) ([]entity.DailyUsage, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var dailyUsages []entity.DailyUsage

	// 获取所有账户按天分组的统计
	err := db.Model(&entity.Usage{}).
		Select("DATE(create_time) as date, SUM(consume) as total_consume, COUNT(*) as record_count").
		Where("create_time >= ? AND create_time < ?", startTime, endTime).
		Group("DATE(create_time)").
		Order("date DESC").
		Scan(&dailyUsages).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}

	return dailyUsages, nil
}
