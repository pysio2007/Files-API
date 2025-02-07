package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"pysio.online/Files-API/internal/config"
	"pysio.online/Files-API/internal/service"
)

type ExternalURLMiddleware struct {
	minioService *service.MinioService
	config       *config.Config
	client       *http.Client
	sync.RWMutex
	lastCheck map[string]time.Time
}

func NewExternalURLMiddleware(minioService *service.MinioService, config *config.Config) *ExternalURLMiddleware {
	return &ExternalURLMiddleware{
		minioService: minioService,
		config:       config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		lastCheck: make(map[string]time.Time),
	}
}

func (m *ExternalURLMiddleware) downloadURL(ctx context.Context, urls []string) ([]byte, error) {
	var lastErr error

	// 尝试所有URL
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := m.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status code: %d", resp.StatusCode)
			continue
		}

		// 读取响应内容
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		return data, nil
	}

	return nil, fmt.Errorf("all URLs failed, last error: %v", lastErr)
}

func (m *ExternalURLMiddleware) shouldCheck(path string, interval time.Duration) bool {
	m.Lock()
	defer m.Unlock()

	lastCheck, exists := m.lastCheck[path]
	if !exists || time.Since(lastCheck) >= interval {
		m.lastCheck[path] = time.Now()
		return true
	}
	return false
}

func (m *ExternalURLMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否匹配外部URL配置
		var matchedURL *config.ExternalURL
		for _, eu := range m.config.ExternalURLs {
			if eu.Path == r.URL.Path {
				matchedURL = &eu
				break
			}
		}

		if matchedURL == nil {
			next.ServeHTTP(w, r)
			return
		}

		// 解析检查间隔
		interval, err := parseDuration(matchedURL.CheckInterval)
		if err != nil {
			interval = 1 * time.Hour
		}

		// 检查是否需要更新
		if m.shouldCheck(matchedURL.Path, interval) {
			// 构建URL列表,主URL在前
			urls := append([]string{matchedURL.MainURL}, matchedURL.BackupURLs...)

			// 尝试下载
			data, err := m.downloadURL(r.Context(), urls)
			if err != nil {
				log.Printf("下载失败 %s: %v", matchedURL.Path, err)
			} else {
				// 修改：使用正确的 Minio PutObject 选项
				_, err = m.minioService.PutObject(
					matchedURL.MinioPath,
					bytes.NewReader(data),
					int64(len(data)),
					map[string]string{
						"X-Amz-Meta-Cache-Control": matchedURL.CacheControl, // 修改：添加 X-Amz-Meta- 前缀
					},
				)
				if err != nil {
					log.Printf("上传到Minio失败 %s: %v", matchedURL.Path, err)
				} else {
					log.Printf("成功更新外部资源: %s", matchedURL.Path)
				}
			}
		}

		// 从Minio获取文件并返回
		obj, err := m.minioService.GetObject(matchedURL.MinioPath)
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
		w.Header().Set("Cache-Control", matchedURL.CacheControl)

		// 输出文件内容
		io.Copy(w, obj)
	})
}

// 添加：初始化时进行首次同步
func (m *ExternalURLMiddleware) Init() {
	for _, eu := range m.config.ExternalURLs {
		// 立即执行首次同步
		interval, _ := parseDuration(eu.CheckInterval)
		if interval > 0 {
			go func(url *config.ExternalURL) {
				urls := append([]string{url.MainURL}, url.BackupURLs...)
				data, err := m.downloadURL(context.Background(), urls)
				if err != nil {
					log.Printf("初始化外部URL失败 %s: %v", url.Path, err)
					return
				}
				_, err = m.minioService.PutObject(
					url.MinioPath,
					bytes.NewReader(data),
					int64(len(data)),
					map[string]string{
						"X-Amz-Meta-Cache-Control": url.CacheControl,
					},
				)
				if err != nil {
					log.Printf("初始化上传到Minio失败 %s: %v", url.Path, err)
				}
				m.lastCheck[url.Path] = time.Now()
			}(&eu)
		}
	}
}
