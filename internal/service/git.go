package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
		log.Printf("浅克隆仓库到: %s", fullPath)
		// 浅克隆，仅拉取最新提交
		cmd := exec.Command("git", "clone", "--depth", "1", "-b", repo.Branch, repo.URL, fullPath)
		return cmd.Run()
	}

	log.Printf("更新仓库: %s", fullPath)
	// 使用 fetch --depth 1 拉取最新提交
	cmd := exec.Command("git", "-C", fullPath, "fetch", "--depth", "1", "origin", repo.Branch)
	if err := cmd.Run(); err != nil {
		return err
	}
	// 使用 reset --hard 同步至最新版本
	cmd = exec.Command("git", "-C", fullPath, "reset", "--hard", "origin/"+repo.Branch)
	return cmd.Run()
}

// 新增辅助函数：解析自定义时长
func parseDurationCustom(val string) (time.Duration, error) {
	if val == "" {
		return 10 * time.Minute, nil
	}
	// 支持以 d（天）或 y（年）结尾
	last := val[len(val)-1]
	if last == 'd' {
		num, err := strconv.Atoi(strings.TrimSuffix(val, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(num) * 24 * time.Hour, nil
	}
	if last == 'y' {
		num, err := strconv.Atoi(strings.TrimSuffix(val, "y"))
		if err != nil {
			return 0, err
		}
		return time.Duration(num) * 365 * 24 * time.Hour, nil
	}
	// 其他情况交给 time.ParseDuration 处理
	return time.ParseDuration(val)
}

// GetCheckInterval 返回指定仓库的检查间隔，无效或未设置则默认 10 分钟
func (s *GitService) GetCheckInterval(repo *config.Repository) time.Duration {
	if repo.CheckInterval == "" {
		return 10 * time.Minute
	}
	d, err := parseDurationCustom(repo.CheckInterval)
	if err != nil {
		log.Printf("解析仓库 %s 的检查间隔失败: %v, 使用默认值10分钟", repo.URL, err)
		return 10 * time.Minute
	}
	return d
}
