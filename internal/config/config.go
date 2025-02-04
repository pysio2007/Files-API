package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       Server        `yaml:"server"`
	Minio        Minio         `yaml:"minio"`
	Git          Git           `yaml:"git"`
	ExposedPaths []ExposedPath `yaml:"exposedPaths"`
	Logs         LogConfig     `yaml:"logs"` // 新增日志配置
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

type Server struct {
	Port      int    `yaml:"port"`
	Host      string `yaml:"host"`
	EnableAPI bool   `yaml:"enableAPI"` // 新增：是否启用 API
	APIOnly   bool   `yaml:"apiOnly"`   // 新增：仅启用 API
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
					CheckInterval: "1h", // 新增：默认检查间隔
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
