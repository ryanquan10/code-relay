package repository

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"fmt"

	"gorm.io/gorm"
)

// ProductSourceRepository 处理 ProductSource 数据的存储和读取
type ProductSourceRepository struct{}

// NewProductSourceRepository 创建新的 ProductSourceRepository 实例
func NewProductSourceRepository() *ProductSourceRepository {
	return &ProductSourceRepository{}
}

// GetByProductAndSource 根据 ProductID 和 SourceID 查询 ProductSource
func (r *ProductSourceRepository) GetByProductAndSource(productID, sourceID int64) (*entity.ProductSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var productSource entity.ProductSource
	err := db.Where("product_id = ? AND source_id = ?", productID, sourceID).First(&productSource).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get product_source by product_id %d and source_id %d: %w", productID, sourceID, err)
	}

	return &productSource, nil
}

// GetByProductID 根据 ProductID 查询所有 ProductSource
func (r *ProductSourceRepository) GetByProductID(productID int64) ([]entity.ProductSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var productSources []entity.ProductSource
	err := db.Where("product_id = ? AND status = 1", productID).
		Order("priority ASC, weight DESC").
		Find(&productSources).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get product_sources by product_id %d: %w", productID, err)
	}

	return productSources, nil
}

// GetBySourceID 根据 SourceID 查询所有 ProductSource
func (r *ProductSourceRepository) GetBySourceID(sourceID int64) ([]entity.ProductSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var productSources []entity.ProductSource
	err := db.Where("source_id = ?", sourceID).Find(&productSources).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get product_sources by source_id %d: %w", sourceID, err)
	}

	return productSources, nil
}

// GetByID 根据 ID 查询 ProductSource
func (r *ProductSourceRepository) GetByID(id int64) (*entity.ProductSource, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var productSource entity.ProductSource
	err := db.First(&productSource, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get product_source by id %d: %w", id, err)
	}

	return &productSource, nil
}
