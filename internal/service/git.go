package service

import (
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
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// 首次克隆到缓存目录
		cmd := exec.Command("git", "clone", "-b", repo.Branch, repo.URL, fullPath)
		return cmd.Run()
	}

	// 更新已存在的仓库
	cmd := exec.Command("git", "-C", fullPath, "pull", "origin", repo.Branch)
	return cmd.Run()
}
