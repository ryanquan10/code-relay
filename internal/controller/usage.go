package controller

import (
	"codex-relay/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func Usage(c *gin.Context) {
	customerToken := strings.TrimSpace(c.Query("customerToken"))
	if customerToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customerToken is required"})
		return
	}

	dates := c.QueryArray("date") // 返回 []string
	for i, d := range dates {
		dates[i] = strings.TrimSpace(d)
	}
	if len(dates) == 0 {
		dates = []string{time.Now().Format("2006-01-02")} // 默认当前日期
	}

	// 从 MySQL 查询使用量（返回消费金额，单位：元）
	consume, err := service.GetUsageFromRedis(customerToken, dates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query usage", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customerToken": customerToken,
		"date":          dates,
		"consume":       consume, // 消费金额（元）
	})
}

func NormalizeBasePath(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "" || clean == "/" {
		return ""
	}
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return strings.TrimRight(clean, "/")
}
