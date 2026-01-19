package redis

import (
	"context"
	"fmt"
	"log"

	redislib "github.com/redis/go-redis/v9"
)

var (
	client *redislib.Client
	ctx    = context.Background()
)

func Init(addr, password string, db int) error {
	client = redislib.NewClient(&redislib.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if _, err := client.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("无法连接到 Redis: %w", err)
	}

	log.Printf("Redis 连接成功: %s (DB: %d)", addr, db)
	return nil
}

func Client() *redislib.Client {
	return client
}

func Context() context.Context {
	return ctx
}
