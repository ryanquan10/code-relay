package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"codex-relay/internal/logstream"
)

// ListConsoleLogs returns recent console log lines from an in-memory ring buffer.
func ListConsoleLogs(c *gin.Context) {
	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}
	lines := logstream.Recent(limit)
	c.JSON(http.StatusOK, gin.H{
		"lines": lines,
	})
}
