package controller

import (
	"codex-relay/internal/service"
	"codex-relay/pkg/entity"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GetDailyUsage 获取每日使用统计
// GET /api/usage/daily?days=7
func GetDailyUsage(c *gin.Context) {
	customerToken := c.GetHeader("Authorization")
	if customerToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	if strings.HasPrefix(customerToken, "Bearer ") {
		customerToken = strings.TrimPrefix(customerToken, "Bearer ")
	}

	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 7
	}

	usageService := service.NewUsageService()
	dailyUsages, err := usageService.GetDailyUsageByTokenForDays(customerToken, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get daily usage",
			"details": err.Error(),
		})
		return
	}
	if dailyUsages == nil {
		dailyUsages = make([]entity.DailyUsage, 0)
	}

	var totalConsume float64
	var totalRecords int64
	for _, daily := range dailyUsages {
		totalConsume += daily.TotalConsume
		totalRecords += daily.RecordCount
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"daily_usages": dailyUsages,
			"summary": gin.H{
				"total_consume": totalConsume,
				"total_records": totalRecords,
				"days":          len(dailyUsages),
			},
		},
	})
}

// GetDailyUsageByDateRange 获取指定日期范围的每日统计
// GET /api/usage/daily/range?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD
func GetDailyUsageByDateRange(c *gin.Context) {
	customerToken := c.GetHeader("Authorization")
	if customerToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	if strings.HasPrefix(customerToken, "Bearer ") {
		customerToken = strings.TrimPrefix(customerToken, "Bearer ")
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	startTime, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, should be YYYY-MM-DD"})
		return
	}
	endTime, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, should be YYYY-MM-DD"})
		return
	}

	// 标准化到整天范围
	startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())

	usageService := service.NewUsageService()
	dailyUsages, err := usageService.GetDailyUsageByToken(customerToken, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get daily usage",
			"details": err.Error(),
		})
		return
	}
	if dailyUsages == nil {
		dailyUsages = make([]entity.DailyUsage, 0)
	}

	var totalConsume float64
	var totalRecords int64
	for _, daily := range dailyUsages {
		totalConsume += daily.TotalConsume
		totalRecords += daily.RecordCount
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"daily_usages": dailyUsages,
			"summary": gin.H{
				"total_consume": totalConsume,
				"total_records": totalRecords,
				"days":          len(dailyUsages),
			},
		},
	})
}
