package middleware

import (
	"codex-relay/internal/redis"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "未登录或登录已过期，请先登录",
			})
			c.Abort()
			return
		}

		// 支持 "Bearer token" 格式
		token := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 验证 token 是否存在于 Redis
		client := redis.Client()
		ctx := redis.Context()
		tokenKey := "admin_token:" + token

		val, err := client.Get(ctx, tokenKey).Result()
		if err != nil || val == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "登录已过期，请重新登录",
			})
			c.Abort()
			return
		}

		// Token 有效，继续处理
		c.Set("user", val) // 将用户信息存入 context
		c.Next()
	}
}
