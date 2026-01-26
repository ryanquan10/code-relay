package mysql

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"codex-relay/config"
	"codex-relay/internal/logstream"
)

var db *gorm.DB

// InitFromConfig 从配置初始化 MySQL 连接
func InitFromConfig(cfg config.DatasourceConfig) error {
	dsn := convertJdbcUrlToGormDsn(cfg.URL, cfg.Username, cfg.Password)

	// 打印连接信息日志
	log.Printf("[MySQL] JDBC URL: %s", cfg.URL)
	log.Printf("[MySQL] Username: %s", cfg.Username)
	log.Printf("[MySQL] Generated DSN: %s", dsn)

	gormLogger := logger.New(
		log.New(io.MultiWriter(os.Stdout, logstream.Writer()), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      false,
			Colorful:                  false,
		},
	)

	gormConfig := &gorm.Config{
		Logger: gormLogger,
	}

	var err error
	db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.Hikari.MinimumIdle)
	sqlDB.SetMaxOpenConns(cfg.Hikari.MaximumPoolSize)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Hikari.MaxLifetime) * time.Millisecond)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Hikari.IdleTimeout) * time.Millisecond)

	return nil
}

// DB 返回数据库实例
func DB() *gorm.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// convertJdbcUrlToGormDsn 将 JDBC URL 转换为 GORM DSN
// 例如: jdbc:mysql://host:port/dbname?params -> user:password@tcp(host:port)/dbname?params
func convertJdbcUrlToGormDsn(jdbcUrl, username, password string) string {
	// 移除 "jdbc:mysql://" 前缀
	dsn := strings.TrimPrefix(jdbcUrl, "jdbc:mysql://")

	// 分离主机、端口、数据库和参数
	hostPort := dsn
	dbName := ""
	params := ""

	// 分离路径和查询参数
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		params = dsn[idx+1:]
		dsn = dsn[:idx]
	}

	// 分离主机和数据库名
	if idx := strings.Index(dsn, "/"); idx >= 0 {
		hostPort = dsn[:idx]
		dbName = dsn[idx+1:]
	}

	// URL encode username and password to handle special characters
	encodedUsername := url.QueryEscape(username)
	encodedPassword := url.QueryEscape(password)

	// 构建 GORM DSN: user:password@tcp(host:port)/dbname?params
	if dbName == "" && params == "" {
		return fmt.Sprintf("%s:%s@tcp(%s)", encodedUsername, encodedPassword, hostPort)
	}
	if params == "" {
		return fmt.Sprintf("%s:%s@tcp(%s)/%s", encodedUsername, encodedPassword, hostPort, dbName)
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", encodedUsername, encodedPassword, hostPort, dbName, params)
}
