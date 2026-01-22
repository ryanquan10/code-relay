package client

import (
	"net/http"
	"sync"
)

// RelayClient 定义中继客户端的接口
type RelayClient interface {
	HandleRequest(w http.ResponseWriter, r *http.Request)
	GetAppType() string
}

// Registry 客户端注册表
type Registry struct {
	clients map[string]RelayClient
	mu      sync.RWMutex
}

var globalRegistry = &Registry{
	clients: make(map[string]RelayClient),
}

// Register 注册一个中继客户端
func Register(client RelayClient) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	appType := client.GetAppType()
	globalRegistry.clients[appType] = client
}

// GetClient 根据 AppType 获取客户端
func GetClient(appType string) (RelayClient, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	client, exists := globalRegistry.clients[appType]
	return client, exists
}

// GetAllClients 获取所有已注册的客户端
func GetAllClients() map[string]RelayClient {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make(map[string]RelayClient)
	for k, v := range globalRegistry.clients {
		result[k] = v
	}
	return result
}
