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

	// 初始化 MySQL 数据库
	log.Println("Initializing MySQL connection...")
	if err := mysql.InitFromConfig(cfg.Spring.Datasource); err != nil {
		log.Fatalf("init mysql: %v", err)
	}
	defer func() {
		_ = mysql.Close()
	}()
	log.Println("MySQL connection established")

	// 初始化 Redis
	log.Println("Initializing Redis connection...")
	if err := redis.InitFromConfig(cfg.Spring.Redis); err != nil {
		log.Fatalf("init redis: %v", err)
	}
	defer func() {
		_ = redis.Close()
	}()
	log.Println("Redis connection established")

	// 创建 context 用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动流量统计消费者
	consumer := service.NewTokenUsageConsumer("usage-group", "consumer-1")
	if err := consumer.InitConsumerGroup(); err != nil {
		log.Printf("⚠ 初始化消费者组失败: %v (如果是首次运行可忽略)", err)
	}

	// 在后台启动消费者
	go func() {
		log.Println("🚀 流量统计消费者已启动，开始处理 Redis Stream 消息...")
		consumer.StartConsuming(ctx)
	}()

	// 处理优雅关闭信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("📡 收到关闭信号，正在停止消费者...")
		cancel()
	}()

	srv := server.New(cfg, FrontendFS)
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
