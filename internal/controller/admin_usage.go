package controller

import (
	"codex-relay/internal/mysql"
	"codex-relay/internal/service"
	"codex-relay/pkg/entity"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ListUsages 获取使用量列表（包含 account.token 作为 customer_key）
func ListUsages(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	// 分页参数（默认每页100）
	var qp struct {
		Page int `form:"page"`
		Size int `form:"size"`
	}
	_ = c.ShouldBindQuery(&qp)
	page := qp.Page
	size := qp.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 100
	}
	if size > 500 {
		size = 500
	}
	offset := (page - 1) * size

	// 总数
	var total int64
	if err := db.Model(&entity.Usage{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 连接 account 以取出 token 作为 customer_key
	type row struct {
		ID         uint64    `json:"id"`
		AccountID  uint64    `json:"account_id"`
		Tokens     uint64    `json:"tokens"`
		Consume    float64   `json:"consume"`
		CreateTime time.Time `json:"create_time"`
		Token      *string   `json:"token"`
	}

	var rows []row
	if err := db.Table("usage").
		Select("usage.id, usage.account_id, usage.tokens, usage.consume, usage.create_time, account.token").
		Joins("LEFT JOIN account ON account.id = usage.account_id").
		Order("usage.create_time DESC").
		Limit(size).
		Offset(offset).
		Scan(&rows).Error; err != nil {
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

	resp := make([]UsageResponse, 0, len(rows))
	for _, r := range rows {
		ck := "N/A"
		if r.Token != nil && *r.Token != "" {
			ck = *r.Token
		}
		resp = append(resp, UsageResponse{
			ID:          r.ID,
			UserID:      0,
			AccountID:   r.AccountID,
			CustomerKey: ck,
			Tokens:      r.Tokens,
			Consume:     r.Consume,
			Date:        r.CreateTime.Format("2006-01-02"),
			CreatedAt:   r.CreateTime.Format("2006-01-02 15:04:05"),
		})
	}

	// 返回分页结构
	c.JSON(http.StatusOK, gin.H{
		"items":       resp,
		"page":        page,
		"size":        size,
		"total":       total,
		"total_pages": (total + int64(size) - 1) / int64(size),
	})
}

// QueryUsageByToken 根据 customer_key（即 account.token）查询使用量
func QueryUsageByToken(c *gin.Context) {
	type reqBody struct {
		CustomerKey string   `json:"customer_key"`
		Dates       []string `json:"dates"`
	}
	var req reqBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CustomerKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_key is required"})
		return
	}

	us := service.NewUsageService()

	// 计算查询范围与统计
	var (
		start    time.Time
		end      time.Time
		useRange bool
	)
	if len(req.Dates) > 0 {
		useRange = true
		// 取 dates 的最小/最大天作为范围
		first := true
		for _, ds := range req.Dates {
			if ds == "" { // 跳过空
				continue
			}
			d, err := time.Parse("2006-01-02", ds)
			if err != nil {
				continue
			}
			if first {
				start = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
				end = start.Add(24 * time.Hour)
				first = false
			} else {
				curStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
				curEnd := curStart.Add(24 * time.Hour)
				if curStart.Before(start) {
					start = curStart
				}
				if curEnd.After(end) {
					end = curEnd
				}
			}
		}
		if first { // 全部非法
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dates"})
			return
		}
	}

	var total float64
	var list []entity.Usage
	var err error

	if useRange {
		// 列表按范围取，统计按逐日精确累加
		list, err = us.GetUsageListByTokenAndDateRange(req.CustomerKey, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total, err = us.GetUsageByDates(req.CustomerKey, req.Dates)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// 默认今天
		// 统计
		total, err = us.GetTodayUsageByToken(req.CustomerKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 列表
		today := time.Now()
		start = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
		end = start.Add(24 * time.Hour)
		list, err = us.GetUsageListByTokenAndDateRange(req.CustomerKey, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 组装响应
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

	details := make([]UsageResponse, 0, len(list))
	for _, u := range list {
		details = append(details, UsageResponse{
			ID:          u.ID,
			UserID:      0,
			AccountID:   u.AccountID,
			CustomerKey: req.CustomerKey,
			Tokens:      u.TOKENS,
			Consume:     u.Consume,
			Date:        u.CreateTime.Format("2006-01-02"),
			CreatedAt:   u.CreateTime.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_consume": total,
		"details":       details,
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

	query.Select("COALESCE(SUM(consume),0) as total_consume, COALESCE(SUM(tokens),0) as total_tokens, COUNT(*) as record_count").Scan(&result)

	c.JSON(http.StatusOK, gin.H{
		"total_consume": result.TotalConsume,
		"total_tokens":  result.TotalTokens,
		"record_count":  result.RecordCount,
	})
}
