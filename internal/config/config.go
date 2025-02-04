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
}

type Server struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type Minio struct {
	Endpoint     string `yaml:"endpoint"`
	AccessKey    string `yaml:"accessKey"`
	SecretKey    string `yaml:"secretKey"`
	UseSSL       bool   `yaml:"useSSL"`
	Bucket       string `yaml:"bucket"`
	UsePublicURL bool   `yaml:"usePublicURL"`
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
			Port: 8080,
			Host: "0.0.0.0",
		},
		Minio: Minio{
			Endpoint:     "play.min.io",
			AccessKey:    "your-access-key",
			SecretKey:    "your-secret-key",
			UseSSL:       true,
			Bucket:       "documents",
			UsePublicURL: true,
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
