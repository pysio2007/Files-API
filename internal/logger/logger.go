package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"pysio.online/Files-API/internal/config"
)

type Logger struct {
	config     *config.LogConfig
	currentLog *os.File
}

func New(cfg *config.LogConfig) (*Logger, error) {
	logger := &Logger{config: cfg}

	if !cfg.SaveToFile {
		// 仅输出到控制台
		log.SetOutput(os.Stdout)
		return logger, nil
	}

	if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	if err := logger.rotateLog(); err != nil {
		return nil, err
	}

	// 启动定时器，每天零点切换日志文件
	go logger.scheduledRotation()

	return logger, nil
}

func (l *Logger) rotateLog() error {
	if l.currentLog != nil {
		l.currentLog.Close()
	}

	// 生成新日志文件名
	timestamp := time.Now().Format("2006-01-02")
	logPath := filepath.Join(l.config.Directory, fmt.Sprintf("Files-API-%s.log", timestamp))

	// 打开新日志文件
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}

	l.currentLog = logFile

	// 设置日志输出到文件和控制台
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// 检查并清理旧日志
	l.cleanOldLogs()

	return nil
}

func (l *Logger) cleanOldLogs() {
	// 获取所有日志文件
	files, err := filepath.Glob(filepath.Join(l.config.Directory, "Files-API-*.log"))
	if err != nil {
		log.Printf("获取日志文件列表失败: %v", err)
		return
	}

	// 按修改时间排序
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		return fi.ModTime().Before(fj.ModTime())
	})

	// 计算总大小
	var totalSize int64
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		totalSize += info.Size()
	}

	// 如果超过限制，从最旧的文件开始删除
	maxSize := int64(l.config.MaxSize) * 1024 * 1024 // 转换为字节
	for i := 0; totalSize > maxSize && i < len(files); i++ {
		info, err := os.Stat(files[i])
		if err != nil {
			continue
		}
		if err := os.Remove(files[i]); err != nil {
			log.Printf("删除旧日志文件失败 %s: %v", files[i], err)
			continue
		}
		totalSize -= info.Size()
		log.Printf("已删除旧日志文件: %s", files[i])
	}
}

func (l *Logger) scheduledRotation() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		duration := next.Sub(now)

		time.Sleep(duration)
		if err := l.rotateLog(); err != nil {
			log.Printf("轮换日志文件失败: %v", err)
		}
	}
}

func (l *Logger) Close() {
	if l.currentLog != nil {
		l.currentLog.Close()
	}
}
