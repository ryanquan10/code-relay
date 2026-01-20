package server

import (
	"codex-relay/config"
	"codex-relay/internal/controller"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

type Server struct {
	engine      *gin.Engine
	addr        string
	frontendCmd *exec.Cmd
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
	srv := &Server{
		engine: engine,
		addr:   addr,
	}

	// 自动启动前端开发服务器
	if os.Getenv("DEV_MODE") == "true" {
		if err := srv.startFrontendDev(); err != nil {
			log.Printf("警告: 启动前端开发服务器失败: %v", err)
		}
	}

	return srv
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

// startFrontendDev 启动前端开发服务器
func (s *Server) startFrontendDev() error {
	// 获取项目根目录
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("无法获取当前文件路径")
	}

	// 从 internal/server/server.go 回退到项目根目录
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	frontendDir := filepath.Join(projectRoot, "frontend")

	// 检查前端目录是否存在
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		return fmt.Errorf("前端目录不存在: %s", frontendDir)
	}

	// 检查 package.json 是否存在
	packageJSON := filepath.Join(frontendDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("package.json 不存在: %s", packageJSON)
	}

	// 根据操作系统选择命令
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "npm run dev")
	} else {
		cmd = exec.Command("sh", "-c", "npm run dev")
	}

	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("正在启动前端开发服务器...")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动前端开发服务器失败: %w", err)
	}

	s.frontendCmd = cmd
	log.Printf("前端开发服务器已启动 (PID: %d)", cmd.Process.Pid)

	// 启动一个 goroutine 来监控前端进程
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("前端开发服务器已退出: %v", err)
		}
	}()

	return nil
}

// Stop 停止服务器和前端开发服务器
func (s *Server) Stop() error {
	if s.frontendCmd != nil && s.frontendCmd.Process != nil {
		log.Printf("正在停止前端开发服务器...")
		if err := s.frontendCmd.Process.Kill(); err != nil {
			return fmt.Errorf("停止前端开发服务器失败: %w", err)
		}
		log.Printf("前端开发服务器已停止")
	}
	return nil
}
