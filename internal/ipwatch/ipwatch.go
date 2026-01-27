package ipwatch

import (
	"context"
	"log"
	"strings"
	"sync"
)

var (
	mu        sync.RWMutex
	currentIP string
	ch        = make(chan string, 4)
)

// SetCurrentIP sets the current IP; if it changes, it broadcasts to waiters.
func SetCurrentIP(ip string) {
	ip = strings.TrimSpace(ip)
	mu.Lock()
	defer mu.Unlock()
	if ip == currentIP {
		return
	}
	old := currentIP
	currentIP = ip
	select {
	case ch <- ip:
		if old != "" {
			log.Printf("[IPWatch] IP updated: %s -> %s", old, ip)
		} else {
			log.Printf("[IPWatch] IP set: %s", ip)
		}
	default:
		// channel full; drop broadcast to avoid blocking
	}
}

// Current returns the current IP.
func Current() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentIP
}

// WaitForUsable blocks until a non-empty, non-0.0.0.0 IP is available or ctx canceled.
func WaitForUsable(ctx context.Context) (string, bool) {
	isBad := func(ip string) bool { return ip == "" || ip == "0.0.0.0" }
	if ip := Current(); !isBad(ip) {
		return ip, true
	}
	log.Printf("[IPWatch] Waiting for IP from nslookup...")
	for {
		select {
		case <-ctx.Done():
			return "", false
		case ip := <-ch:
			if !isBad(ip) {
				return ip, true
			}
		}
	}
}
