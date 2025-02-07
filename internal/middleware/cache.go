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
	"sort"
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
	case "s": // 新增：支持秒单位
		return time.Duration(value) * time.Second, nil
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

func (cm *CacheMiddleware) shouldCache(path string) bool {
	// 检查是否是 API 请求
	if strings.HasPrefix(path, "/api/") {
		// 如果 API 缓存未启用，直接返回 false
		if !cm.config.EnableAPICache {
			return false
		}

		// 检查是否在例外列表中
		for _, excludePath := range cm.config.APIExcludePaths {
			if strings.HasPrefix(path, excludePath) {
				return false
			}
		}
	}
	return true
}

func (cm *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cm.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// 检查是否应该缓存这个请求
		if !cm.shouldCache(r.URL.Path) {
			if cm.config.CacheLog {
				log.Printf("Skip caching for path: %s", r.URL.Path)
			}
			next.ServeHTTP(w, r)
			return
		}

		// 检查是否是API请求（用于设置不同的缓存时间）
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
			if isAPIRequest {
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

// 检查缓存是否过期
func (cm *CacheMiddleware) isExpired(path string, isAPI bool) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return true, err
	}

	var ttl time.Duration
	if isAPI {
		// API 使用 apiCacheControl 作为 TTL
		ttl, err = parseDuration(cm.config.APICacheControl)
	} else {
		// 文件使用 cacheControl 作为 TTL
		ttl, err = parseDuration(cm.config.CacheControl)
	}
	if err != nil {
		return true, err
	}

	return time.Since(info.ModTime()) > ttl, nil
}

// 删除过期的缓存文件
func (cm *CacheMiddleware) removeExpiredCache(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("删除过期缓存文件失败 %s: %v", path, err)
	}
	metaPath := path + ".meta"
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		log.Printf("删除过期缓存元数据失败 %s: %v", metaPath, err)
	}
}

func (cm *CacheMiddleware) getFromCache(path string) ([]byte, map[string]string, bool) {
	cm.cacheMutex.RLock()
	defer cm.cacheMutex.RUnlock()

	// 判断是否是 API 请求
	isAPI := strings.Contains(path, "/api/")

	// 检查是否过期
	expired, err := cm.isExpired(path, isAPI)
	if err != nil || expired {
		if expired {
			// 立即删除过期缓存
			go cm.removeExpiredCache(path)
		}
		return nil, nil, false
	}

	// 读取缓存元数据
	metaPath := path + ".meta"
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, false
	}

	// 读取缓存内容
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false
	}

	var headers map[string]string
	if err := json.Unmarshal(metaData, &headers); err != nil {
		return nil, nil, false
	}

	// 更新访问时间
	now := time.Now()
	os.Chtimes(path, now, now)
	os.Chtimes(metaPath, now, now)

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

func (cm *CacheMiddleware) cleanup() {
	cm.cacheMutex.Lock()
	defer cm.cacheMutex.Unlock()

	var totalSize int64
	// 遍历缓存目录
	err := filepath.Walk(cm.config.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && !strings.HasSuffix(path, ".meta") {
			isAPI := strings.Contains(path, "/api/")
			expired, err := cm.isExpired(path, isAPI)
			if err != nil || expired {
				// 删除过期文件
				cm.removeExpiredCache(path)
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

	// 检查缓存大小
	maxSize := int64(cm.config.MaxSize) * 1024 * 1024
	if totalSize > maxSize {
		if err := cm.cleanupLRU(totalSize, maxSize); err != nil {
			log.Printf("LRU缓存清理失败: %v", err)
		}
	}
}

// LRU缓存清理
func (cm *CacheMiddleware) cleanupLRU(totalSize, maxSize int64) error {
	var items []cacheItem

	// 收集所有缓存项信息
	err := filepath.Walk(cm.config.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(path, ".meta") {
			items = append(items, cacheItem{
				path:     path,
				size:     info.Size(),
				lastUsed: info.ModTime(),
				metaPath: path + ".meta",
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk cache directory failed: %v", err)
	}

	// 按最后访问时间排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].lastUsed.Before(items[j].lastUsed)
	})

	// 从最旧的开始删除，直到总大小低于限制
	for i := 0; i < len(items) && totalSize > maxSize; i++ {
		item := items[i]

		// 删除缓存文件
		if err := os.Remove(item.path); err != nil {
			log.Printf("删除缓存文件失败 %s: %v", item.path, err)
			continue
		}

		// 删除元数据文件
		if err := os.Remove(item.metaPath); err != nil {
			log.Printf("删除元数据文件失败 %s: %v", item.metaPath, err)
		}

		totalSize -= item.size
		if cm.config.CacheLog {
			log.Printf("LRU清理: 删除文件 %s (已释放: %d bytes)", item.path, item.size)
		}
	}

	if cm.config.CacheLog {
		log.Printf("LRU缓存清理完成，当前大小: %d MB", totalSize/1024/1024)
	}

	return nil
}

// 更频繁地运行清理
func (cm *CacheMiddleware) cleanupRoutine() {
	ticker := time.NewTicker(15 * time.Minute) // 每15分钟检查一次
	for range ticker.C {
		cm.cleanup()
	}
}

// 缓存项信息
type cacheItem struct {
	path     string    // 缓存文件路径
	size     int64     // 文件大小
	lastUsed time.Time // 最后访问时间
	metaPath string    // 元数据文件路径
}
