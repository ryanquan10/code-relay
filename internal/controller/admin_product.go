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
	response = make([]ProductResponse, 0)
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

		// 新表结构：Product 直接包含 SourceID，返回单个 SourceID
		sourceIDs := []int64{p.SourceID}

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
			AutoDelivery:  p.AutoDelivery,
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

	category := &req.Category
	if req.Category == "" {
		category = nil
	}

	description := &req.Description
	if req.Description == "" {
		description = nil
	}

	// 新表结构：Product 直接包含 SourceID（取第一个）
	var sourceID int64 = 0
	if len(req.SourceIDs) > 0 {
		sourceID = req.SourceIDs[0]
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
		AutoDelivery:  req.AutoDelivery,
		SortOrder:     req.SortOrder,
		Status:        status,
		Description:   description,
		SourceID:      sourceID,
	}

	if err := db.Create(&product).Error; err != nil {
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
	product.AutoDelivery = req.AutoDelivery
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

	// 更新 SourceID（取第一个）
	if len(req.SourceIDs) > 0 {
		product.SourceID = req.SourceIDs[0]
	}

	if err := db.Save(&product).Error; err != nil {
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

	// 直接删除产品（新表结构无需删除关联）
	if err := db.Delete(&entity.Product{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
