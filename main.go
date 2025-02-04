package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"pysio.online/Files-API/internal/config"
	"pysio.online/Files-API/internal/handler"
	"pysio.online/Files-API/internal/service"
)

// 执行Git同步
func syncGitRepositories(gitService *service.GitService, minioService *service.MinioService, cfg *config.Config) {
	for _, repo := range cfg.Git.Repositories {
		if err := gitService.SyncRepository(&repo); err != nil {
			log.Printf("同步仓库失败 %s: %v", repo.URL, err)
			continue
		}
		if err := minioService.UploadDirectory(repo.LocalPath, repo.MinioPath); err != nil {
			log.Printf("上传到Minio失败 %s: %v", repo.MinioPath, err)
		}
	}
}

func main() {
	log.Printf("正在加载配置文件...")
	cfg, err := config.LoadConfig("config.yaml") // 改为使用根目录的配置文件
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}
	log.Printf("配置文件加载成功")

	// 初始化服务
	minioService, err := service.NewMinioService(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 检查Minio连通性
	if err := minioService.CheckConnection(); err != nil {
		log.Fatalf("Minio服务器检查失败: %v", err)
	}
	log.Printf("Minio服务器连接正常")

	gitService := service.NewGitService(cfg)
	docsHandler := handler.NewDocsHandler(minioService, cfg)

	// 启动时立即执行一次同步
	log.Printf("开始初始同步...")
	syncGitRepositories(gitService, minioService, cfg)
	log.Printf("初始同步完成")

	// 设置定时同步 (10分钟)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			log.Printf("开始定时同步...")
			syncGitRepositories(gitService, minioService, cfg)
			log.Printf("定时同步完成")
		}
	}()

	// 设置路由 - 直接在根路径处理请求
	http.Handle("/", docsHandler)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
