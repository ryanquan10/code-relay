package server

import (
	"codex-relay/config"
	"codex-relay/internal/controller"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

type Server struct {
	engine *gin.Engine
	addr   string
}

func New(cfg config.Config) *Server {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	api := &engine.RouterGroup
	if base := controller.NormalizeBasePath(cfg.Server.Servlet.ContextPath); base != "" {
		api = engine.Group(base)
	}

	internal := api.Group("/internal")
	internal.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": cfg.Spring.Application.Name,
		})
	})

	internal.GET("/usage", controller.Usage)

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
