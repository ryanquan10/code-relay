package middleware

import (
	"codex-relay/internal/authsession"
	"codex-relay/internal/redis"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
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
		token := strings.TrimSpace(authHeader)
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "登录已过期，请重新登录",
			})
			c.Abort()
			return
		}

		// 验证 token 是否存在于 Redis
		client := redis.Client()
		if client == nil {
			// Redis 暂不可用时，走进程内会话兜底，避免误判为 401 导致前端强制退出。
			if authsession.IsValid(token, time.Now()) {
				c.Set("user", "admin")
				c.Next()
				return
			}
			log.Printf("[AuthMiddleware] redis client nil and fallback session invalid: path=%s", c.Request.URL.Path)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "auth_unavailable",
				"message": "认证服务暂不可用，请稍后重试",
			})
			c.Abort()
			return
		}
		ctx := redis.Context()
		tokenKey := "admin_token:" + token

		val, err := client.Get(ctx, tokenKey).Result()
		switch {
		case err == nil && val != "":
			// 滑动续期，活跃会话自动延长。
			const sessionTTL = 24 * time.Hour
			_ = client.Expire(ctx, tokenKey, sessionTTL).Err()
			authsession.Save(token, time.Now().Add(sessionTTL))
		case errors.Is(err, goredis.Nil) || val == "":
			// Redis 中 key 不存在时，尝试本地会话兜底并回写 Redis。
			if authsession.IsValid(token, time.Now()) {
				const sessionTTL = 24 * time.Hour
				_ = client.Set(ctx, tokenKey, "admin", sessionTTL).Err()
				val = "admin"
				break
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "登录已过期，请重新登录",
			})
			c.Abort()
			return
		default:
			// Redis 异常不应映射成 401，避免前端误清 token。
			log.Printf("[AuthMiddleware] redis get failed: path=%s err=%v", c.Request.URL.Path, err)
			if authsession.IsValid(token, time.Now()) {
				val = "admin"
				break
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "auth_unavailable",
				"message": "认证服务暂不可用，请稍后重试",
			})
			c.Abort()
			return
		}

		if val == "" {
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
