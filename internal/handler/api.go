package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"pysio.online/Files-API/internal/config"
	"pysio.online/Files-API/internal/service"
)

type APIHandler struct {
	minioService *service.MinioService
	config       *config.Config
}

func NewAPIHandler(minioService *service.MinioService, config *config.Config) *APIHandler {
	return &APIHandler{
		minioService: minioService,
		config:       config,
	}
}

// API 响应格式
type APIResponse struct {
	Code    int         `json:"code"`                 // 状态码
	Message string      `json:"message"`              // 提示信息
	Data    interface{} `json:"data,omitempty"`       // 数据
	Page    *Pagination `json:"pagination,omitempty"` // 分页信息
}

// 分页信息
type Pagination struct {
	Current  int `json:"current"`  // 当前页
	PageSize int `json:"pageSize"` // 每页大小
	Total    int `json:"total"`    // 总条数
}

// 文件信息
type FileInfo struct {
	Name         string    `json:"name"`          // 文件名
	Path         string    `json:"path"`          // 完整路径
	Size         int64     `json:"size"`          // 文件大小
	LastModified time.Time `json:"lastModified"`  // 最后修改时间
	IsDirectory  bool      `json:"isDirectory"`   // 是否是目录
	URL          string    `json:"url,omitempty"` // 访问URL（仅文件有）
}

// 新增：同步状态响应结构
type SyncStatusResponse struct {
	Code    int                            `json:"code"`
	Message string                         `json:"message"`
	Data    map[string]*service.SyncStatus `json:"data"`
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 设置 JSON 响应头
	w.Header().Set("Content-Type", "application/json")

	// 解析请求路径
	prefix := strings.TrimPrefix(r.URL.Path, "/api/files/")

	// 访问日志
	if h.config.Logs.AccessLog {
		log.Printf("API Access: %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
	}

	// 添加新的路由
	if strings.HasSuffix(r.URL.Path, "/sync/status") {
		h.handleSyncStatus(w, r)
		return
	}

	// 处理 PATCH 请求
	if r.Method == http.MethodPatch {
		h.handlePatchRequest(w, r)
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取文件列表
	objects, err := h.minioService.ListObjects(prefix)
	if err != nil {
		h.responseError(w, http.StatusInternalServerError, "获取文件列表失败")
		return
	}

	// 构建文件列表
	var files []FileInfo
	seenDirs := make(map[string]bool)

	for _, obj := range objects {
		// 跳过当前目录
		if obj.Key == prefix {
			continue
		}

		// 相对于当前目录的路径
		relPath := strings.TrimPrefix(obj.Key, prefix)
		parts := strings.Split(relPath, "/")

		if len(parts) > 1 {
			// 这是子目录中的文件，添加目录条目
			dirName := parts[0]
			dirPath := path.Join(prefix, dirName) + "/"
			if !seenDirs[dirPath] {
				files = append(files, FileInfo{
					Name:        dirName,
					Path:        dirPath,
					IsDirectory: true,
				})
				seenDirs[dirPath] = true
			}
		} else {
			// 这是文件
			fileURL := ""
			if h.config.Minio.UsePublicURL {
				fileURL = h.minioService.GetPublicURL(obj.Key)
			}

			files = append(files, FileInfo{
				Name:         path.Base(obj.Key),
				Path:         obj.Key,
				Size:         obj.Size,
				LastModified: obj.LastModified,
				IsDirectory:  false,
				URL:          fileURL,
			})
		}
	}

	// 计算分页
	total := len(files)
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	if start >= total {
		files = []FileInfo{}
	} else {
		files = files[start:end]
	}

	// 返回响应
	h.responseSuccess(w, files, &Pagination{
		Current:  page,
		PageSize: pageSize,
		Total:    total,
	})
}

// 新增：处理同步状态请求
func (h *APIHandler) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.responseError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 收集所有仓库的同步状态
	statuses := make(map[string]*service.SyncStatus)
	for _, repo := range h.config.Git.Repositories {
		status := h.minioService.GetSyncStatus(repo.MinioPath)
		statuses[repo.MinioPath] = status
	}

	response := SyncStatusResponse{
		Code:    200,
		Message: "success",
		Data:    statuses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 新增 PATCH 请求结构
type PatchRequest struct {
	Bucket string `json:"bucket"`
	Path   string `json:"path"`
}

// 处理 PATCH 请求
func (h *APIHandler) handlePatchRequest(w http.ResponseWriter, r *http.Request) {
	var req PatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.responseError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	// 检查桶是否存在且有权限
	var bucketConfig *config.BucketConfig
	for _, b := range h.config.Buckets {
		if b.Name == req.Bucket {
			bucketConfig = &b
			break
		}
	}

	if bucketConfig == nil {
		h.responseError(w, http.StatusNotFound, "存储桶不存在")
		return
	}

	if bucketConfig.ReadOnly {
		h.responseError(w, http.StatusForbidden, "存储桶为只读")
		return
	}

	// 获取对象
	obj, err := h.minioService.GetObjectFromBucket(req.Bucket, req.Path)
	if err != nil {
		h.responseError(w, http.StatusNotFound, "文件不存在")
		return
	}
	defer obj.Close()

	// 获取文件信息并返回
	info, err := obj.Stat()
	if err != nil {
		h.responseError(w, http.StatusInternalServerError, "获取文件信息失败")
		return
	}

	fileInfo := FileInfo{
		Name:         path.Base(req.Path),
		Path:         req.Path,
		Size:         info.Size,
		LastModified: info.LastModified,
		IsDirectory:  false,
	}

	h.responseSuccess(w, fileInfo, nil)
}

func (h *APIHandler) responseSuccess(w http.ResponseWriter, data interface{}, pagination *Pagination) {
	resp := APIResponse{
		Code:    200,
		Message: "success",
		Data:    data,
		Page:    pagination,
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *APIHandler) responseError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	resp := APIResponse{
		Code:    code,
		Message: message,
	}
	json.NewEncoder(w).Encode(resp)
}
