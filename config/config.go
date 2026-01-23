package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

// 全局配置实例
var globalConfig *Config

// SetConfig 设置全局配置
func SetConfig(cfg Config) {
	globalConfig = &cfg
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return globalConfig
}

type Config struct {
	Internal InternalConfig `mapstructure:"internal"`
	Server   ServerConfig   `mapstructure:"server"`
	Spring   SpringConfig   `mapstructure:"spring"`
	Admin    AdminConfig    `mapstructure:"admin"`
	NSLookup NSLookupConfig `mapstructure:"nslookup"`
}

type AdminConfig struct {
	Password string `mapstructure:"password"`
}

type NSLookupConfig struct {
	CheckInterval int    `mapstructure:"check_interval"` // 检查间隔（秒）
	RemoteURL     string `mapstructure:"remote_url"`     // 远程服务器地址
	AuthToken     string `mapstructure:"auth_token"`     // 认证 Token
}

type InternalConfig struct {
	Host          string `mapstructure:"host"`
	NSLookupToken string `mapstructure:"nslookup_token"`
}

type ServerConfig struct {
	Port    int           `mapstructure:"port"`
	Servlet ServletConfig `mapstructure:"servlet"`
}

type ServletConfig struct {
	ContextPath string `mapstructure:"context-path"`
}

type SpringConfig struct {
	Application ApplicationConfig `mapstructure:"application"`
	Datasource  DatasourceConfig  `mapstructure:"datasource"`
	Redis       RedisConfig       `mapstructure:"redis"`
}

type ApplicationConfig struct {
	Name string `mapstructure:"name"`
}

type DatasourceConfig struct {
	DriverClassName string       `mapstructure:"driver-class-name"`
	URL             string       `mapstructure:"url"`
	Username        string       `mapstructure:"username"`
	Password        string       `mapstructure:"password"`
	Hikari          HikariConfig `mapstructure:"hikari"`
}

type HikariConfig struct {
	MinimumIdle       int    `mapstructure:"minimum-idle"`
	MaximumPoolSize   int    `mapstructure:"maximum-pool-size"`
	AutoCommit        bool   `mapstructure:"auto-commit"`
	IdleTimeout       int    `mapstructure:"idle-timeout"`
	PoolName          string `mapstructure:"pool-name"`
	MaxLifetime       int    `mapstructure:"max-lifetime"`
	ConnectionTimeout int    `mapstructure:"connection-timeout"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func Default() Config {
	return Config{
		Internal: InternalConfig{
			Host:          "192.168.3.176",
			NSLookupToken: "nslookup-ddns-2026",
		},
		Server: ServerConfig{
			Port: 8082,
			Servlet: ServletConfig{
				ContextPath: "/api",
			},
		},
		Spring: SpringConfig{
			Application: ApplicationConfig{
				Name: "account-rental-backend",
			},
			Datasource: DatasourceConfig{
				DriverClassName: "com.mysql.cj.jdbc.Driver",
				URL:             "jdbc:mysql://${internal.host}:3306/codex_rental?useUnicode=true&characterEncoding=utf8&useSSL=false&serverTimezone=Asia/Shanghai&allowPublicKeyRetrieval=true",
				Username:        "admin",
				Password:        "aaop77439la",
				Hikari: HikariConfig{
					MinimumIdle:       5,
					MaximumPoolSize:   40,
					AutoCommit:        true,
					IdleTimeout:       30000,
					PoolName:          "AccountRentalHikariCP",
					MaxLifetime:       1800000,
					ConnectionTimeout: 30000,
				},
			},
			Redis: RedisConfig{
				Host:     "a806698083.ticp.io",
				Port:     6379,
				Password: "31897197%12312A",
				DB:       0,
			},
		},
		Admin: AdminConfig{
			Password: "admin1237788",
		},
		NSLookup: NSLookupConfig{
			CheckInterval: 60,
			RemoteURL:     "http://keySwift.top/api/internal/update-ip",
			AuthToken:     "nslookup-ddns-2026",
		},
	}
}

func DefaultPath() string {
	return filepath.Join("config", "app.yaml")
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
			path = envPath
		} else {
			path = DefaultPath()
		}
	}

	v := viper.New()
	v.SetConfigType("yaml")
	applyDefaults(v, cfg)

	if data, err := os.ReadFile(path); err == nil {
		expanded := expandEnv(data)
		if err := v.ReadConfig(bytes.NewReader(expanded)); err != nil {
			return cfg, fmt.Errorf("read config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("read config file: %w", err)
	}

	bindEnvs(v)
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Spring.Datasource.URL = resolveInternalHost(cfg.Spring.Datasource.URL, cfg.Internal.Host)
	return cfg, nil
}

func applyDefaults(v *viper.Viper, cfg Config) {
	v.SetDefault("internal.host", cfg.Internal.Host)
	v.SetDefault("server.port", cfg.Server.Port)
	v.SetDefault("server.servlet.context-path", cfg.Server.Servlet.ContextPath)
	v.SetDefault("spring.application.name", cfg.Spring.Application.Name)
	v.SetDefault("spring.datasource.driver-class-name", cfg.Spring.Datasource.DriverClassName)
	v.SetDefault("spring.datasource.url", cfg.Spring.Datasource.URL)
	v.SetDefault("spring.datasource.username", cfg.Spring.Datasource.Username)
	v.SetDefault("spring.datasource.password", cfg.Spring.Datasource.Password)
	v.SetDefault("spring.datasource.hikari.minimum-idle", cfg.Spring.Datasource.Hikari.MinimumIdle)
	v.SetDefault("spring.datasource.hikari.maximum-pool-size", cfg.Spring.Datasource.Hikari.MaximumPoolSize)
	v.SetDefault("spring.datasource.hikari.auto-commit", cfg.Spring.Datasource.Hikari.AutoCommit)
	v.SetDefault("spring.datasource.hikari.idle-timeout", cfg.Spring.Datasource.Hikari.IdleTimeout)
	v.SetDefault("spring.datasource.hikari.pool-name", cfg.Spring.Datasource.Hikari.PoolName)
	v.SetDefault("spring.datasource.hikari.max-lifetime", cfg.Spring.Datasource.Hikari.MaxLifetime)
	v.SetDefault("spring.datasource.hikari.connection-timeout", cfg.Spring.Datasource.Hikari.ConnectionTimeout)
	v.SetDefault("spring.redis.host", cfg.Spring.Redis.Host)
	v.SetDefault("spring.redis.port", cfg.Spring.Redis.Port)
	v.SetDefault("spring.redis.password", cfg.Spring.Redis.Password)
	v.SetDefault("spring.redis.db", cfg.Spring.Redis.DB)
	v.SetDefault("admin.password", cfg.Admin.Password)
}

func bindEnvs(v *viper.Viper) {
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("internal.host", "INTERNAL_HOST")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("server.servlet.context-path", "SERVER_CONTEXT_PATH")
	_ = v.BindEnv("spring.application.name", "SPRING_APPLICATION_NAME")
	_ = v.BindEnv("spring.datasource.url", "SPRING_DATASOURCE_URL")
	_ = v.BindEnv("spring.datasource.username", "SPRING_DATASOURCE_USERNAME")
	_ = v.BindEnv("spring.datasource.password", "SPRING_DATASOURCE_PASSWORD")
	_ = v.BindEnv("spring.redis.host", "SPRING_REDIS_HOST", "REDIS_HOST")
	_ = v.BindEnv("spring.redis.port", "SPRING_REDIS_PORT", "REDIS_PORT")
	_ = v.BindEnv("spring.redis.password", "SPRING_REDIS_PASSWORD", "REDIS_PASSWORD")
	_ = v.BindEnv("spring.redis.db", "SPRING_REDIS_DB", "REDIS_DB")
}

var envPattern = regexp.MustCompile(`\$\{([A-Z0-9_]+)(:([^}]*))?\}`)

func expandEnv(input []byte) []byte {
	return envPattern.ReplaceAllFunc(input, func(match []byte) []byte {
		parts := envPattern.FindSubmatch(match)
		key := string(parts[1])
		def := ""
		if len(parts) > 3 {
			def = string(parts[3])
		}
		if val, ok := os.LookupEnv(key); ok {
			return []byte(val)
		}
		return []byte(def)
	})
}

func resolveInternalHost(input, host string) string {
	if input == "" || host == "" {
		return input
	}
	return strings.ReplaceAll(input, "${internal.host}", host)
}
