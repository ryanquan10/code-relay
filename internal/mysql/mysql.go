package mysql

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"codex-relay/config"
)

var db *gorm.DB

// InitFromConfig 从配置初始化 MySQL 连接
func InitFromConfig(cfg config.DatasourceConfig) error {
	dsn := convertJdbcUrlToGormDsn(cfg.URL, cfg.Username, cfg.Password)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
	dsn := jdbcUrl
	if len(dsn) > 13 && dsn[:13] == "jdbc:mysql://" {
		dsn = dsn[13:]
	}

	// 构建 GORM DSN: user:password@tcp(host:port)/dbname?params
	return fmt.Sprintf("%s:%s@tcp(%s)", username, password, dsn)
}
