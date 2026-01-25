package repository

import (
	"codex-relay/pkg/entity"
	"fmt"

	"gorm.io/gorm"
)

// DeliveryRepository 处理 delivery 模块的数据查询
type DeliveryRepository struct {
	db *gorm.DB
}

// NewDeliveryRepository 创建新的 DeliveryRepository 实例
func NewDeliveryRepository(db *gorm.DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

// FindProductByPlatform 根据 Platform 和 ProductCode 查询 Product
// 在 Product.Platforms JSON 数组中查找匹配的记录
func (r *DeliveryRepository) FindProductByPlatform(platform, productCode string) (*entity.Product, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var product entity.Product
	// 使用 JSON_CONTAINS 查询 platforms 字段
	err := r.db.Where("JSON_CONTAINS(platforms, JSON_OBJECT('platform', ?, 'product_code', ?)) = 1",
		platform, productCode).
		First(&product).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get product by platform %s and product_code %s: %w", platform, productCode, err)
	}

	return &product, nil
}

// FindAvailableAccount 根据 ProductID 查询可用的 Account (use_status=0, status=active)
func (r *DeliveryRepository) FindAvailableAccount(productID int64) (*entity.Account, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var account entity.Account
	err := r.db.Where("product_id = ? AND use_status = 0 AND status = 'active'", productID).
		Order("id ASC").
		First(&account).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get available account by product_id %d: %w", productID, err)
	}

	return &account, nil
}

// UpdateAccountUseStatus 更新 Account 的 UseStatus
func (r *DeliveryRepository) UpdateAccountUseStatus(accountID uint64, useStatus int) error {
	if r.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	err := r.db.Model(&entity.Account{}).
		Where("id = ?", accountID).
		Update("use_status", useStatus).Error

	if err != nil {
		return fmt.Errorf("failed to update use_status for account %d: %w", accountID, err)
	}

	return nil
}
