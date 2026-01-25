package controller

import (
	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListSources 获取账号供应列表
func ListSources(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		log.Println("[ListSources] Error: database not initialized")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var sources []entity.AccountSource
	if err := db.Find(&sources).Error; err != nil {
		log.Printf("[ListSources] Error querying sources: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[ListSources] Found %d sources", len(sources))
	c.JSON(http.StatusOK, sources)
}

// CreateSource 创建账号供应
func CreateSource(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var req struct {
		SourceName    string          `json:"source_name" binding:"required"`
		UpstreamURL   *string         `json:"upstream_url"`
		UpstreamToken *string         `json:"upstream_token"`
		SourceType    string          `json:"source_type" binding:"required"`
		Config        json.RawMessage `json:"config"`
		Priority      *int            `json:"priority"`
		AutoRental    *bool           `json:"auto_rental"`
		Status        *int            `json:"status"`
		Remark        *string         `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.SourceName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_name cannot be empty"})
		return
	}
	if strings.TrimSpace(req.SourceType) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_type cannot be empty"})
		return
	}

	priority := 1
	if req.Priority != nil {
		priority = *req.Priority
	}
	autoRental := false
	if req.AutoRental != nil {
		autoRental = *req.AutoRental
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	remark := normalizeOptionalString(req.Remark)

	config := req.Config
	if len(config) == 0 {
		defaultConfig, _ := json.Marshal(entity.AccountSourceConfig{
			APIURL:      nil,
			APIKey:      nil,
			HandlerType: "manual",
		})
		config = defaultConfig
	}

	source := entity.AccountSource{
		SourceName:    req.SourceName,
		UpstreamURL:   normalizeOptionalString(req.UpstreamURL),
		UpstreamToken: normalizeOptionalString(req.UpstreamToken),
		SourceType:    req.SourceType,
		Config:        config,
		Priority:      priority,
		AutoRental:    autoRental,
		Status:        status,
		Remark:        remark,
	}

	if err := db.Create(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": source.ID})
}

// UpdateSource 更新账号供应
func UpdateSource(c *gin.Context) {
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
		SourceName    *string          `json:"source_name"`
		UpstreamURL   *string          `json:"upstream_url"`
		UpstreamToken *string          `json:"upstream_token"`
		SourceType    *string          `json:"source_type"`
		Config        *json.RawMessage `json:"config"`
		Priority      *int             `json:"priority"`
		AutoRental    *bool            `json:"auto_rental"`
		Status        *int             `json:"status"`
		Remark        *string          `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var source entity.AccountSource
	if err := db.First(&source, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "source not found"})
		return
	}

	if req.SourceName != nil {
		if strings.TrimSpace(*req.SourceName) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_name cannot be empty"})
			return
		}
		source.SourceName = *req.SourceName
	}
	if req.SourceType != nil {
		if strings.TrimSpace(*req.SourceType) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_type cannot be empty"})
			return
		}
		source.SourceType = *req.SourceType
	}
	if req.UpstreamURL != nil {
		source.UpstreamURL = normalizeOptionalString(req.UpstreamURL)
	}
	if req.UpstreamToken != nil {
		source.UpstreamToken = normalizeOptionalString(req.UpstreamToken)
	}
	if req.Config != nil {
		source.Config = *req.Config
	}
	if req.Priority != nil {
		source.Priority = *req.Priority
	}
	if req.AutoRental != nil {
		source.AutoRental = *req.AutoRental
	}
	if req.Status != nil {
		source.Status = *req.Status
	}
	if req.Remark != nil {
		source.Remark = normalizeOptionalString(req.Remark)
	}

	if err := db.Save(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": source.ID})
}

// DeleteSource 删除货源
func DeleteSource(c *gin.Context) {
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

	if err := db.Delete(&entity.AccountSource{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
