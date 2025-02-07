package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       Server         `yaml:"server"`
	Minio        Minio          `yaml:"minio"`
	Git          Git            `yaml:"git"`
	ExposedPaths []ExposedPath  `yaml:"exposedPaths"`
	Logs         LogConfig      `yaml:"logs"`
	Cache        CacheConfig    `yaml:"cache"`        // 新增缓存配置
	Buckets      []BucketConfig `yaml:"buckets"`      // 新增多桶配置
	ExternalURLs []ExternalURL  `yaml:"externalURLs"` // 新增外部URL配置
}

// 新增：日志配置结构
type LogConfig struct {
	AccessLog   bool   `yaml:"accessLog"`   // 访问日志
	ProcessLog  bool   `yaml:"processLog"`  // 处理详情
	RedirectLog bool   `yaml:"redirectLog"` // 跳转详情
	PresignLog  bool   `yaml:"presignLog"`  // 预签名URL详情
	SaveToFile  bool   `yaml:"saveToFile"`  // 是否保存到文件
	MaxSize     int    `yaml:"maxSize"`     // 日志目录最大大小(MB)
	Directory   string `yaml:"directory"`   // 日志保存目录
}

// 新增：缓存配置结构
type CacheConfig struct {
	Enabled         bool     `yaml:"enabled"`         // 是否启用缓存
	Directory       string   `yaml:"directory"`       // 缓存目录
	MaxSize         int      `yaml:"maxSize"`         // 缓存目录最大大小(MB)
	TTL             string   `yaml:"ttl"`             // 缓存有效期
	CacheControl    string   `yaml:"cacheControl"`    // CDN缓存时间
	CacheLog        bool     `yaml:"cacheLog"`        // 是否记录缓存操作日志
	HitLog          bool     `yaml:"hitLog"`          // 是否记录缓存命中日志
	EnableAPICache  bool     `yaml:"enableAPICache"`  // 是否启用API缓存控制
	APICacheControl string   `yaml:"apiCacheControl"` // API缓存控制时间
	APIExcludePaths []string `yaml:"apiExcludePaths"` // 不缓存的API路径
}

// 新增存储桶配置结构
type BucketConfig struct {
	Name       string `yaml:"name"`
	Endpoint   string `yaml:"endpoint"`
	AccessKey  string `yaml:"accessKey"`
	SecretKey  string `yaml:"secretKey"`
	UseSSL     bool   `yaml:"useSSL"`
	BucketName string `yaml:"bucketName"`
	BasePath   string `yaml:"basePath"` // 基础路径
	ReadOnly   bool   `yaml:"readOnly"` // 是否只读
}

// 添加新的配置结构
type ExternalURL struct {
	Path          string   `yaml:"path"`          // 访问路径
	MainURL       string   `yaml:"mainURL"`       // 主URL
	BackupURLs    []string `yaml:"backupURLs"`    // 备用URL列表
	MinioPath     string   `yaml:"minioPath"`     // Minio存储路径
	CacheControl  string   `yaml:"cacheControl"`  // 缓存控制,如 "no-cache" 或 "max-age=3600"
	CheckInterval string   `yaml:"checkInterval"` // 检查间隔
}

type Server struct {
	Port         int      `yaml:"port"`
	Host         string   `yaml:"host"`
	EnableAPI    bool     `yaml:"enableAPI"`    // 新增：是否启用 API
	APIOnly      bool     `yaml:"apiOnly"`      // 新增：仅启用 API
	LegacyAPI    bool     `yaml:"legacyAPI"`    // 是否支持旧版API格式
	AllowOrigins []string `yaml:"allowOrigins"` // 新增: CORS 允许的域名列表
}

type Minio struct {
	Endpoint     string `yaml:"endpoint"`
	AccessKey    string `yaml:"accessKey"`
	SecretKey    string `yaml:"secretKey"`
	UseSSL       bool   `yaml:"useSSL"`
	Bucket       string `yaml:"bucket"`
	UsePublicURL bool   `yaml:"usePublicURL"`
	MaxWorkers   int    `yaml:"maxWorkers"` // 新增：最大并发上传线程数
}

type Git struct {
	CachePath    string       `yaml:"cachePath"`
	Repositories []Repository `yaml:"repositories"`
}

type Repository struct {
	URL           string `yaml:"url"`
	Branch        string `yaml:"branch"`
	LocalPath     string `yaml:"localPath"`
	MinioPath     string `yaml:"minioPath"`
	DisabledSync  bool   `yaml:"disabledSync"`  // 新增：是否禁用同步
	CheckInterval string `yaml:"checkInterval"` // 新增：仓库检查间隔
}

type ExposedPath struct {
	URLPath   string `yaml:"urlPath"`
	MinioPath string `yaml:"minioPath"`
}

func LoadConfig(path string) (*Config, error) {
	// 首次运行时创建默认配置
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefaultConfig(path); err != nil {
			return nil, fmt.Errorf("无法创建默认配置文件: %v", err)
		}
		fmt.Printf("已创建默认配置文件: %s\n", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

func createDefaultConfig(path string) error {
	defaultConfig := Config{
		Server: Server{
			Port:      8080,
			Host:      "0.0.0.0",
			EnableAPI: true,  // 默认启用 API
			APIOnly:   false, // 默认不仅启用 API
			LegacyAPI: false, // 默认不启用旧版API支持
			AllowOrigins: []string{ // 新增: 默认允许的域名
				"http://localhost:8080",
			},
		},
		Minio: Minio{
			Endpoint:     "play.min.io",
			AccessKey:    "your-access-key",
			SecretKey:    "your-secret-key",
			UseSSL:       true,
			Bucket:       "documents",
			UsePublicURL: true,
			MaxWorkers:   16, // 默认16个线程
		},
		Git: Git{
			CachePath: ".cache/repos",
			Repositories: []Repository{
				{
					URL:           "https://github.com/user/repo1",
					Branch:        "main",
					LocalPath:     "docs/repo1",
					MinioPath:     "repo1",
					DisabledSync:  false, // 默认启用同步
					CheckInterval: "1h",  // 新增：默认检查间隔
				},
			},
		},
		ExposedPaths: []ExposedPath{
			{
				URLPath:   "/public-docs",
				MinioPath: "public",
			},
		},
		Logs: LogConfig{
			AccessLog:   true,   // 默认开启访问日志
			ProcessLog:  false,  // 默认关闭处理详情
			RedirectLog: false,  // 默认关闭跳转详情
			PresignLog:  false,  // 默认关闭预签名URL详情
			SaveToFile:  true,   // 默认保存到文件
			MaxSize:     100,    // 默认100MB
			Directory:   "logs", // 默认logs目录
		},
		Cache: CacheConfig{
			Enabled:         true,
			Directory:       ".cache/files",
			MaxSize:         1000,  // 默认1GB
			TTL:             "7d",  // 默认7天
			CacheControl:    "30d", // CDN缓存30天
			CacheLog:        false, // 默认不记录缓存操作
			HitLog:          false, // 默认不记录命中日志
			EnableAPICache:  true,  // 默认启用API缓存控制
			APICacheControl: "5m",  // API默认缓存5分钟
			APIExcludePaths: []string{
				"/api/files/sync/status", // 默认不缓存同步状态接口
			},
		},
		Buckets: []BucketConfig{
			{
				Name:       "blog-assets",
				Endpoint:   "play.min.io",
				AccessKey:  "bucket1-access-key",
				SecretKey:  "bucket1-secret-key",
				UseSSL:     true,
				BucketName: "blog-assets",
				BasePath:   "assets",
				ReadOnly:   true,
			},
		},
		ExternalURLs: []ExternalURL{
			{
				Path:          "/external/example.jpg",
				MainURL:       "https://example.com/image.jpg",
				BackupURLs:    []string{"https://backup1.com/image.jpg", "https://backup2.com/image.jpg"},
				MinioPath:     "external/example.jpg",
				CacheControl:  "max-age=3600",
				CheckInterval: "1h",
			},
		},
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return fmt.Errorf("生成默认配置失败: %v", err)
	}

	// 创建配置文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}
