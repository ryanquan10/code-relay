package controller

import (
	"codex-relay/internal/mysql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ListAccountSourceTypesPublic 返回去重后的 AccountSource.SourceType 列表（公开接口）
func ListAccountSourceTypesPublic(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var types []string
	// 仅返回启用中的来源类型，并去重
	if err := db.Table("account_source").
		Distinct("source_type").
		Where("status = ?", 1).
		Order("source_type ASC").
		Pluck("source_type", &types).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"types": types,
	})
}

// PublicListUsagesByToken 根据 token 与可选的 source_type 返回使用流水（公开接口）
// GET /api/public/usages?token=...&source_type=...&start_date=YYYY-MM-DD&end_date=YYYY-MM-DD
func PublicListUsagesByToken(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	sourceType := c.Query("source_type")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 结果行，仅包含所需字段
	type row struct {
		Consume    float64   `json:"consume"`
		Tokens     uint64    `json:"tokens"`
		CreateTime time.Time `json:"create_time"`
		UpdateTime time.Time `json:"update_time"`
	}

	q := db.Table("usage").
		Select("usage.consume, usage.tokens, usage.create_time, usage.update_time").
		Joins("LEFT JOIN account ON account.id = usage.account_id").
		Joins("LEFT JOIN account_source ON account.source_id = account_source.id").
		Where("account.token = ?", token)

	if sourceType != "" {
		q = q.Where("account_source.source_type = ?", sourceType)
	}

	// 可选的日期范围（按 create_time 过滤，包含端点天）
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			q = q.Where("usage.create_time >= ?", start)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			q = q.Where("usage.create_time <= ?", end)
		}
	}

	q = q.Order("usage.create_time DESC").Limit(1000) // 避免一次性返回过多

	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": rows})
}
