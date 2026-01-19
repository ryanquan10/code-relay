package main

import (
	"embed"
	"flag"
	"log"

	"codex-relay/config"
	"codex-relay/internal/redis"
	"codex-relay/internal/server"
)

var frontendFS embed.FS

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "config file path (YAML)")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := redis.InitFromConfig(cfg.Spring.Redis); err != nil {
		log.Fatalf("init redis: %v", err)
	}
	defer func() {
		_ = redis.Close()
	}()

	srv := server.New(cfg, frontendFS)
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
