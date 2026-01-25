package database

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 数据库配置
type Config struct {
	URL      string
	Username string
	Password string
	Hikari   HikariConfig
}

type HikariConfig struct {
	MinimumIdle       int
	MaximumPoolSize   int
	AutoCommit        bool
	IdleTimeout       int
	PoolName          string
	MaxLifetime       int
	ConnectionTimeout int
}

// InitMySQL 初始化 MySQL 连接
func InitMySQL(cfg Config) (*gorm.DB, error) {
	dsn := convertJdbcUrlToGormDsn(cfg.URL, cfg.Username, cfg.Password)

	fmt.Printf("[MySQL] JDBC URL: %s\n", cfg.URL)
	fmt.Printf("[MySQL] Username: %s\n", cfg.Username)
	fmt.Printf("[MySQL] Generated DSN: %s\n", dsn)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.Hikari.MinimumIdle)
	sqlDB.SetMaxOpenConns(cfg.Hikari.MaximumPoolSize)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Hikari.MaxLifetime) * time.Millisecond)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Hikari.IdleTimeout) * time.Millisecond)

	return db, nil
}

// convertJdbcUrlToGormDsn 将 JDBC URL 转换为 GORM DSN
func convertJdbcUrlToGormDsn(jdbcUrl, username, password string) string {
	dsn := strings.TrimPrefix(jdbcUrl, "jdbc:mysql://")

	hostPort := dsn
	dbName := ""
	params := ""

	if idx := strings.Index(dsn, "?"); idx >= 0 {
		params = dsn[idx+1:]
		dsn = dsn[:idx]
	}

	if idx := strings.Index(dsn, "/"); idx >= 0 {
		hostPort = dsn[:idx]
		dbName = dsn[idx+1:]
	}

	encodedUsername := url.QueryEscape(username)
	encodedPassword := url.QueryEscape(password)

	// 转换 JDBC 参数为 MySQL Go driver 参数
	params = convertJdbcParamsToMySQLParams(params)

	if dbName == "" && params == "" {
		return fmt.Sprintf("%s:%s@tcp(%s)", encodedUsername, encodedPassword, hostPort)
	}
	if params == "" {
		return fmt.Sprintf("%s:%s@tcp(%s)/%s", encodedUsername, encodedPassword, hostPort, dbName)
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", encodedUsername, encodedPassword, hostPort, dbName, params)
}

// convertJdbcParamsToMySQLParams 转换 JDBC 参数为 MySQL Go driver 参数
func convertJdbcParamsToMySQLParams(jdbcParams string) string {
	if jdbcParams == "" {
		return ""
	}

	// 解析参数
	params := make(map[string]string)
	for _, pair := range strings.Split(jdbcParams, "&") {
		if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}

	// 构建 MySQL driver 参数
	var result []string

	// 字符集转换
	if _, hasUnicode := params["useUnicode"]; hasUnicode {
		if encoding, hasEncoding := params["characterEncoding"]; hasEncoding {
			if encoding == "utf8" {
				result = append(result, "charset=utf8mb4")
			} else {
				result = append(result, fmt.Sprintf("charset=%s", encoding))
			}
		}
	}

	// 时区转换
	if tz, hasTZ := params["serverTimezone"]; hasTZ {
		result = append(result, "parseTime=True")
		result = append(result, fmt.Sprintf("loc=%s", url.QueryEscape(tz)))
	}

	// 处理 allowPublicKeyRetrieval（JDBC参数转换为Go driver参数）
	if val, exists := params["allowPublicKeyRetrieval"]; exists && val == "true" {
		result = append(result, "allowNativePasswords=true")
	}

	return strings.Join(result, "&")
}
