package server

import (
	"codex-relay/client"
	"codex-relay/config"
	"codex-relay/internal/controller"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	engine      *gin.Engine
	addr        string
	frontendCmd *exec.Cmd
}

const indexPath = "frontend/dist/index.html"

func serveIndex(c *gin.Context, frontendFS embed.FS) {
	data, err := frontendFS.ReadFile(indexPath)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func New(cfg config.Config, frontendFS embed.FS) *Server {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	// 禁用自动重定向
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false

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

	// ==================== API 路由 ====================
	api := engine.Group("/api")
	{
		// 内部健康检查
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

		// 认证路由
		auth := api.Group("/auth")
		{
			auth.POST("/login", controller.Login)
		}

		// 使用量接口（按天）
		usage := api.Group("/usage")
		{
			usage.GET("/daily", controller.GetDailyUsage)
			usage.GET("/daily/range", controller.GetDailyUsageByDateRange)
		}

		// 管理后台路由
		admin := api.Group("/admin")
		{
			// 货源管理
			admin.GET("/sources", controller.ListSources)
			admin.POST("/sources", controller.CreateSource)
			admin.PUT("/sources/:id", controller.UpdateSource)
			admin.DELETE("/sources/:id", controller.DeleteSource)

			// 产品管理
			admin.GET("/products", controller.ListProducts)
			admin.POST("/products", controller.CreateProduct)
			admin.PUT("/products/:id", controller.UpdateProduct)
			admin.DELETE("/products/:id", controller.DeleteProduct)

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
	}

	// ==================== Codex 中继路由 ====================
	// 动态路由：根据 URL 第一段匹配对应的 client
	// 例如: /codex/v1/xxx -> 匹配 AppType="codex" 的客户端
	engine.Any("/:appType/*path", func(c *gin.Context) {
		appType := c.Param("appType")

		// 跳过已知的路由前缀
		if appType == "api" || appType == "assets" {
			c.Next()
			return
		}

		// 根据 appType 查找对应的客户端
		relayClient, exists := client.GetClient(appType)
		if !exists {
			// 如果找不到对应的客户端，继续执行后续路由（最终会到 NoRoute）
			c.Next()
			return
		}

		// 使用找到的客户端处理请求
		log.Printf("[路由] 匹配到客户端类型: %s, 路径: %s", appType, c.Request.URL.Path)
		relayClient.HandleRequest(c.Writer, c.Request)
		c.Abort() // 终止后续处理
	})

	// ==================== 静态资源路由 ====================
	// 处理 /assets/* 下的静态资源
	engine.GET("/assets/*filepath", func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		// 安全检查：防止路径遍历攻击
		if strings.Contains(reqPath, "..") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		filePath := path.Join("frontend/dist", reqPath)
		c.FileFromFS(filePath, http.FS(frontendFS))
	})

	// ==================== SPA 路由 ====================
	// 所有非 API、非静态资源的请求都返回 index.html
	engine.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// 如果是 API 路径，返回 404 JSON
		if strings.HasPrefix(reqPath, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// 如果请求的是静态文件（根据扩展名判断），尝试提供文件
		if isStaticFile(reqPath) {
			filePath := path.Join("frontend/dist", strings.TrimPrefix(reqPath, "/"))
			// 尝试提供文件，如果失败 Gin 会自动返回 404
			c.FileFromFS(filePath, http.FS(frontendFS))
			return
		}

		// 其他所有请求返回 index.html，让前端路由处理
		serveIndex(c, frontendFS)
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &Server{
		engine: engine,
		addr:   addr,
	}

	if os.Getenv("DEV_MODE") == "true" {
		if err := srv.startFrontendDev(); err != nil {
			log.Printf("警告: 启动前端开发服务器失败: %v", err)
		}
	}

	return srv
}

// isStaticFile 检查路径是否是静态文件
func isStaticFile(reqPath string) bool {
	staticExtensions := []string{
		".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg",
		".ico", ".woff", ".woff2", ".ttf", ".eot", ".otf",
		".json", ".xml", ".txt", ".pdf",
	}

	for _, ext := range staticExtensions {
		if strings.HasSuffix(reqPath, ext) {
			return true
		}
	}
	return false
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

func (s *Server) startFrontendDev() error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("无法获取当前文件路径")
	}

	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	frontendDir := filepath.Join(projectRoot, "frontend")

	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		return fmt.Errorf("前端目录不存在: %s", frontendDir)
	}

	packageJSON := filepath.Join(frontendDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("package.json 不存在: %s", packageJSON)
	}

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

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("前端开发服务器已退出: %v", err)
		}
	}()

	return nil
}

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
