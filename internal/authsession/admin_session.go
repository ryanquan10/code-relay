package authsession

import (
	"sync"
	"time"
)

type adminSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

var store = &adminSessionStore{
	sessions: make(map[string]time.Time),
}

// Save 记录管理员会话及过期时间。
func Save(token string, expiresAt time.Time) {
	if token == "" {
		return
	}
	store.mu.Lock()
	store.sessions[token] = expiresAt
	store.mu.Unlock()
}

// Delete 删除管理员会话。
func Delete(token string) {
	if token == "" {
		return
	}
	store.mu.Lock()
	delete(store.sessions, token)
	store.mu.Unlock()
}

// IsValid 检查会话是否有效；过期会自动清理。
func IsValid(token string, now time.Time) bool {
	if token == "" {
		return false
	}
	store.mu.RLock()
	expiresAt, ok := store.sessions[token]
	store.mu.RUnlock()
	if !ok {
		return false
	}
	if now.After(expiresAt) {
		Delete(token)
		return false
	}
	return true
}
