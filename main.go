package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"codex-relay/config"
	"codex-relay/internal/mysql"
	"codex-relay/internal/redis"
	"codex-relay/internal/server"
	"codex-relay/internal/service"

	"time"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "config file path (YAML)")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 设置全局配置
	config.SetConfig(cfg)

	// 创建 context 用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 优先启动 HTTP 服务器，保证能接收 nslookup 的 /update-ip 推送
	srv := server.New(cfg, FrontendFS)
	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 非致命初始化 MySQL
	log.Println("Initializing MySQL connection...")
	if err := mysql.InitFromConfig(cfg.Spring.Datasource); err != nil {
		log.Printf("init mysql (non-fatal): %v", err)
	} else {
		log.Println("MySQL connection established")
	}
	defer func() { _ = mysql.Close() }()

	// 非致命初始化 Redis
	log.Println("Initializing Redis connection...")
	if err := redis.InitFromConfig(cfg.Spring.Redis); err != nil {
		log.Printf("init redis (non-fatal): %v", err)
	} else {
		log.Println("Redis connection established")
	}
	defer func() { _ = redis.Close() }()

	// 启动流量统计消费者（等待 Redis 就绪后再启动）
	consumer := service.NewTokenUsageConsumer("usage-group", "consumer-1")
	if err := consumer.InitConsumerGroup(); err != nil {
		log.Printf("⚠ 初始化消费者组失败: %v (如果是首次运行或 Redis 未就绪可忽略)", err)
	}
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
			log.Println("🚀 流量统计消费者已启动，开始处理 Redis Stream 消息...")
			if err := consumer.InitConsumerGroup(); err != nil {
				log.Printf("Init consumer group failed: %v", err)
			}
			consumer.StartConsuming(ctx)
			return
		}
	}()

	// 处理优雅关闭信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("📡 收到关闭信号，开始优雅关闭...")
	cancel()
	_ = srv.Stop()
	log.Println("✓ 程序已完全退出")
}
