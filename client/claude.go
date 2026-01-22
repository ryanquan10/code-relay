package client

import (
	"fmt"
	"log"
	"net/http"
)

type claudeRelay struct{}

func NewClaudeRelay() *claudeRelay {
	return &claudeRelay{}
}

// GetAppType 返回客户端类型
func (c *claudeRelay) GetAppType() string {
	return "claude"
}

// HandleRequest 处理HTTP请求并转发到上游
func (c *claudeRelay) HandleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Claude] 收到请求: %s %s", r.Method, r.URL.Path)

	// TODO: 实现 Claude 的具体转发逻辑
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Claude relay endpoint", "path": "%s"}`, r.URL.Path)
}

func (c *claudeRelay) Relay() {
	log.Println("Claude Relay started")
	// Claude 中继逻辑
	select {}
}

// 导出全局变量
var Claude = NewClaudeRelay()

func init() {
	// 自动注册到全局注册表
	Register(Claude)
}
