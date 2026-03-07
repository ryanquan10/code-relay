package controller

import (
	"codex-relay/config"
	"codex-relay/internal/authsession"
	"codex-relay/internal/redis"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LoginRequest 登录请求体
type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应体
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
}

// User 用户信息
type User struct {
	Role     string `json:"role"`
	Username string `json:"username"`
}

// 生成随机 token
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Login 管理员登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{
			Success: false,
			Message: "请求参数错误",
		})
		return
	}

	// 从配置中获取管理员密码
	cfg := config.GetConfig()
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, LoginResponse{
			Success: false,
			Message: "服务器配置错误",
		})
		return
	}

	// 验证密码
	if req.Password != cfg.Admin.Password {
		c.JSON(http.StatusUnauthorized, LoginResponse{
			Success: false,
			Message: "密码错误",
		})
		return
	}

	// 生成 token
	token := generateToken()

	// 将 token 存储到 Redis，设置过期时间 24 小时
	expireTTL := 24 * time.Hour
	client := redis.Client()
	ctx := redis.Context()
	tokenKey := "admin_token:" + token
	if err := client.Set(ctx, tokenKey, "admin", expireTTL).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, LoginResponse{
			Success: false,
			Message: "Token 生成失败",
		})
		return
	}
	// 进程内兜底会话，避免 Redis 重连瞬时导致已登录用户被踢下线。
	authsession.Save(token, time.Now().Add(expireTTL))

	// 登录成功
	c.JSON(http.StatusOK, LoginResponse{
		Success: true,
		Message: "登录成功",
		Token:   token,
		User: &User{
			Role:     "admin",
			Username: "admin",
		},
	})
}

// Logout 登出
func Logout(c *gin.Context) {
	// 从请求头获取 token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "登出成功",
		})
		return
	}

	// 支持 "Bearer token" 格式
	token := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// 从 Redis 删除 token
	client := redis.Client()
	ctx := redis.Context()
	tokenKey := "admin_token:" + token
	client.Del(ctx, tokenKey)
	authsession.Delete(token)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登出成功",
	})
}
