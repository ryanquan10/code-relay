package controller

import (
	"codex-relay/config"
	"codex-relay/internal/mysql"
	"codex-relay/internal/redis"
	"fmt"
	"log"
	"net/http"
	"strings"

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

	// 更新配置
	newRedisHost := req.IP
	newMySQLHost := req.IP

	log.Printf("📡 收到 IP 更新请求: %s", req.IP)
	log.Printf("🔄 开始更新数据库连接...")

	// 更新 Redis 连接
	if err := reconnectRedis(newRedisHost, cfg.Spring.Redis.Port, cfg.Spring.Redis.Password, cfg.Spring.Redis.DB); err != nil {
		log.Printf("❌ Redis 重连失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reconnect_failed",
			"message": "Redis 重连失败: " + err.Error(),
		})
		return
	}
	log.Printf("✅ Redis 重连成功: %s:%d", newRedisHost, cfg.Spring.Redis.Port)

	// 更新 MySQL 连接
	newDSN := buildMySQLDSN(
		newMySQLHost,
		cfg.Spring.Datasource.Username,
		cfg.Spring.Datasource.Password,
		extractDatabase(cfg.Spring.Datasource.URL),
	)

	if err := reconnectMySQL(newDSN); err != nil {
		log.Printf("❌ MySQL 重连失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reconnect_failed",
			"message": "MySQL 重连失败: " + err.Error(),
		})
		return
	}
	log.Printf("✅ MySQL 重连成功: %s:3306", newMySQLHost)

	// 更新配置中的 host
	cfg.Spring.Redis.Host = newRedisHost
	cfg.Internal.Host = newMySQLHost

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "IP 更新成功",
		"data": gin.H{
			"ip":         req.IP,
			"redis_host": newRedisHost,
			"mysql_host": newMySQLHost,
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

// reconnectMySQL 重新连接 MySQL
func reconnectMySQL(dsn string) error {
	// 关闭旧连接
	if err := mysql.Close(); err != nil {
		log.Printf("⚠️  关闭旧 MySQL 连接失败: %v", err)
	}

	// 创建新连接
	newConfig := config.DatasourceConfig{
		DriverClassName: "com.mysql.cj.jdbc.Driver",
		URL:             dsn,
		Username:        config.GetConfig().Spring.Datasource.Username,
		Password:        config.GetConfig().Spring.Datasource.Password,
		Hikari:          config.GetConfig().Spring.Datasource.Hikari,
	}

	return mysql.InitFromConfig(newConfig)
}

// buildMySQLDSN 构建 MySQL DSN
func buildMySQLDSN(host, username, password, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, database)
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
