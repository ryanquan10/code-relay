package repository

import (
	"codex-relay/pkg/entity"
	"fmt"
	"time"

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

func (r *DeliveryRepository) FindProductByInnerProductCode(productCode string) (*entity.Product, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var product entity.Product
	err := r.db.Where("product_code = ?", productCode).
		Order("CASE WHEN validity_days = 1 THEN 0 ELSE 1 END, id ASC").
		First(&product).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get product by inner product code %s: %w", productCode, err)
	}

	//todo 找到product以後要按照其他方法一樣
	return &product, nil
}

func (r *DeliveryRepository) FindProductByPlatform(platform, productCode string, sku *string) (*entity.Product, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var product entity.Product
	var err error

	if sku != nil && *sku != "" {
		// 当提供了 SKU 时，必须匹配 platform、product_code 和 sku
		err = r.db.Where("JSON_CONTAINS(platforms, JSON_OBJECT('platform', ?, 'product_code', ?, 'sku', ?)) = 1",
			platform, productCode, *sku).
			First(&product).Error
	} else {
		// 没有提供 SKU 时，只匹配 platform 和 product_code；若匹配多条，优先 validity_days=1
		err = r.db.Where("JSON_CONTAINS(platforms, JSON_OBJECT('platform', ?, 'product_code', ?)) = 1",
			platform, productCode).
			Order("CASE WHEN validity_days = 1 THEN 0 ELSE 1 END, id ASC").
			First(&product).Error
	}

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

	updates := map[string]interface{}{
		"use_status": useStatus,
	}

	// 当 useStatus 设置为 1（已使用）时，更新 StartTime 为当前时间
	if useStatus == 1 {
		now := time.Now()
		updates["start_time"] = now
	}

	err := r.db.Model(&entity.Account{}).
		Where("id = ?", accountID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update use_status for account %d: %w", accountID, err)
	}

	return nil
}

// ListProductsByPlatform 返回匹配 platform + product_code 的所有 Product（不考虑 SKU）
func (r *DeliveryRepository) ListProductsByPlatform(platform, productCode string) ([]entity.Product, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	var products []entity.Product
	err := r.db.Where("JSON_CONTAINS(platforms, JSON_OBJECT('platform', ?, 'product_code', ?)) = 1",
		platform, productCode).
		Order("id ASC").
		Find(&products).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list products by platform %s and product_code %s: %w", platform, productCode, err)
	}
	return products, nil
}
