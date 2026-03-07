// internal/controller/console.go
package controller

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"codex-relay/internal/logstream"
)

// AddConsoleLog 供其他模块调用（写入 logstream 的共享 writer）
func AddConsoleLog(line string) {
	_, _ = io.WriteString(logstream.Writer(), line+"\n")
}

// ListConsoleLogs Gin Handler
func ListConsoleLogs(c *gin.Context) {
	limit := 200
	//字符串转整数
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 2000 {
				n = 2000
			}
			limit = n
		}
	}

	lines := logstream.Recent(limit)
	c.JSON(http.StatusOK, gin.H{
		"lines": lines,
	})
}

// Register 在 main 或 router 初始化时统一注册
func Register(r *gin.Engine) {
	r.GET("/console/logs", ListConsoleLogs)
}
