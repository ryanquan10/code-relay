package repository

import (
	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"
	"fmt"

	"gorm.io/gorm"
)

// AccountSourceProductRepository 处理 AccountSourceProcut 数据的存储和读取
type AccountSourceProductRepository struct{}

// NewAccountSourceProductRepository 创建新的 AccountSourceProductRepository 实例
func NewAccountSourceProductRepository() *AccountSourceProductRepository {
	return &AccountSourceProductRepository{}
}

// GetSourcesByProductID 根据 ProductID 查询所有关联的 AccountSource（按 Priority 排序）
func (r *AccountSourceProductRepository) GetSourcesByProductID(productID int64) ([]entity.AccountSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var sources []entity.AccountSource
	err := db.
		Table("account_source").
		Joins("INNER JOIN account_source_procut ON account_source.id = account_source_procut.source_id").
		Where("account_source_procut.product_id = ? AND account_source.status = 1", productID).
		Order("account_source.priority ASC, account_source_procut.weight DESC").
		Find(&sources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get sources by product_id %d: %w", productID, err)
	}

	return sources, nil
}

// GetByProductIDAndSourceID 根据 ProductID 和 SourceID 查询关联记录
func (r *AccountSourceProductRepository) GetByProductIDAndSourceID(productID, sourceID int64) (*entity.AccountSourceProcut, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var mapping entity.AccountSourceProcut
	err := db.Where("product_id = ? AND source_id = ?", productID, sourceID).First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get mapping by product_id %d and source_id %d: %w", productID, sourceID, err)
	}

	return &mapping, nil
}
