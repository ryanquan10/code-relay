package controller

import (
	"codex-relay/config"
	"net/http"

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
	User    *User  `json:"user,omitempty"`
}

// User 用户信息
type User struct {
	Role     string `json:"role"`
	Username string `json:"username"`
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

	// 登录成功
	c.JSON(http.StatusOK, LoginResponse{
		Success: true,
		Message: "登录成功",
		User: &User{
			Role:     "admin",
			Username: "admin",
		},
	})
}
