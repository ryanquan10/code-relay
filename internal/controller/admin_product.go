package controller

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

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

	// 转换为前端需要的格式
	type ProductResponse struct {
		ID               int64    `json:"id"`
		ProductCode      string   `json:"product_code"`
		ProductName      string   `json:"product_name"`
		AccountType      string   `json:"account_type"`
		Category         string   `json:"category"`
		Price            float64  `json:"price"`
		OriginalPrice    *float64 `json:"original_price"`
		ValidityDays     int      `json:"validity_days"`
		SharedLimit      int      `json:"shared_limit"`
		SalesCount       int      `json:"sales_count"`
		AutoDelivery     bool     `json:"auto_delivery"`
		SortOrder        int      `json:"sort_order"`
		Status           string   `json:"status"`
		Description      string   `json:"description"`
		CreatedAt        string   `json:"created_at"`
		UpdatedAt        string   `json:"updated_at"`
		SourceIDs        []int64  `json:"source_ids"`
	}

	var response []ProductResponse
	for _, p := range products {
		status := "inactive"
		if p.Status == 1 {
			status = "active"
		}

		category := ""
		if p.Category != nil {
			category = *p.Category
		}

		description := ""
		if p.Description != nil {
			description = *p.Description
		}

		autoDelivery := p.AutoDelivery == 1

		// 查询关联的货源
		var productSources []entity.ProductSource
		db.Where("product_id = ?", p.ID).Find(&productSources)

		sourceIDs := make([]int64, 0, len(productSources))
		for _, ps := range productSources {
			sourceIDs = append(sourceIDs, ps.SourceID)
		}

		response = append(response, ProductResponse{
			ID:            p.ID,
			ProductCode:   p.ProductCode,
			ProductName:   p.ProductName,
			AccountType:   p.AccountType,
			Category:      category,
			Price:         p.Price,
			OriginalPrice: p.OriginalPrice,
			ValidityDays:  p.ValidityDays,
			SharedLimit:   p.SharedLimit,
			SalesCount:    p.SalesCount,
			AutoDelivery:  autoDelivery,
			SortOrder:     p.SortOrder,
			Status:        status,
			Description:   description,
			CreatedAt:     p.CreateTime.Format("2006-01-02 15:04:05"),
			UpdatedAt:     p.UpdateTime.Format("2006-01-02 15:04:05"),
			SourceIDs:     sourceIDs,
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
		ProductCode   string   `json:"product_code" binding:"required"`
		ProductName   string   `json:"product_name" binding:"required"`
		AccountType   string   `json:"account_type" binding:"required"`
		Category      string   `json:"category"`
		Price         float64  `json:"price" binding:"required"`
		OriginalPrice *float64 `json:"original_price"`
		ValidityDays  int      `json:"validity_days"`
		SharedLimit   int      `json:"shared_limit"`
		AutoDelivery  bool     `json:"auto_delivery"`
		SortOrder     int      `json:"sort_order"`
		Status        string   `json:"status"`
		Description   string   `json:"description"`
		SourceIDs     []int64  `json:"source_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := 0
	if req.Status == "active" {
		status = 1
	}

	autoDelivery := 0
	if req.AutoDelivery {
		autoDelivery = 1
	}

	category := &req.Category
	if req.Category == "" {
		category = nil
	}

	description := &req.Description
	if req.Description == "" {
		description = nil
	}

	product := entity.Product{
		ProductCode:   req.ProductCode,
		ProductName:   req.ProductName,
		AccountType:   req.AccountType,
		Category:      category,
		Price:         req.Price,
		OriginalPrice: req.OriginalPrice,
		ValidityDays:  req.ValidityDays,
		SharedLimit:   req.SharedLimit,
		SalesCount:    0,
		AutoDelivery:  autoDelivery,
		SortOrder:     req.SortOrder,
		Status:        status,
		Description:   description,
	}

	// 使用事务创建产品和关联货源
	tx := db.Begin()
	if err := tx.Create(&product).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建产品-货源关联
	for _, sourceID := range req.SourceIDs {
		productSource := entity.ProductSource{
			ProductID: product.ID,
			SourceID:  sourceID,
			Priority:  1,
			Weight:    1,
			Stock:     0,
			Status:    1,
		}
		if err := tx.Create(&productSource).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	tx.Commit()
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
		ProductCode   string   `json:"product_code"`
		ProductName   string   `json:"product_name"`
		AccountType   string   `json:"account_type"`
		Category      string   `json:"category"`
		Price         float64  `json:"price"`
		OriginalPrice *float64 `json:"original_price"`
		ValidityDays  int      `json:"validity_days"`
		SharedLimit   int      `json:"shared_limit"`
		AutoDelivery  bool     `json:"auto_delivery"`
		SortOrder     int      `json:"sort_order"`
		Status        string   `json:"status"`
		Description   string   `json:"description"`
		SourceIDs     []int64  `json:"source_ids"`
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
	if req.ProductCode != "" {
		product.ProductCode = req.ProductCode
	}
	if req.ProductName != "" {
		product.ProductName = req.ProductName
	}
	if req.AccountType != "" {
		product.AccountType = req.AccountType
	}
	if req.Category != "" {
		product.Category = &req.Category
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	product.OriginalPrice = req.OriginalPrice
	if req.ValidityDays > 0 {
		product.ValidityDays = req.ValidityDays
	}
	if req.SharedLimit >= 0 {
		product.SharedLimit = req.SharedLimit
	}
	if req.AutoDelivery {
		product.AutoDelivery = 1
	} else {
		product.AutoDelivery = 0
	}
	if req.SortOrder >= 0 {
		product.SortOrder = req.SortOrder
	}
	if req.Status != "" {
		if req.Status == "active" {
			product.Status = 1
		} else {
			product.Status = 0
		}
	}
	if req.Description != "" {
		product.Description = &req.Description
	}

	// 使用事务更新产品和关联货源
	tx := db.Begin()
	if err := tx.Save(&product).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 删除旧的产品-货源关联
	if err := tx.Where("product_id = ?", product.ID).Delete(&entity.ProductSource{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建新的产品-货源关联
	for _, sourceID := range req.SourceIDs {
		productSource := entity.ProductSource{
			ProductID: product.ID,
			SourceID:  sourceID,
			Priority:  1,
			Weight:    1,
			Stock:     0,
			Status:    1,
		}
		if err := tx.Create(&productSource).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	tx.Commit()
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

	// 使用事务删除产品和关联货源
	tx := db.Begin()

	// 删除产品-货源关联
	if err := tx.Where("product_id = ?", id).Delete(&entity.ProductSource{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 删除产品
	if err := tx.Delete(&entity.Product{}, id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
