package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"pysio.online/Files-API/internal/config"
)

type GitService struct {
	config *config.Config
}

func NewGitService(config *config.Config) *GitService {
	return &GitService{config: config}
}

func (s *GitService) SyncRepository(repo *config.Repository) error {
	// 构建完整的缓存路径
	fullPath := filepath.Join(s.config.Git.CachePath, repo.LocalPath)

	// 确保缓存目录存在
	if err := os.MkdirAll(s.config.Git.CachePath, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %v", err)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("克隆仓库到: %s", fullPath)
		// 首次克隆到缓存目录
		cmd := exec.Command("git", "clone", "-b", repo.Branch, repo.URL, fullPath)
		return cmd.Run()
	}

	log.Printf("更新仓库: %s", fullPath)
	// 更新已存在的仓库
	cmd := exec.Command("git", "-C", fullPath, "pull", "origin", repo.Branch)
	return cmd.Run()
}
