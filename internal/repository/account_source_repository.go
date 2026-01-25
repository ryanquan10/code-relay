package repository

import (
	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"
	"fmt"

	"gorm.io/gorm"
)

// AccountSourceRepository 处理 AccountSource 数据的存储和读取
type AccountSourceRepository struct{}

// NewAccountSourceRepository 创建新的 AccountSourceRepository 实例
func NewAccountSourceRepository() *AccountSourceRepository {
	return &AccountSourceRepository{}
}

// GetByID 根据 ID 查询 AccountSource
func (r *AccountSourceRepository) GetByID(id int64) (*entity.AccountSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var source entity.AccountSource
	err := db.First(&source, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get account_source by id %d: %w", id, err)
	}

	return &source, nil
}

// GetBySourceName 根据 SourceName 查询 AccountSource
func (r *AccountSourceRepository) GetBySourceName(sourceName string) (*entity.AccountSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var source entity.AccountSource
	err := db.Where("source_name = ?", sourceName).First(&source).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get account_source by source_name %s: %w", sourceName, err)
	}

	return &source, nil
}

// GetByStatus 根据状态查询 AccountSource 列表
func (r *AccountSourceRepository) GetByStatus(status int) ([]entity.AccountSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var sources []entity.AccountSource
	err := db.Where("status = ?", status).Order("priority ASC").Find(&sources).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get account_sources by status %d: %w", status, err)
	}

	return sources, nil
}

// GetAll 获取所有 AccountSource
func (r *AccountSourceRepository) GetAll() ([]entity.AccountSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var sources []entity.AccountSource
	err := db.Order("priority ASC").Find(&sources).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all account_sources: %w", err)
	}

	return sources, nil
}
