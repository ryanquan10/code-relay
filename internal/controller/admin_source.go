package controller

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListSources 获取货源列表
func ListSources(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var sources []entity.AccountSource
	if err := db.Find(&sources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为前端需要的格式
	type SourceResponse struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	}

	var response []SourceResponse
	for _, s := range sources {
		status := "inactive"
		if s.Status == 1 {
			status = "active"
		}
		remark := ""
		if s.Remark != nil {
			remark = *s.Remark
		}
		response = append(response, SourceResponse{
			ID:          s.ID,
			Name:        s.SourceName,
			Description: remark,
			Status:      status,
			CreatedAt:   s.CreateTime.Format("2006-01-02 15:04:05"),
			UpdatedAt:   s.UpdateTime.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, response)
}

// CreateSource 创建货源
func CreateSource(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := 0
	if req.Status == "active" {
		status = 1
	}

	source := entity.AccountSource{
		SourceName:  req.Name,
		SourceType:  "manual",
		HandlerType: "manual",
		Priority:    1,
		AutoRental:  0,
		Status:      status,
		Remark:      &req.Description,
	}

	if err := db.Create(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": source.ID})
}

// UpdateSource 更新货源
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
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
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

	if req.Name != "" {
		source.SourceName = req.Name
	}
	if req.Description != "" {
		source.Remark = &req.Description
	}
	if req.Status != "" {
		if req.Status == "active" {
			source.Status = 1
		} else {
			source.Status = 0
		}
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
