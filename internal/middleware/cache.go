package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"pysio.online/Files-API/internal/config"
)

type CacheMiddleware struct {
	config     *config.CacheConfig
	cacheMutex sync.RWMutex
}

func NewCacheMiddleware(config *config.CacheConfig) (*CacheMiddleware, error) {
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %v", err)
	}

	cm := &CacheMiddleware{
		config: config,
	}

	// 启动定期清理过期缓存的goroutine
	go cm.cleanupRoutine()

	return cm, nil
}

// 生成缓存键
func (cm *CacheMiddleware) generateCacheKey(r *http.Request) string {
	h := sha256.New()
	io.WriteString(h, r.URL.Path)
	io.WriteString(h, r.URL.RawQuery)
	return hex.EncodeToString(h.Sum(nil))
}

// 获取缓存文件路径
func (cm *CacheMiddleware) getCachePath(key string) string {
	return filepath.Join(cm.config.Directory, key)
}

// 解析时间间隔
func parseDuration(duration string) (time.Duration, error) {
	if duration == "" {
		return 0, fmt.Errorf("empty duration")
	}

	unit := duration[len(duration)-1:]
	value, err := strconv.Atoi(duration[:len(duration)-1])
	if err != nil {
		return 0, err
	}

	switch strings.ToLower(unit) {
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "y":
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %s", unit)
	}
}

type cacheWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (cw *cacheWriter) WriteHeader(statusCode int) {
	cw.statusCode = statusCode
	cw.ResponseWriter.WriteHeader(statusCode)
}

func (cw *cacheWriter) Write(b []byte) (int, error) {
	cw.body = append(cw.body, b...)
	return cw.ResponseWriter.Write(b)
}

func (cm *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cm.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// 检查是否是API请求
		isAPIRequest := strings.HasPrefix(r.URL.Path, "/api/")

		key := cm.generateCacheKey(r)
		cachePath := cm.getCachePath(key)

		// 尝试从缓存读取
		if content, headers, ok := cm.getFromCache(cachePath); ok {
			// 设置原始响应头
			for k, v := range headers {
				w.Header().Set(k, v)
			}

			// 添加缓存控制头
			if isAPIRequest && cm.config.EnableAPICache {
				if duration, err := parseDuration(cm.config.APICacheControl); err == nil {
					w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(duration.Seconds())))
				}
			} else if cm.config.CacheControl != "" {
				if duration, err := parseDuration(cm.config.CacheControl); err == nil {
					w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(duration.Seconds())))
				}
			}

			if cm.config.HitLog {
				log.Printf("Cache hit: %s", r.URL.Path)
			}
			w.Write(content)
			return
		}

		// 包装响应写入器以捕获响应
		cw := &cacheWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(cw, r)

		// 仅缓存成功的响应
		if cw.statusCode == http.StatusOK {
			headers := make(map[string]string)
			for k, v := range w.Header() {
				headers[k] = v[0]
			}
			cm.saveToCache(cachePath, cw.body, headers)
		}
	})
}

func (cm *CacheMiddleware) getFromCache(path string) ([]byte, map[string]string, bool) {
	cm.cacheMutex.RLock()
	defer cm.cacheMutex.RUnlock()

	// 读取缓存元数据
	metaPath := path + ".meta"
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, false
	}

	// 解析元数据
	var headers map[string]string
	ttl, err := parseDuration(cm.config.TTL)
	if err != nil {
		log.Printf("解析缓存TTL失败: %v", err)
		return nil, nil, false
	}

	// 检查是否过期
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > ttl {
		if cm.config.CacheLog {
			log.Printf("Cache expired: %s", path)
		}
		return nil, nil, false
	}

	// 读取缓存内容
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false
	}

	if err := json.Unmarshal(metaData, &headers); err != nil {
		return nil, nil, false
	}

	return content, headers, true
}

func (cm *CacheMiddleware) saveToCache(path string, content []byte, headers map[string]string) {
	cm.cacheMutex.Lock()
	defer cm.cacheMutex.Unlock()

	if cm.config.CacheLog {
		log.Printf("Caching: %s", path)
	}

	// 保存内容
	if err := os.WriteFile(path, content, 0644); err != nil {
		log.Printf("缓存写入失败: %v", err)
		return
	}

	// 保存元数据
	metaData, err := json.Marshal(headers)
	if err != nil {
		log.Printf("元数据序列化失败: %v", err)
		return
	}

	metaPath := path + ".meta"
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		log.Printf("元数据写入失败: %v", err)
	}
}

func (cm *CacheMiddleware) cleanupRoutine() {
	ticker := time.NewTicker(6 * time.Hour)
	for range ticker.C {
		cm.cleanup()
	}
}

func (cm *CacheMiddleware) cleanup() {
	cm.cacheMutex.Lock()
	defer cm.cacheMutex.Unlock()

	var totalSize int64
	ttl, err := parseDuration(cm.config.TTL)
	if err != nil {
		log.Printf("解析缓存TTL失败: %v", err)
		return
	}

	cutoff := time.Now().Add(-ttl)

	// 遍历缓存目录
	err = filepath.Walk(cm.config.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if info.ModTime().Before(cutoff) {
				os.Remove(path)
				if cm.config.CacheLog {
					log.Printf("Removed expired cache: %s", path)
				}
			} else {
				totalSize += info.Size()
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("缓存清理失败: %v", err)
		return
	}

	// 检查缓存大小是否超过限制
	maxSize := int64(cm.config.MaxSize) * 1024 * 1024
	if totalSize > maxSize {
		if cm.config.CacheLog {
			log.Printf("Cache size exceeded limit: %d MB", cm.config.MaxSize)
		}
		// TODO: 实现基于LRU的缓存清理
	}
}
