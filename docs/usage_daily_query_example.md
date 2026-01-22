# Usage 按天分组查询使用示例

## 功能说明

新增了按天（GROUP BY DATE）统计用户使用量的功能，可以方便地查看每天的消费情况。

## 数据结构

### DailyUsage 结构体

```go
type DailyUsage struct {
    Date         string  `json:"date"`          // 日期格式: 2006-01-02
    TotalConsume float64 `json:"total_consume"` // 当天总消费
    RecordCount  int64   `json:"record_count"`  // 记录条数
}
```

## 使用方法

### 1. 获取指定日期范围的每日统计

```go
import (
    "codex-relay/internal/service"
    "time"
)

usageService := service.NewUsageService()

// 设置日期范围（例如：最近 30 天）
endTime := time.Now()
startTime := endTime.AddDate(0, 0, -30)

// 查询指定 token 的每日使用统计
dailyUsages, err := usageService.GetDailyUsageByToken(customerToken, startTime, endTime)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}

// 遍历结果
for _, daily := range dailyUsages {
    fmt.Printf("日期: %s, 消费: %.4f 元, 请求次数: %d\n",
        daily.Date, daily.TotalConsume, daily.RecordCount)
}
```

### 2. 获取最近 N 天的统计

```go
usageService := service.NewUsageService()

// 获取最近 7 天的统计
dailyUsages, err := usageService.GetDailyUsageByTokenForDays(customerToken, 7)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}

// 输出结果
for _, daily := range dailyUsages {
    fmt.Printf("%s: %.4f 元 (%d 次)\n",
        daily.Date, daily.TotalConsume, daily.RecordCount)
}
```

### 3. 管理员查询所有账户的每日统计

```go
usageService := service.NewUsageService()

// 查询所有账户在指定日期范围的统计
startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
endTime := time.Date(2026, 1, 31, 23, 59, 59, 0, time.Local)

dailyUsages, err := usageService.GetAllDailyUsage(startTime, endTime)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}

// 计算总消费
var totalConsume float64
for _, daily := range dailyUsages {
    totalConsume += daily.TotalConsume
    fmt.Printf("%s: %.2f 元\n", daily.Date, daily.TotalConsume)
}
fmt.Printf("总计: %.2f 元\n", totalConsume)
```

## API 接口示例

### 创建 Handler 示例

```go
package handler

import (
    "codex-relay/internal/service"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
)

// GetDailyUsage 获取每日使用统计
// GET /api/usage/daily?days=7
func GetDailyUsage(c *gin.Context) {
    // 从 Header 获取 token
    customerToken := c.GetHeader("Authorization")
    if customerToken == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
        return
    }

    // 去掉 "Bearer " 前缀
    if len(customerToken) > 7 && customerToken[:7] == "Bearer " {
        customerToken = customerToken[7:]
    }

    // 获取查询天数参数
    daysStr := c.DefaultQuery("days", "7")
    days, err := strconv.Atoi(daysStr)
    if err != nil || days <= 0 {
        days = 7
    }

    // 查询使用统计
    usageService := service.NewUsageService()
    dailyUsages, err := usageService.GetDailyUsageByTokenForDays(customerToken, days)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "failed to get daily usage",
            "details": err.Error(),
        })
        return
    }

    // 计算总消费
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
                "days": len(dailyUsages),
            },
        },
    })
}

// GetDailyUsageByDateRange 获取指定日期范围的每日统计
// GET /api/usage/daily/range?start_date=2026-01-01&end_date=2026-01-31
func GetDailyUsageByDateRange(c *gin.Context) {
    // 从 Header 获取 token
    customerToken := c.GetHeader("Authorization")
    if customerToken == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
        return
    }

    // 去掉 "Bearer " 前缀
    if len(customerToken) > 7 && customerToken[:7] == "Bearer " {
        customerToken = customerToken[7:]
    }

    // 解析日期参数
    startDateStr := c.Query("start_date")
    endDateStr := c.Query("end_date")

    if startDateStr == "" || endDateStr == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "start_date and end_date are required",
        })
        return
    }

    startTime, err := time.Parse("2006-01-02", startDateStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid start_date format, should be YYYY-MM-DD",
        })
        return
    }

    endTime, err := time.Parse("2006-01-02", endDateStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid end_date format, should be YYYY-MM-DD",
        })
        return
    }

    // 设置到当天结束
    endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())

    // 查询使用统计
    usageService := service.NewUsageService()
    dailyUsages, err := usageService.GetDailyUsageByToken(customerToken, startTime, endTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "failed to get daily usage",
            "details": err.Error(),
        })
        return
    }

    // 计算总消费
    var totalConsume float64
    for _, daily := range dailyUsages {
        totalConsume += daily.TotalConsume
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "daily_usages": dailyUsages,
            "start_date": startDateStr,
            "end_date": endDateStr,
            "total_consume": totalConsume,
        },
    })
}
```

## 响应示例

### 成功响应

```json
{
    "success": true,
    "data": {
        "daily_usages": [
            {
                "date": "2026-01-22",
                "total_consume": 0.1523,
                "record_count": 45
            },
            {
                "date": "2026-01-21",
                "total_consume": 0.2341,
                "record_count": 67
            },
            {
                "date": "2026-01-20",
                "total_consume": 0.0987,
                "record_count": 23
            }
        ],
        "summary": {
            "total_consume": 0.4851,
            "total_records": 135,
            "days": 3
        }
    }
}
```

## SQL 查询示例

生成的 SQL 查询类似于：

```sql
SELECT
    DATE(create_time) as date,
    SUM(consume) as total_consume,
    COUNT(*) as record_count
FROM usage
WHERE
    account_id = ?
    AND create_time >= ?
    AND create_time < ?
GROUP BY DATE(create_time)
ORDER BY date DESC
```

## 性能优化建议

1. **索引优化**：确保 `usage` 表有 `(account_id, create_time)` 的复合索引
   ```sql
   CREATE INDEX idx_account_time ON usage(account_id, create_time);
   ```

2. **查询范围限制**：建议限制查询的日期范围（例如最多查询 90 天）

3. **缓存**：对于历史数据，可以考虑使用 Redis 缓存查询结果

## 注意事项

1. `DATE(create_time)` 按照数据库服务器的时区进行日期计算
2. 返回的日期格式为 `YYYY-MM-DD`
3. 结果按日期降序排列（最新的日期在前）
4. 如果某天没有使用记录，则不会返回该天的数据
