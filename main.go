package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"pysio.online/Files-API/internal/config"
	"pysio.online/Files-API/internal/handler"
	"pysio.online/Files-API/internal/logger"
	"pysio.online/Files-API/internal/middleware"
	"pysio.online/Files-API/internal/service"
)

// 同步任务
type syncTask struct {
	repo         *config.Repository
	gitService   *service.GitService
	minioService *service.MinioService
}

func syncWorker(taskChan <-chan syncTask) {
	for task := range taskChan {
		// 获取仓库的检查间隔
		interval := task.gitService.GetCheckInterval(task.repo)

		if err := task.gitService.SyncRepository(task.repo); err != nil {
			log.Printf("同步仓库失败 %s: %v", task.repo.URL, err)
			continue
		}

		// 传递检查间隔到 UploadDirectory
		if err := task.minioService.UploadDirectory(task.repo.LocalPath, task.repo.MinioPath, interval); err != nil {
			log.Printf("上传到Minio失败 %s: %v", task.repo.MinioPath, err)
		}
	}
}

func startSyncWorkers(numWorkers int) chan<- syncTask {
	taskChan := make(chan syncTask)
	for i := 0; i < numWorkers; i++ {
		go syncWorker(taskChan)
	}
	return taskChan
}

func main() {
	// 解析命令行参数
	flags := middleware.ParseFlags()

	// 显示帮助信息
	if flags.Help {
		flag.Usage()
		return
	}

	// 处理日志压缩/解压命令
	if flags.ZipLogs {
		if err := middleware.CompressLogs("logs"); err != nil {
			log.Fatal(err)
		}
		return
	}

	if flags.UnzipLogs {
		if err := middleware.UncompressLogs("logs"); err != nil {
			log.Fatal(err)
		}
		return
	}

	// 处理清理命令
	if flags.ClearAll || flags.ClearLogs {
		if err := middleware.ClearLogs("logs"); err != nil {
			log.Fatal(err)
		}
	}

	if flags.ClearAll || flags.ClearCache {
		if err := middleware.ClearCache(".cache"); err != nil {
			log.Fatal(err)
		}
	}

	if flags.ClearAll || flags.ClearLogs || flags.ClearCache {
		return
	}

	// 加载配置文件
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化日志系统
	logManager, err := logger.New(&cfg.Logs)
	if err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	defer logManager.Close()

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

	// 处理单次同步命令
	if flags.Sync {
		log.Printf("执行单次同步检查...")
		for _, repo := range cfg.Git.Repositories {
			log.Printf("同步仓库: %s", repo.URL)
			if err := gitService.SyncRepository(&repo); err != nil {
				log.Printf("同步仓库失败 %s: %v", repo.URL, err)
				continue
			}
			if err := minioService.UploadDirectory(repo.LocalPath, repo.MinioPath, 0); err != nil {
				log.Printf("上传到Minio失败 %s: %v", repo.MinioPath, err)
			}
		}
		log.Printf("单次同步检查完成")
		return
	}

	// 新增：处理 rsync 命令
	if flags.RSync != "" {
		log.Printf("执行指定仓库同步: %s", flags.RSync)
		found := false
		for _, repo := range cfg.Git.Repositories {
			if repo.MinioPath == flags.RSync {
				found = true
				log.Printf("同步仓库: %s", repo.URL)
				if err := gitService.SyncRepository(&repo); err != nil {
					log.Printf("同步仓库失败 %s: %v", repo.URL, err)
					os.Exit(1)
				}
				if err := minioService.UploadDirectory(repo.LocalPath, repo.MinioPath, 0); err != nil {
					log.Printf("上传到Minio失败 %s: %v", repo.MinioPath, err)
					os.Exit(1)
				}
				log.Printf("指定仓库同步完成: %s", repo.MinioPath)
				os.Exit(0)
			}
		}
		if !found {
			log.Printf("未找到指定的仓库: %s", flags.RSync)
			os.Exit(1)
		}
	}

	// 仅在非 API-only 模式时启动自动同步任务
	if !cfg.Server.APIOnly {
		// 启动同步工作池
		taskChan := startSyncWorkers(2) // 使用2个工作线程

		if !flags.Skip {
			log.Printf("开始初始同步...")
			for _, repo := range cfg.Git.Repositories {
				taskChan <- syncTask{
					repo:         &repo,
					gitService:   gitService,
					minioService: minioService,
				}
			}
		} else {
			log.Printf("已跳过初始同步，等待下一个检查周期...")
			for _, repo := range cfg.Git.Repositories {
				minioService.InitLastSync(repo.MinioPath)
			}
		}

		// 定时同步任务
		go func() {
			ticker := time.NewTicker(10 * time.Minute)
			for range ticker.C {
				log.Printf("开始定时同步...")
				for _, repo := range cfg.Git.Repositories {
					log.Printf("正在等待同步仓库: %s", repo.URL)
					taskChan <- syncTask{
						repo:         &repo,
						gitService:   gitService,
						minioService: minioService,
					}
					log.Printf("已添加同步任务: %s", repo.URL)
				}
				log.Printf("已添加所有同步任务到队列")
			}
		}()
	} else {
		log.Printf("API-only 模式，跳过文件同步任务")
	}

	// 在路由注册前初始化缓存中间件
	cacheMiddleware, err := middleware.NewCacheMiddleware(&cfg.Cache)
	if err != nil {
		log.Fatalf("初始化缓存中间件失败: %v", err)
	}

	// 设置路由
	if cfg.Server.EnableAPI {
		apiHandler := handler.NewAPIHandler(minioService, cfg)
		http.Handle("/api/files/", cacheMiddleware.Middleware(apiHandler))
		log.Printf("API 服务已启用: /api/files/")
	}

	if !cfg.Server.APIOnly {
		docsHandler := handler.NewDocsHandler(minioService, cfg)
		http.Handle("/", cacheMiddleware.Middleware(docsHandler))
		log.Printf("文件服务已启用: /")
	}

	// 两个服务都未启用时退出
	if !cfg.Server.EnableAPI && cfg.Server.APIOnly {
		log.Fatal("错误: API 和文件服务都未启用")
	}

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
