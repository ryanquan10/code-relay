package controller

import (
	"codex-relay/config"
	ipwatch "codex-relay/internal/ipwatch"
	"codex-relay/internal/mysql"
	"codex-relay/internal/redis"
	"codex-relay/internal/service"
	"codex-relay/pkg/entity"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// IPUpdateRequest IP 更新请求
type IPUpdateRequest struct {
	IP        string `json:"ip" binding:"required"`
	Timestamp int64  `json:"timestamp"`
}

// UpdateIP 更新本地 IP 地址（动态 DDNS）
func UpdateIP(c *gin.Context) {
	// 验证 Token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "缺少认证信息",
		})
		return
	}

	// 支持 "Bearer token" 格式
	token := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// 验证 Token
	cfg := config.GetConfig()
	if cfg == nil || cfg.Internal.NSLookupToken == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "server_error",
			"message": "服务器配置错误",
		})
		return
	}

	if token != cfg.Internal.NSLookupToken {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "认证失败",
		})
		return
	}

	// 解析请求
	var req IPUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	log.Printf("📡 收到 IP 更新请求: %s", req.IP)
	ipwatch.SetCurrentIP(req.IP)

	oldIP := cfg.Internal.Host
	if req.IP == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "IP 不能为空",
		})
		return
	}

	// 如果 IP 未变化，但后端不健康，则按上传 IP 强制重连
	if req.IP == oldIP {
		redisHealthy := testRedisConnection()
		mysqlHealthy := testMySQLConnection()
		if redisHealthy && mysqlHealthy {
			log.Printf("✅ IP 未变化且后端健康: %s，跳过重连", req.IP)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "IP 未变化，后端健康，无需更新",
				"data": gin.H{
					"ip":            req.IP,
					"redis_updated": false,
					"mysql_updated": false,
					"action":        "noop",
				},
			})
			return
		}
		log.Printf("🔧 IP 未变化但后端不健康，按上传 IP 强制重连")
	} else {
		log.Printf("🔄 检测到 IP 变化: %s -> %s，开始重连 Redis/MySQL", oldIP, req.IP)
	}

	// 构造新的连接参数
	newRedisHost := req.IP
	database := extractDatabase(cfg.Spring.Datasource.URL)
	newJdbcURL := fmt.Sprintf("jdbc:mysql://%s:3306/%s?charset=utf8mb4&parseTime=True&loc=Local", req.IP, database)

	// 重连 Redis
	if err := reconnectRedis(newRedisHost, cfg.Spring.Redis.Port, cfg.Spring.Redis.Password, cfg.Spring.Redis.DB); err != nil {
		log.Printf("❌ Redis 重连失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reconnect_failed",
			"message": "Redis 重连失败: " + err.Error(),
		})
		return
	}
	log.Printf("✅ Redis 重连成功: %s:%d", newRedisHost, cfg.Spring.Redis.Port)

	// 重连 MySQL
	if err := reconnectMySQL(newJdbcURL); err != nil {
		log.Printf("❌ MySQL 重连失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reconnect_failed",
			"message": "MySQL 重连失败: " + err.Error(),
		})
		return
	}
	log.Printf("✅ MySQL 重连成功: %s:3306", req.IP)

	// 更新内存中的配置，便于后续使用
	cfg.Internal.Host = req.IP
	cfg.Spring.Redis.Host = req.IP
	cfg.Spring.Datasource.URL = newJdbcURL

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "IP 更新并重连成功",
		"data": gin.H{
			"ip":            req.IP,
			"redis_updated": true,
			"mysql_updated": true,
			"action":        "updated",
		},
	})
}

// reconnectRedis 重新连接 Redis
func reconnectRedis(host string, port int, password string, db int) error {
	// 关闭旧连接
	if err := redis.Close(); err != nil {
		log.Printf("⚠️  关闭旧 Redis 连接失败: %v", err)
	}

	// 创建新连接
	newConfig := config.RedisConfig{
		Host:     host,
		Port:     port,
		Password: password,
		DB:       db,
	}

	return redis.InitFromConfig(newConfig)
}

// reconnectMySQL 重新连接 MySQL（参数是 JDBC URL）
func reconnectMySQL(jdbcURL string) error {
	// 关闭旧连接
	if err := mysql.Close(); err != nil {
		log.Printf("⚠️  关闭旧 MySQL 连接失败: %v", err)
	}

	// 创建新连接
	newConfig := config.DatasourceConfig{
		DriverClassName: "com.mysql.cj.jdbc.Driver",
		URL:             jdbcURL,
		Username:        config.GetConfig().Spring.Datasource.Username,
		Password:        config.GetConfig().Spring.Datasource.Password,
		Hikari:          config.GetConfig().Spring.Datasource.Hikari,
	}

	if err := mysql.InitFromConfig(newConfig); err != nil {
		return err
	}

	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("mysql db is nil after reconnect")
	}

	// 重连后补做迁移，避免因运行时重连导致新连接缺少表结构（例如 pricing）
	if err := db.AutoMigrate(
		&entity.AccountSource{},
		&entity.Product{},
		&entity.Pricing{},
		&entity.AccountSourceProcut{},
		&entity.Account{},
		&entity.Usage{},
		&entity.UpstreamErrorLog{},
	); err != nil {
		return fmt.Errorf("failed to automigrate after mysql reconnect: %w", err)
	}

	// 补齐 pricing 缺省数据（按 product.account_type）
	if err := service.NewPricingService().EnsureDefaultsFromProducts(); err != nil {
		return fmt.Errorf("failed to init pricing defaults after mysql reconnect: %w", err)
	}

	return nil
}

// extractDatabase 从 JDBC URL 中提取数据库名
func extractDatabase(jdbcURL string) string {
	// jdbc:mysql://host:3306/database?params
	parts := strings.Split(jdbcURL, "/")
	if len(parts) < 4 {
		return "codex_rental" // 默认数据库名
	}
	dbPart := parts[len(parts)-1]
	// 去掉参数
	if idx := strings.Index(dbPart, "?"); idx != -1 {
		dbPart = dbPart[:idx]
	}
	return dbPart
}

// testMySQLConnection 测试 MySQL 连接是否正常
func testMySQLConnection() bool {
	db := mysql.DB()
	if db == nil {
		return false
	}

	sqlDB, err := db.DB()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Printf("⚠️  MySQL 健康检查失败: %v", err)
		return false
	}

	return true
}

// testRedisConnection 测试 Redis 连接是否正常
func testRedisConnection() bool {
	client := redis.Client()
	if client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis 健康检查失败: %v", err)
		return false
	}

	return true
}
