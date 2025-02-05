package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path"
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
	lastSync    map[string]time.Time     // 新增：记录每个仓库最后同步时间
	syncMutex   sync.Mutex               // 新增：保护 lastSync map 的互斥锁
	syncStatus  map[string]*SyncStatus   // 新增：同步状态追踪
	statusMutex sync.RWMutex             // 新增：状态锁
	buckets     map[string]*minio.Client // 新增多桶客户端映射
}

// 新增：同步状态结构
type SyncStatus struct {
	LastSync     time.Time `json:"lastSync"`        // 最后同步时间
	NextSync     time.Time `json:"nextSync"`        // 下次同步时间
	Progress     float64   `json:"progress"`        // 同步进度(0-100)
	TotalFiles   int       `json:"totalFiles"`      // 总文件数
	CurrentFiles int       `json:"currentFiles"`    // 已处理文件数
	Status       string    `json:"status"`          // 同步状态(idle/syncing/error)
	Error        string    `json:"error,omitempty"` // 错误信息
}

func NewMinioService(config *config.Config) (*MinioService, error) {
	client, err := minio.New(config.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Minio.AccessKey, config.Minio.SecretKey, ""),
		Secure: config.Minio.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	// 初始化多桶客户端
	buckets := make(map[string]*minio.Client)
	for _, bucketConfig := range config.Buckets {
		client, err := minio.New(bucketConfig.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(bucketConfig.AccessKey, bucketConfig.SecretKey, ""),
			Secure: bucketConfig.UseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("初始化桶 %s 失败: %v", bucketConfig.Name, err)
		}
		buckets[bucketConfig.Name] = client
	}

	return &MinioService{
		client:     client,
		config:     config,
		lastSync:   make(map[string]time.Time),
		syncStatus: make(map[string]*SyncStatus),
		buckets:    buckets,
	}, nil
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

// 修改 SHA1 校验，统一使用元数据键 "Sha1"
func (s *MinioService) needsUpdate(objectName, localPath string) (bool, error) {
	// 处理路径分隔符
	objectName = strings.ReplaceAll(objectName, string(os.PathSeparator), "/")
	localPath = filepath.Clean(localPath)

	// 检查本地文件权限
	if err := ensureFilePermissions(localPath); err != nil {
		log.Printf("权限检查失败 %s: %v", localPath, err)
		return false, err
	}

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

	// 调试输出：打印远程所有元数据和本地SHA1
	// log.Printf("检查文件: %s, localSHA1: %s, remote元数据: %+v", objectName, localSHA1, stat.UserMetadata)

	// 使用 "Sha1" 键进行检查
	remoteSHA1, ok := stat.UserMetadata["Sha1"]
	if ok && remoteSHA1 == localSHA1 {
		log.Printf("文件未改变, 跳过上传: %s", objectName)
		return false, nil
	}
	return true, nil
}

// 存储远程文件列表
type MinioObjects struct {
	Objects map[string]struct{}
}

// 获取Minio中指定路径下的所有文件
func (s *MinioService) ListObjects(prefix string) ([]MinioObject, error) {
	ctx := context.Background()
	var objects []MinioObject

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	// 遍历 minio 对象列表，并构造返回列表
	for object := range s.client.ListObjects(ctx, s.config.Minio.Bucket, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, MinioObject{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
		})
	}
	return objects, nil
}

// 定义公开的 MinioObject 结构体，如未定义则添加
type MinioObject struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// 删除Minio中的文件
func (s *MinioService) removeObject(objectPath string) error {
	return s.client.RemoveObject(context.Background(), s.config.Minio.Bucket, objectPath, minio.RemoveObjectOptions{})
}

// 检查是否需要同步
func (s *MinioService) shouldSync(minioPath string, checkInterval time.Duration) bool {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	lastSync, exists := s.lastSync[minioPath]
	if !exists {
		s.lastSync[minioPath] = time.Now()
		return true
	}

	if time.Since(lastSync) >= checkInterval {
		s.lastSync[minioPath] = time.Now()
		return true
	}

	return false
}

// 初始化最后同步时间
func (s *MinioService) InitLastSync(minioPath string) {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()
	s.lastSync[minioPath] = time.Now()
	log.Printf("初始化同步时间: %s", minioPath)
}

// 新增：更新同步状态
func (s *MinioService) updateSyncStatus(minioPath string, update func(*SyncStatus)) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	status, exists := s.syncStatus[minioPath]
	if !exists {
		status = &SyncStatus{Status: "idle"}
		s.syncStatus[minioPath] = status
	}
	update(status)
}

// 新增：获取同步状态
func (s *MinioService) GetSyncStatus(minioPath string) *SyncStatus {
	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()

	if status, exists := s.syncStatus[minioPath]; exists {
		return status
	}
	return &SyncStatus{Status: "unknown"}
}

func (s *MinioService) UploadDirectory(localPath, minioPath string, checkInterval time.Duration) error {
	// 更新同步开始状态
	s.updateSyncStatus(minioPath, func(status *SyncStatus) {
		status.Status = "syncing"
		status.Progress = 0
		status.CurrentFiles = 0
		status.Error = ""
		status.LastSync = time.Now()
		if checkInterval > 0 {
			status.NextSync = time.Now().Add(checkInterval)
		}
	})

	defer func() {
		if r := recover(); r != nil {
			s.updateSyncStatus(minioPath, func(status *SyncStatus) {
				status.Status = "error"
				status.Error = fmt.Sprintf("panic: %v", r)
			})
		}
	}()

	// 修改检查逻辑：当 checkInterval 为 0 时强制同步
	if checkInterval > 0 && !s.shouldSync(minioPath, checkInterval) {
		log.Printf("跳过同步，未到检查时间: %s", minioPath)
		return nil
	}

	log.Printf("开始同步目录: %s, 间隔时间: %v", minioPath, checkInterval)
	// 构建完整的本地路径
	fullPath := filepath.Join(s.config.Git.CachePath, localPath)
	fullPath = filepath.Clean(fullPath) // 清理路径

	// 确保目录存在并设置正确权限
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("创建目录失败 %s: %v", fullPath, err)
	}

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
			// 处理权限错误
			if os.IsPermission(err) {
				log.Printf("权限不足 %s: %v", path, err)
				return nil // 跳过此文件但继续处理
			}
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
		// 规范化对象名称
		objName := filepath.Join(minioPath, relPath)
		// 统一使用 / 作为路径分隔符
		objName = strings.ReplaceAll(objName, string(os.PathSeparator), "/")
		jobs = append(jobs, fileJob{fullLocalPath: path, objectName: objName, info: info})
		return nil
	})
	if err != nil {
		return err
	}

	// 并发上传任务，使用工作池处理
	processedFiles := make(map[string]struct{})
	var pfMutex sync.Mutex

	// 使用配置的线程数，如果配置值小于1则使用默认值16
	maxWorkers := s.config.Minio.MaxWorkers
	if maxWorkers < 1 {
		maxWorkers = 16
	}

	jobChan := make(chan fileJob)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for job := range jobChan {
			if s.config.Logs.ProcessLog {
				log.Printf("Processing: %s -> %s", job.fullLocalPath, job.objectName)
			}
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

			// 修改上传文件时设置的元数据键为 "Sha1"
			userMetadata := map[string]string{
				"Sha1": sha1Hash,
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

	wg.Add(maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		go worker()
	}
	// 发送任务
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)
	wg.Wait()

	// 删除Minio中存在但本地不存在的文件
	existingObjects, err := s.ListObjects(minioPath)
	if err != nil {
		return fmt.Errorf("获取Minio文件列表失败: %v", err)
	}
	for _, obj := range existingObjects {
		pfMutex.Lock()
		_, exists := processedFiles[obj.Key]
		pfMutex.Unlock()
		if !exists {
			log.Printf("删除已移除的文件: %s", obj.Key)
			if err := s.removeObject(obj.Key); err != nil {
				log.Printf("删除文件失败 %s: %v", obj.Key, err)
			}
		}
	}

	// 更新进度
	s.updateSyncStatus(minioPath, func(status *SyncStatus) {
		status.Status = "idle"
		status.Progress = 100
		status.LastSync = time.Now()
	})

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
		if s.config.Logs.PresignLog {
			log.Printf("PreSign failed: %s: %v", objectPath, err)
		}
		return ""
	}
	if s.config.Logs.PresignLog {
		log.Printf("PreSign success: %s -> %s", objectPath, presignedURL.String())
	}
	return presignedURL.String()
}

// 修改权限检查和设置函数
func ensureFilePermissions(path string) error {
	// 获取文件信息
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// 设置目录权限
	if info.IsDir() {
		if err := os.Chmod(path, 0755); err != nil {
			return fmt.Errorf("设置目录权限失败 %s: %v", path, err)
		}
	} else {
		// 设置文件权限
		if err := os.Chmod(path, 0644); err != nil {
			return fmt.Errorf("设置文件权限失败 %s: %v", path, err)
		}
	}

	return nil
}

// 新增从指定桶获取对象的方法
func (s *MinioService) GetObjectFromBucket(bucketName, objectPath string) (*minio.Object, error) {
	for _, bucket := range s.config.Buckets {
		if bucket.Name == bucketName {
			client := s.buckets[bucketName]
			return client.GetObject(
				context.Background(),
				bucket.BucketName,
				path.Join(bucket.BasePath, objectPath),
				minio.GetObjectOptions{},
			)
		}
	}
	return nil, fmt.Errorf("bucket not found: %s", bucketName)
}
