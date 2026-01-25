package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type productSourcePayload struct {
	SourceID int64 `json:"source_id"`
	Weight   int   `json:"weight"`
}

type productResponse struct {
	entity.Product
	Sources []productSourcePayload `json:"sources"`
}

// ListProducts 获取产品列表
func ListProducts(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var products []entity.Product
	if err := db.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(products) == 0 {
		c.JSON(http.StatusOK, []productResponse{})
		return
	}

	productIDs := make([]int64, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.ID)
	}

	var mappings []entity.AccountSourceProcut
	if err := db.Where("product_id IN ?", productIDs).Find(&mappings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sourceMap := make(map[int64][]productSourcePayload, len(products))
	for _, mapping := range mappings {
		sourceMap[mapping.ProductID] = append(sourceMap[mapping.ProductID], productSourcePayload{
			SourceID: mapping.SourceID,
			Weight:   mapping.Weight,
		})
	}

	response := make([]productResponse, 0, len(products))
	for i := range products {
		product := products[i]
		response = append(response, productResponse{
			Product: product,
			Sources: sourceMap[product.ID],
		})
	}

	c.JSON(http.StatusOK, response)
}

// CreateProduct 创建产品
func CreateProduct(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var req struct {
		ProductCode      string                 `json:"product_code" binding:"required"`
		ProductName      string                 `json:"product_name" binding:"required"`
		AccountType      *string                `json:"account_type"`
		Category         *string                `json:"category"`
		Icon             *string                `json:"icon"`
		ImageURL         *string                `json:"image_url"`
		DownStreamURL    *string                `json:"down_stream_url"`
		Description      *string                `json:"description"`
		Price            *float64               `json:"price" binding:"required"`
		OriginalPrice    *float64               `json:"original_price"`
		SalesCount       *int                   `json:"sales_count"`
		ContactInfo      *string                `json:"contact_info"`
		UsageInstruction *string                `json:"usage_instruction"`
		ValidityDays     *int                   `json:"validity_days"`
		SharedLimit      *int                   `json:"shared_limit"`
		Sources          []productSourcePayload `json:"sources" binding:"required"`
		CostPrice        *float64               `json:"cost_price"`
		DefaultBalance   *float64               `json:"default_balance"`
		OriginalBalance  *float64               `json:"original_balance"`
		Stock            *int                   `json:"stock"`
		AutoDelivery     *bool                  `json:"auto_delivery"`
		SortOrder        *int                   `json:"sort_order"`
		Status           *int                   `json:"status"`
		Version          *int                   `json:"version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	productCode := strings.TrimSpace(req.ProductCode)
	if productCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_code cannot be empty"})
		return
	}
	productName := strings.TrimSpace(req.ProductName)
	if productName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_name cannot be empty"})
		return
	}
	if err := validateProductSources(req.Sources); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountType := ""
	if req.AccountType != nil {
		accountType = strings.TrimSpace(*req.AccountType)
	}
	if req.Price == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price required"})
		return
	}
	price := *req.Price
	validityDays := 9999
	if req.ValidityDays != nil {
		validityDays = *req.ValidityDays
	}
	sharedLimit := 0
	if req.SharedLimit != nil {
		sharedLimit = *req.SharedLimit
	}
	salesCount := 0
	if req.SalesCount != nil {
		salesCount = *req.SalesCount
	}
	costPrice := 0.0
	if req.CostPrice != nil {
		costPrice = *req.CostPrice
	}
	defaultBalance := 0.0
	if req.DefaultBalance != nil {
		defaultBalance = *req.DefaultBalance
	}
	originalBalance := 0.0
	if req.OriginalBalance != nil {
		originalBalance = *req.OriginalBalance
	}
	stock := 0
	if req.Stock != nil {
		stock = *req.Stock
	}
	autoDelivery := false
	if req.AutoDelivery != nil {
		autoDelivery = *req.AutoDelivery
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	version := 0
	if req.Version != nil {
		version = *req.Version
	}

	product := entity.Product{
		ProductCode:      productCode,
		ProductName:      productName,
		AccountType:      accountType,
		Category:         normalizeOptionalString(req.Category),
		Icon:             normalizeOptionalString(req.Icon),
		ImageURL:         normalizeOptionalString(req.ImageURL),
		DownStreamURL:    normalizeOptionalString(req.DownStreamURL),
		Description:      normalizeOptionalString(req.Description),
		Price:            price,
		OriginalPrice:    req.OriginalPrice,
		SalesCount:       salesCount,
		ContactInfo:      normalizeOptionalString(req.ContactInfo),
		UsageInstruction: normalizeOptionalString(req.UsageInstruction),
		ValidityDays:     validityDays,
		SharedLimit:      sharedLimit,
		CostPrice:        costPrice,
		DefaultBalance:   defaultBalance,
		OriginalBalance:  originalBalance,
		Stock:            stock,
		AutoDelivery:     autoDelivery,
		SortOrder:        sortOrder,
		Status:           status,
		Version:          version,
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&product).Error; err != nil {
			return err
		}

		mappings := make([]entity.AccountSourceProcut, 0, len(req.Sources))
		for _, source := range req.Sources {
			mappings = append(mappings, entity.AccountSourceProcut{
				ProductID: product.ID,
				SourceID:  source.SourceID,
				Weight:    source.Weight,
			})
		}

		if err := tx.Create(&mappings).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": product.ID})
}

// UpdateProduct 更新产品
func UpdateProduct(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		ProductCode      *string                 `json:"product_code"`
		ProductName      *string                 `json:"product_name"`
		AccountType      *string                 `json:"account_type"`
		Category         *string                 `json:"category"`
		Icon             *string                 `json:"icon"`
		ImageURL         *string                 `json:"image_url"`
		DownStreamURL    *string                 `json:"down_stream_url"`
		Description      *string                 `json:"description"`
		Price            *float64                `json:"price"`
		OriginalPrice    *float64                `json:"original_price"`
		SalesCount       *int                    `json:"sales_count"`
		ContactInfo      *string                 `json:"contact_info"`
		UsageInstruction *string                 `json:"usage_instruction"`
		ValidityDays     *int                    `json:"validity_days"`
		SharedLimit      *int                    `json:"shared_limit"`
		Sources          *[]productSourcePayload `json:"sources"`
		CostPrice        *float64                `json:"cost_price"`
		DefaultBalance   *float64                `json:"default_balance"`
		OriginalBalance  *float64                `json:"original_balance"`
		Stock            *int                    `json:"stock"`
		AutoDelivery     *bool                   `json:"auto_delivery"`
		SortOrder        *int                    `json:"sort_order"`
		Status           *int                    `json:"status"`
		Version          *int                    `json:"version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var product entity.Product
	if err := db.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	// 更新产品字段
	if req.ProductCode != nil {
		productCode := strings.TrimSpace(*req.ProductCode)
		if productCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_code cannot be empty"})
			return
		}
		product.ProductCode = productCode
	}
	if req.ProductName != nil {
		productName := strings.TrimSpace(*req.ProductName)
		if productName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_name cannot be empty"})
			return
		}
		product.ProductName = productName
	}
	if req.AccountType != nil {
		product.AccountType = strings.TrimSpace(*req.AccountType)
	}
	if req.Category != nil {
		product.Category = normalizeOptionalString(req.Category)
	}
	if req.Icon != nil {
		product.Icon = normalizeOptionalString(req.Icon)
	}
	if req.ImageURL != nil {
		product.ImageURL = normalizeOptionalString(req.ImageURL)
	}
	if req.DownStreamURL != nil {
		product.DownStreamURL = normalizeOptionalString(req.DownStreamURL)
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.OriginalPrice != nil {
		product.OriginalPrice = req.OriginalPrice
	}
	if req.SalesCount != nil {
		product.SalesCount = *req.SalesCount
	}
	if req.ContactInfo != nil {
		product.ContactInfo = normalizeOptionalString(req.ContactInfo)
	}
	if req.UsageInstruction != nil {
		product.UsageInstruction = normalizeOptionalString(req.UsageInstruction)
	}
	if req.ValidityDays != nil {
		product.ValidityDays = *req.ValidityDays
	}
	if req.SharedLimit != nil {
		product.SharedLimit = *req.SharedLimit
	}
	if req.CostPrice != nil {
		product.CostPrice = *req.CostPrice
	}
	if req.DefaultBalance != nil {
		product.DefaultBalance = *req.DefaultBalance
	}
	if req.OriginalBalance != nil {
		product.OriginalBalance = *req.OriginalBalance
	}
	if req.Stock != nil {
		product.Stock = *req.Stock
	}
	if req.AutoDelivery != nil {
		product.AutoDelivery = *req.AutoDelivery
	}
	if req.SortOrder != nil {
		product.SortOrder = *req.SortOrder
	}
	if req.Status != nil {
		product.Status = *req.Status
	}
	if req.Description != nil {
		product.Description = normalizeOptionalString(req.Description)
	}
	if req.Version != nil {
		product.Version = *req.Version
	}

	if req.Sources != nil {
		if err := validateProductSources(*req.Sources); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&product).Error; err != nil {
			return err
		}

		if req.Sources != nil {
			if err := tx.Where("product_id = ?", product.ID).Delete(&entity.AccountSourceProcut{}).Error; err != nil {
				return err
			}
			mappings := make([]entity.AccountSourceProcut, 0, len(*req.Sources))
			for _, source := range *req.Sources {
				mappings = append(mappings, entity.AccountSourceProcut{
					ProductID: product.ID,
					SourceID:  source.SourceID,
					Weight:    source.Weight,
				})
			}
			if err := tx.Create(&mappings).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": product.ID})
}

// DeleteProduct 删除产品
func DeleteProduct(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", id).Delete(&entity.AccountSourceProcut{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&entity.Product{}, id).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func validateProductSources(sources []productSourcePayload) error {
	if len(sources) == 0 {
		return fmt.Errorf("sources required")
	}
	seen := make(map[int64]struct{}, len(sources))
	for _, source := range sources {
		if source.SourceID <= 0 {
			return fmt.Errorf("source_id must be greater than 0")
		}
		if source.Weight <= 0 {
			return fmt.Errorf("weight must be greater than 0")
		}
		if _, ok := seen[source.SourceID]; ok {
			return fmt.Errorf("duplicate source_id %d", source.SourceID)
		}
		seen[source.SourceID] = struct{}{}
	}
	return nil
}
