package redis

import (
	"fmt"
	"net"
	"strconv"

	"codex-relay/config"
)

func InitFromConfig(cfg config.RedisConfig) error {
	addr := cfg.Host
	if addr == "" {
		return fmt.Errorf("redis host is empty")
	}
	port := cfg.Port
	if port == 0 {
		port = 6379
	}
	addr = net.JoinHostPort(addr, strconv.Itoa(port))
	return Init(addr, cfg.Password, cfg.DB)
}

func Close() error {
	if client == nil {
		return nil
	}
	return client.Close()
}
