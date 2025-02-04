package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"pysio.online/Files-API/internal/config"
)

// 辅助函数：根据文件扩展名获取 Content-Type
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

type MinioService struct {
	client      *minio.Client
	config      *config.Config
	uploadMutex sync.Mutex
}

func NewMinioService(config *config.Config) (*MinioService, error) {
	client, err := minio.New(config.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Minio.AccessKey, config.Minio.SecretKey, ""),
		Secure: config.Minio.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MinioService{client: client, config: config}, nil
}

func (s *MinioService) CheckConnection() error {
	// 检查bucket是否存在
	exists, err := s.client.BucketExists(context.Background(), s.config.Minio.Bucket)
	if err != nil {
		return fmt.Errorf("无法连接Minio服务器: %v", err)
	}
	if !exists {
		// 如果bucket不存在，尝试创建
		err = s.client.MakeBucket(context.Background(), s.config.Minio.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("创建bucket失败: %v", err)
		}
	}
	return nil
}

// 计算文件SHA1
func calculateSHA1(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// 检查文件是否需要更新
func (s *MinioService) needsUpdate(objectName, localPath string) (bool, error) {
	// 获取Minio对象的元数据
	stat, err := s.client.StatObject(context.Background(), s.config.Minio.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		if err.Error() == "The specified key does not exist." {
			return true, nil
		}
		return false, err
	}

	// 获取本地文件SHA1
	localSHA1, err := calculateSHA1(localPath)
	if err != nil {
		return false, err
	}

	// 比较SHA1
	minioSHA1, ok := stat.UserMetadata["X-Amz-Meta-Sha1"]
	return !ok || minioSHA1 != localSHA1, nil
}

func (s *MinioService) UploadDirectory(localPath, minioPath string) error {
	s.uploadMutex.Lock()
	defer s.uploadMutex.Unlock()

	// 确保本地路径存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("本地路径不存在: %s", localPath)
	}

	// 遍历目录
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和.git文件夹
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// 构造 Minio 对象路径
		objectName := filepath.Join(minioPath, relPath)
		objectName = strings.ReplaceAll(objectName, "\\", "/") // Windows 路径修正

		// 检查是否需要更新
		needsUpdate, err := s.needsUpdate(objectName, path)
		if err != nil {
			log.Printf("检查文件状态失败 %s: %v", objectName, err)
			return nil
		}

		if !needsUpdate {
			log.Printf("跳过未变更文件: %s", objectName)
			return nil
		}

		// 计算SHA1
		sha1Hash, err := calculateSHA1(path)
		if err != nil {
			return err
		}

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// 设置元数据
		userMetadata := map[string]string{
			"X-Amz-Meta-Sha1": sha1Hash,
		}

		// 上传文件
		_, err = s.client.PutObject(
			context.Background(),
			s.config.Minio.Bucket,
			objectName,
			file,
			info.Size(),
			minio.PutObjectOptions{
				ContentType:  getContentType(path),
				UserMetadata: userMetadata,
			},
		)
		if err != nil {
			return fmt.Errorf("上传失败 %s: %v", objectName, err)
		}

		log.Printf("成功上传文件: %s", objectName)
		return nil
	})
}

func (s *MinioService) GetObject(objectPath string) (*minio.Object, error) {
	return s.client.GetObject(
		context.Background(),
		s.config.Minio.Bucket,
		objectPath,
		minio.GetObjectOptions{},
	)
}

func (s *MinioService) GetPublicURL(objectPath string) string {
	// 生成预签名URL，有效期1小时
	presignedURL, err := s.client.PresignedGetObject(
		context.Background(),
		s.config.Minio.Bucket,
		objectPath,
		time.Hour,
		nil,
	)
	if err != nil {
		return ""
	}
	return presignedURL.String()
}
