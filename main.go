package main

import (
	"flag"
	"log"

	"codex-relay/config"
	"codex-relay/internal/mysql"
	"codex-relay/internal/redis"
	"codex-relay/internal/server"
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

	srv := server.New(cfg, FrontendFS)
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
