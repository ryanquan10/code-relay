package common

import (
	"codex-relay/internal/service"
	"log"
	"sync"
)

type RelayConfig struct {
	UpstreamURL string
	ProxyURL    string
	LogHeaders  bool
}

type ContextKey string

const (
	CustomerTokenContextKey      ContextKey = "customerToken"
	UpstreamConfigContextKey     ContextKey = "upstreamConfig"
	CodexUsageSessionContextKey  ContextKey = "codexUsageSession"
	ClaudeUsageSessionContextKey ContextKey = "claudeUsageSession"
)

var (
	CustomerCounters = make(map[string]*service.TrafficCounter)
	CountersMutex    sync.RWMutex
)

func GetOrCreateCounter(customerToken string) *service.TrafficCounter {
	CountersMutex.RLock()
	counter, exists := CustomerCounters[customerToken]
	CountersMutex.RUnlock()
	if exists {
		return counter
	}
	CountersMutex.Lock()
	defer CountersMutex.Unlock()
	if counter, exists := CustomerCounters[customerToken]; exists {
		return counter
	}
	counter = service.NewTrafficCounter(customerToken)
	CustomerCounters[customerToken] = counter
	log.Printf("[计数器] 为用户 %s 创建新的流量计数器", MaskKey(customerToken))
	return counter
}

func MaskKey(key string) string {
	if len(key) <= 20 {
		return "***"
	}
	return key[:15] + "..." + key[len(key)-8:]
}
