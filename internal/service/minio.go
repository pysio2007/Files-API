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
	client *minio.Client
	config *config.Config
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

// 存储远程文件列表
type MinioObjects struct {
	Objects map[string]struct{}
}

// 获取Minio中指定路径下的所有文件
func (s *MinioService) listObjects(minioPath string) (*MinioObjects, error) {
	ctx := context.Background()
	objects := &MinioObjects{
		Objects: make(map[string]struct{}),
	}

	opts := minio.ListObjectsOptions{
		Prefix:    minioPath,
		Recursive: true,
	}

	// 列出所有对象
	for object := range s.client.ListObjects(ctx, s.config.Minio.Bucket, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		objects.Objects[object.Key] = struct{}{}
	}

	return objects, nil
}

// 删除Minio中的文件
func (s *MinioService) removeObject(objectPath string) error {
	return s.client.RemoveObject(context.Background(), s.config.Minio.Bucket, objectPath, minio.RemoveObjectOptions{})
}

func (s *MinioService) UploadDirectory(localPath, minioPath string) error {
	// 构建完整的本地路径
	fullPath := filepath.Join(s.config.Git.CachePath, localPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("本地路径不存在: %s", fullPath)
	}

	// 先收集所有待处理的文件
	type fileJob struct {
		fullLocalPath string
		objectName    string
		info          os.FileInfo
	}
	var jobs []fileJob
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
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
		relPath, err := filepath.Rel(fullPath, path)
		if err != nil {
			return err
		}
		objName := filepath.Join(minioPath, relPath)
		objName = strings.ReplaceAll(objName, "\\", "/") // Windows 路径修正
		jobs = append(jobs, fileJob{fullLocalPath: path, objectName: objName, info: info})
		return nil
	})
	if err != nil {
		return err
	}

	// 并发上传任务，使用工作池处理
	processedFiles := make(map[string]struct{})
	var pfMutex sync.Mutex
	const maxConcurrentUploads = 5
	jobChan := make(chan fileJob)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for job := range jobChan {
			// 标记已处理文件
			pfMutex.Lock()
			processedFiles[job.objectName] = struct{}{}
			pfMutex.Unlock()

			// 检查是否需要更新
			needsUpd, err := s.needsUpdate(job.objectName, job.fullLocalPath)
			if err != nil {
				log.Printf("检查文件状态失败 %s: %v", job.objectName, err)
				continue
			}
			if !needsUpd {
				log.Printf("跳过未变更文件: %s", job.objectName)
				continue
			}

			// 计算 SHA1
			sha1Hash, err := calculateSHA1(job.fullLocalPath)
			if err != nil {
				log.Printf("计算文件 %s SHA1失败: %v", job.objectName, err)
				continue
			}

			// 打开文件
			file, err := os.Open(job.fullLocalPath)
			if err != nil {
				log.Printf("打开文件失败 %s: %v", job.objectName, err)
				continue
			}

			userMetadata := map[string]string{
				"X-Amz-Meta-Sha1": sha1Hash,
			}
			maxRetries := 3
			var uploadErr error
			for i := 0; i < maxRetries; i++ {
				// 重置文件指针以便重传
				if _, err := file.Seek(0, 0); err != nil {
					log.Printf("重置文件指针失败 %s: %v", job.objectName, err)
					break
				}
				_, uploadErr = s.client.PutObject(
					context.Background(),
					s.config.Minio.Bucket,
					job.objectName,
					file,
					job.info.Size(),
					minio.PutObjectOptions{
						ContentType:  getContentType(job.fullLocalPath),
						UserMetadata: userMetadata,
					},
				)
				if uploadErr == nil {
					log.Printf("成功上传文件: %s", job.objectName)
					break
				}
				log.Printf("第%d次上传失败 %s: %v", i+1, job.objectName, uploadErr)
				time.Sleep(2 * time.Second)
			}
			file.Close()
		}
	}

	wg.Add(maxConcurrentUploads)
	for i := 0; i < maxConcurrentUploads; i++ {
		go worker()
	}
	// 发送任务
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)
	wg.Wait()

	// 删除Minio中存在但本地不存在的文件
	existingObjects, err := s.listObjects(minioPath)
	if err != nil {
		return fmt.Errorf("获取Minio文件列表失败: %v", err)
	}
	for objectPath := range existingObjects.Objects {
		pfMutex.Lock()
		_, exists := processedFiles[objectPath]
		pfMutex.Unlock()
		if !exists {
			log.Printf("删除已移除的文件: %s", objectPath)
			if err := s.removeObject(objectPath); err != nil {
				log.Printf("删除文件失败 %s: %v", objectPath, err)
			}
		}
	}

	return nil
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
