package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"
)

// ListErrorLogs 获取上游错误日志（默认最多200条，按创建时间倒序）
func ListErrorLogs(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}

	var logs []entity.UpstreamErrorLog
	q := db.Model(&entity.UpstreamErrorLog{})

	// 可选过滤条件
	if v := c.Query("source_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			q = q.Where("source_id = ?", id)
		}
	}
	if v := c.Query("account_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			q = q.Where("account_id = ?", id)
		}
	}
	if v := c.Query("source_type"); v != "" {
		q = q.Where("source_type = ?", v)
	}
	if v := c.Query("since_hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			since := time.Now().Add(-time.Duration(n) * time.Hour)
			q = q.Where("created_at >= ?", since)
		}
	}

	if err := q.Order("created_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}
