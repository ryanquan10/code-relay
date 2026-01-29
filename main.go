package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"codex-relay/config"
	"codex-relay/internal/ipwatch"
	"codex-relay/internal/mysql"
	"codex-relay/internal/redis"
	"codex-relay/internal/server"
	"codex-relay/internal/service"
	"codex-relay/internal/task"
)

// buildJdbcURL replaces the host in a jdbc:mysql:// URL while preserving db and params.
func buildJdbcURL(jdbcURL, host string) string {
	if !strings.HasPrefix(jdbcURL, "jdbc:mysql://") {
		return jdbcURL
	}
	u := strings.TrimPrefix(jdbcURL, "jdbc:mysql://")
	params := ""
	if i := strings.Index(u, "?"); i >= 0 {
		params = u[i:]
		u = u[:i]
	}
	db := ""
	if i := strings.Index(u, "/"); i >= 0 {
		db = u[i+1:]
	}
	if db == "" {
		db = "codex_rental"
	}
	return fmt.Sprintf("jdbc:mysql://%s:3306/%s%s", host, db, params)
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "config file path (YAML)")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 设置全局配置并播报初始 IP（可能来自 env 覆盖）
	config.SetConfig(cfg)
	ipwatch.SetCurrentIP(cfg.Internal.Host)

	// 创建 context 用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 先启动 HTTP 服务器，保证能接收 nslookup 的 /update-ip 推送
	srv := server.New(cfg, FrontendFS)
	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 阻塞等待一个可用 IP（非空且非 0.0.0.0），再初始化后端连接
	if ip, ok := ipwatch.WaitForUsable(ctx); ok {
		cfg.Internal.Host = ip

		// Redis 按新 IP 初始化
		redisCfg := cfg.Spring.Redis
		redisCfg.Host = ip
		log.Println("Initializing Redis connection after IP ready...")
		if err := redis.InitFromConfig(redisCfg); err != nil {
			log.Printf("init redis (after IP) non-fatal: %v", err)
		} else {
			log.Println("Redis connection established")
		}
		defer func() { _ = redis.Close() }()

		// MySQL 按新 IP 初始化（保留原参数和库名）
		log.Println("Initializing MySQL connection after IP ready...")
		jdbc := buildJdbcURL(cfg.Spring.Datasource.URL, ip)
		mysqlCfg := cfg.Spring.Datasource
		mysqlCfg.URL = jdbc
		if err := mysql.InitFromConfig(mysqlCfg); err != nil {
			log.Printf("init mysql (after IP) non-fatal: %v", err)
		} else {
			log.Println("MySQL connection established")
		}
		defer func() { _ = mysql.Close() }()
	} else {
		log.Println("No usable IP received; skipping backend init")
	}

	// 启动流量统计消费者（等待 Redis 就绪后再启动）
	consumer := service.NewTokenUsageConsumer("usage-group", "consumer-1")
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			cli := redis.Client()
			if cli == nil {
				time.Sleep(1 * time.Second)
				continue
			}
			if err := cli.Ping(ctx).Err(); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			if err := consumer.InitConsumerGroup(); err != nil {
				log.Printf("Init consumer group failed: %v", err)
			}
			log.Println("🚀 流量统计消费者已启动，开始处理 Redis Stream 消息...")
			consumer.StartConsuming(ctx)
			return
		}
	}()

	// 启动用户追踪清理任务（每30秒清理一次）
	cleanupTask := task.NewUserTrackingCleanupTask(30 * time.Second)
	go cleanupTask.Start()

	// 处理优雅关闭信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("📡 收到关闭信号，开始优雅关闭...")
	cancel()
	_ = srv.Stop()
	log.Println("✓ 程序已完全退出")
}
