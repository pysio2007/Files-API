package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"pysio.online/Files-API/internal/config"
	"pysio.online/Files-API/internal/service"
)

type DocsHandler struct {
	minioService *service.MinioService
	config       *config.Config
}

func NewDocsHandler(minioService *service.MinioService, config *config.Config) *DocsHandler {
	return &DocsHandler{
		minioService: minioService,
		config:       config,
	}
}

func (h *DocsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 访问日志
	if h.config.Logs.AccessLog {
		log.Printf("Access: %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
	}

	// 处理旧版 API 格式
	if h.config.Server.LegacyAPI && strings.HasPrefix(r.URL.Path, "/files/") {
		// 移除 "/files/" 前缀
		newPath := strings.TrimPrefix(r.URL.Path, "/files/")
		if h.config.Logs.RedirectLog {
			log.Printf("Legacy API redirect: %s -> /%s", r.URL.Path, newPath)
		}
		http.Redirect(w, r, "/"+newPath, http.StatusMovedPermanently)
		return
	}

	// 移除前导斜杠
	filePath := strings.TrimPrefix(r.URL.Path, "/")
	if filePath == "" {
		http.Error(w, "无效的访问路径", http.StatusBadRequest)
		return
	}

	// 获取第一级路径
	parts := strings.SplitN(filePath, "/", 2)
	basePath := parts[0]
	remainingPath := ""
	if len(parts) > 1 {
		remainingPath = parts[1]
	}

	// 先检查是否匹配配置的桶
	var matchedBucket *config.BucketConfig
	for _, bucket := range h.config.Buckets {
		if bucket.Name == basePath {
			matchedBucket = &bucket
			break
		}
	}

	if matchedBucket != nil {
		// 处理匹配到的桶
		obj, err := h.minioService.GetObjectFromBucket(matchedBucket.Name, remainingPath)
		if err != nil {
			http.Error(w, "文件不存在", http.StatusNotFound)
			return
		}
		defer obj.Close()

		// 获取文件信息
		info, err := obj.Stat()
		if err != nil {
			http.Error(w, "获取文件信息失败", http.StatusInternalServerError)
			return
		}

		// 设置响应头
		w.Header().Set("Content-Type", info.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))

		// 输出文件内容
		if _, err := io.Copy(w, obj); err != nil {
			log.Printf("发送文件失败 %s: %v", remainingPath, err)
		}
		return
	}

	// 如果不是配置的桶,则按原有逻辑处理
	authorized := false
	// 检查Git仓库配置
	for _, repo := range h.config.Git.Repositories {
		if repo.MinioPath == basePath {
			authorized = true
			break
		}
	}

	// 检查暴露路径配置
	if !authorized {
		for _, exposed := range h.config.ExposedPaths {
			if exposed.MinioPath == basePath {
				authorized = true
				break
			}
		}
	}

	if !authorized {
		http.Error(w, "未授权的访问路径", http.StatusForbidden)
		return
	}

	// 使用Minio API的公共URL
	if h.config.Minio.UsePublicURL {
		publicURL := h.minioService.GetPublicURL(filePath)
		if publicURL != "" {
			if h.config.Logs.RedirectLog {
				log.Printf("Redirect: %s -> %s", r.URL.Path, publicURL)
			}
			http.Redirect(w, r, publicURL, http.StatusFound)
			return
		}
	}

	// 如果获取公共URL失败或未启用，则使用代理方式
	object, err := h.minioService.GetObject(filePath)
	if err != nil {
		log.Printf("获取文件失败 %s: %v", filePath, err)
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}
	defer object.Close()

	// 获取文件信息
	info, err := object.Stat()
	if err != nil {
		log.Printf("获取文件信息失败 %s: %v", filePath, err)
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	// 设置Content-Type和其他头信息
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))

	// 直接复制文件内容到响应
	if _, err := io.Copy(w, object); err != nil {
		log.Printf("发送文件失败 %s: %v", filePath, err)
	}
}
