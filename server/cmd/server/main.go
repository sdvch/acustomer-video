package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/handler"
	"acustomer-video-server/internal/repository"
	"acustomer-video-server/internal/service"
	"acustomer-video-server/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	if err := repository.Init(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer repository.Close()

	// 启动异步合成 worker（与 router 共用 OSS 单例）
	oss := service.NewOSSService(cfg)
	w := worker.New(cfg, oss)
	w.Start()
	defer w.Stop()

	r := handler.NewRouter(cfg)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("acustomer-video-server 启动 :%d (db=%s bucket=%s)", cfg.ServerPort, cfg.DBName, cfg.OSSBucket)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("收到退出信号，关闭服务...")
	_ = srv.Close()
}
