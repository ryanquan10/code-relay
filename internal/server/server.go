package server

import (
	"codex-relay/config"
	"codex-relay/internal/controller"
	"embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

type Server struct {
	engine *gin.Engine
	addr   string
}

func New(cfg config.Config, frontendFS embed.FS) *Server {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	// CORS 中间件
	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 路由分组
	api := &engine.RouterGroup
	if base := controller.NormalizeBasePath(cfg.Server.Servlet.ContextPath); base != "" {
		api = engine.Group(base)
	}

	internal := api.Group("/internal")
	{
		internal.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"service": cfg.Spring.Application.Name,
			})
		})

		internal.GET("/usage", controller.Usage)
	}

	admin := api.Group("/admin")
	{
		// 货源管理
		admin.GET("/sources", controller.ListSources)
		admin.POST("/sources", controller.CreateSource)
		admin.PUT("/sources/:id", controller.UpdateSource)
		admin.DELETE("/sources/:id", controller.DeleteSource)

		// 账号管理
		admin.GET("/accounts", controller.ListAccounts)
		admin.GET("/accounts/token/:token", controller.GetAccountByToken)
		admin.POST("/accounts", controller.CreateAccount)
		admin.POST("/accounts/batch", controller.BatchCreateAccounts)
		admin.PUT("/accounts/:id/balance", controller.UpdateAccountBalance)
		admin.DELETE("/accounts/:id", controller.DeleteAccount)

		// 使用量查看
		admin.GET("/usage", controller.ListUsages)
		admin.POST("/usage/query", controller.QueryUsageByToken)
		admin.GET("/usage/stats", controller.GetUsageStats)
	}

	// ──────────────────────────────────────────────
	//           前端静态文件嵌入（使用根包的 FrontendFS）
	// ──────────────────────────────────────────────

	embeddedFS, err := static.EmbedFolder(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("创建嵌入文件系统失败: %v", err)
	}

	engine.Use(static.Serve("/", embeddedFS))

	// SPA 支持：未匹配路由回退到 index.html（保护 API 路径）
	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/admin/") ||
			strings.HasPrefix(path, "/internal/") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "resource not found",
			})
			return
		}

		c.FileFromFS("frontend/dist/index.html", http.FS(frontendFS))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	return &Server{
		engine: engine,
		addr:   addr,
	}
}

func (s *Server) Run() error {
	log.Printf("internal server listening on %s", s.addr)
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return srv.ListenAndServe()
}
