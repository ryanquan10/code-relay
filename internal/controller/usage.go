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
	tokens, err := service.GetUsageFromRedis(customerToken, dates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read redis"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customerToken": customerToken,
		"date":          dates,
		"tokens":        tokens,
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
