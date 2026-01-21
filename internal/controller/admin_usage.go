package controller

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListUsages 获取使用量列表
func ListUsages(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var usages []entity.Usage
	if err := db.Order("create_time DESC").Limit(1000).Find(&usages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为前端需要的格式
	type UsageResponse struct {
		ID          uint64  `json:"id"`
		UserID      uint64  `json:"user_id"`
		AccountID   uint64  `json:"account_id"`
		CustomerKey string  `json:"customer_key"`
		Tokens      uint64  `json:"tokens"`
		Consume     float64 `json:"consume"`
		Date        string  `json:"date"`
		CreatedAt   string  `json:"created_at"`
	}

	var response []UsageResponse
	for _, u := range usages {
		response = append(response, UsageResponse{
			ID:          u.ID,
			AccountID:   u.AccountID,
			Consume:     u.Consume,
			Date:        u.CreateTime.Format("2006-01-02"),
			CreatedAt:   u.CreateTime.Format("2006-01-02 15:04:05"),
			CustomerKey: "N/A", // 如果有 customer_key 字段请修改
			Tokens:      0,     // 如果有 tokens 字段请修改
		})
	}

	c.JSON(http.StatusOK, response)
}

// QueryUsageByToken 根据 customer_key 查询使用量
func QueryUsageByToken(c *gin.Context) {
	var req struct {
		CustomerKey string   `json:"customer_key"`
		Dates       []string `json:"dates"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 根据 customer_key 查询，需要在数据库中建立映射关系
	// 这里返回示例数据
	c.JSON(http.StatusOK, gin.H{
		"total_consume": 0.0,
		"details":       []gin.H{},
	})
}

// GetUsageStats 获取使用量统计
func GetUsageStats(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var result struct {
		TotalConsume float64
		TotalTokens  uint64
		RecordCount  int64
	}

	query := db.Model(&entity.Usage{})
	if startDate != "" {
		query = query.Where("create_time >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("create_time <= ?", endDate)
	}

	query.Select("SUM(consume) as total_consume, COUNT(*) as record_count").Scan(&result)

	c.JSON(http.StatusOK, gin.H{
		"total_consume": result.TotalConsume,
		"total_tokens":  result.TotalTokens,
		"record_count":  result.RecordCount,
	})
}
