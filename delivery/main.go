package main

import (
	"fmt"
	"log"
	"net/http"

	"codex-relay/config"
	"codex-relay/pkg/database"
	"delivery/internal/handler"
	"delivery/internal/repository"
	"delivery/internal/service"
)

func main() {
	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库连接
	dbCfg := database.Config{
		URL:      cfg.Spring.Datasource.URL,
		Username: cfg.Spring.Datasource.Username,
		Password: cfg.Spring.Datasource.Password,
		Hikari: database.HikariConfig{
			MinimumIdle:       cfg.Spring.Datasource.Hikari.MinimumIdle,
			MaximumPoolSize:   cfg.Spring.Datasource.Hikari.MaximumPoolSize,
			AutoCommit:        cfg.Spring.Datasource.Hikari.AutoCommit,
			IdleTimeout:       cfg.Spring.Datasource.Hikari.IdleTimeout,
			PoolName:          cfg.Spring.Datasource.Hikari.PoolName,
			MaxLifetime:       cfg.Spring.Datasource.Hikari.MaxLifetime,
			ConnectionTimeout: cfg.Spring.Datasource.Hikari.ConnectionTimeout,
		},
	}

	db, err := database.InitMySQL(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 初始化 repository
	deliveryRepo := repository.NewDeliveryRepository(db)

	// 初始化 service
	deliveryService := service.NewDeliveryService(deliveryRepo)

	// 初始化 handler
	deliveryHandler := handler.NewDeliveryHandler(deliveryService)

	// 设置路由
	http.HandleFunc("/api/delivery/account", deliveryHandler.GetAccount)

	// 启动服务器
	port := 8089 // 可以根据需要修改端口
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Delivery service starting on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
