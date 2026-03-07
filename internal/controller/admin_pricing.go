package controller

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"codex-relay/internal/mysql"
	"codex-relay/internal/service"
	"codex-relay/pkg/entity"

	"github.com/gin-gonic/gin"
)

func normalizePricingUnitInput(unit string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(unit)) {
	case "", "US", "USD":
		return "US", true
	case "RMB", "CNY":
		return "RMB", true
	default:
		return "", false
	}
}

// ListPricings 获取计价配置列表（自动补齐 product.account_type 的缺省计价）
func ListPricings(c *gin.Context) {
	pricingService := service.NewPricingService()
	items, err := pricingService.ListByAccountType()
	if err != nil {
		log.Printf("[ListPricings] failed: %v", err)
		fallbackItems, fallbackErr := pricingService.ListExistingPricings()
		if fallbackErr != nil {
			log.Printf("[ListPricings] fallback failed: %v", fallbackErr)
			c.JSON(http.StatusOK, []entity.Pricing{})
			return
		}
		log.Printf("[ListPricings] fallback success: count=%d", len(fallbackItems))
		c.JSON(http.StatusOK, fallbackItems)
		return
	}
	log.Printf("[ListPricings] success: count=%d", len(items))
	c.JSON(http.StatusOK, items)
}

// SyncPricings 手动同步 product.account_type 到 pricing（只补齐，不覆盖已有值）
func SyncPricings(c *gin.Context) {
	pricingService := service.NewPricingService()
	if err := pricingService.EnsureDefaultsFromProducts(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items, err := pricingService.ListByAccountType()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "synced",
		"count":   len(items),
	})
}

// UpdatePricing 更新单条计价配置
func UpdatePricing(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		InTokenUnitPrice  *float64 `json:"in_token_unit_price"`
		OutTokenUnitPrice *float64 `json:"out_token_unit_price"`
		TokenUnit         *int64   `json:"token_unit"`
		Unit              *string  `json:"unit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.InTokenUnitPrice == nil && req.OutTokenUnitPrice == nil && req.TokenUnit == nil && req.Unit == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	var item entity.Pricing
	if err := db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing not found"})
		return
	}

	if req.InTokenUnitPrice != nil {
		if *req.InTokenUnitPrice < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "in_token_unit_price must be >= 0"})
			return
		}
		item.InTokenUnitPrice = *req.InTokenUnitPrice
	}

	if req.OutTokenUnitPrice != nil {
		if *req.OutTokenUnitPrice < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "out_token_unit_price must be >= 0"})
			return
		}
		item.OutTokenUnitPrice = *req.OutTokenUnitPrice
	}

	if req.TokenUnit != nil {
		if *req.TokenUnit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token_unit must be > 0"})
			return
		}
		item.TokenUnit = *req.TokenUnit
	}

	if req.Unit != nil {
		unit, ok := normalizePricingUnitInput(*req.Unit)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unit must be one of: US, RMB"})
			return
		}
		item.Unit = unit
	}

	if err := db.Save(&item).Error; err != nil {
		log.Printf("[UpdatePricing] save failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}
