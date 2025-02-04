package middleware

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CliFlags struct {
	Help       bool
	Skip       bool
	ZipLogs    bool
	UnzipLogs  bool
	Sync       bool   // 新增：执行单次同步
	ClearLogs  bool   // 新增：清除所有日志
	ClearCache bool   // 新增：清除缓存目录
	ClearAll   bool   // 新增：清除所有
	RSync      string // 新增：指定要同步的仓库路径
}

func ParseFlags() *CliFlags {
	flags := &CliFlags{}

	flag.BoolVar(&flags.Help, "h", false, "显示帮助信息")
	flag.BoolVar(&flags.Help, "help", false, "显示帮助信息")
	flag.BoolVar(&flags.Skip, "skip", false, "跳过首次同步，等待下一个检查周期")
	flag.BoolVar(&flags.ZipLogs, "zip-logs", false, "压缩所有日志文件")
	flag.BoolVar(&flags.UnzipLogs, "unzip-logs", false, "解压所有日志文件")
	flag.BoolVar(&flags.Sync, "sync", false, "执行单次同步检查") // 新增
	flag.BoolVar(&flags.ClearLogs, "clear-logs", false, "清除所有日志文件")
	flag.BoolVar(&flags.ClearLogs, "cl", false, "清除所有日志文件")
	flag.BoolVar(&flags.ClearCache, "clear-cache", false, "清除所有缓存")
	flag.BoolVar(&flags.ClearCache, "cc", false, "清除所有缓存")
	flag.BoolVar(&flags.ClearAll, "clear-all", false, "清除所有日志和缓存")
	flag.StringVar(&flags.RSync, "rsync", "", "指定同步的仓库（使用配置中的 minioPath）")

	flag.Usage = showHelp
	flag.Parse()

	return flags
}

func showHelp() {
	fmt.Print(`Files-API 文件同步服务

用法:
  Files-API [选项]

选项:
  -h, --help           显示帮助信息
  --skip               跳过首次同步，等待下一个检查周期
  --sync              执行单次同步检查后退出
  --rsync string      指定同步的仓库（例如：--rsync=static）
  --zip-logs          压缩所有日志文件为zip格式
  --unzip-logs        解压所有zip格式的日志文件
  --clear-logs, -cl   清除所有日志文件
  --clear-cache, -cc  清除所有缓存
  --clear-all         清除所有日志和缓存

`)
}

// 压缩指定目录下的所有日志文件
func CompressLogs(directory string) error {
	// 获取所有日志文件
	logFiles, err := filepath.Glob(filepath.Join(directory, "*.log"))
	if err != nil {
		return fmt.Errorf("获取日志文件失败: %v", err)
	}

	for _, logFile := range logFiles {
		// 创建zip文件
		zipName := strings.TrimSuffix(logFile, ".log") + ".zip"
		zipFile, err := os.Create(zipName)
		if err != nil {
			log.Printf("创建zip文件失败 %s: %v", zipName, err)
			continue
		}

		zw := zip.NewWriter(zipFile)

		// 打开源文件
		file, err := os.Open(logFile)
		if err != nil {
			log.Printf("打开日志文件失败 %s: %v", logFile, err)
			zipFile.Close()
			continue
		}

		// 创建zip条目
		w, err := zw.Create(filepath.Base(logFile))
		if err != nil {
			log.Printf("创建zip条目失败 %s: %v", logFile, err)
			file.Close()
			zipFile.Close()
			continue
		}

		// 复制文件内容
		if _, err := io.Copy(w, file); err != nil {
			log.Printf("写入zip文件失败 %s: %v", logFile, err)
			file.Close()
			zipFile.Close()
			continue
		}

		file.Close()
		zw.Close()
		zipFile.Close()

		// 删除原始日志文件
		if err := os.Remove(logFile); err != nil {
			log.Printf("删除原始日志文件失败 %s: %v", logFile, err)
			continue
		}

		log.Printf("已压缩日志文件: %s -> %s", logFile, zipName)
	}

	return nil
}

// 解压指定目录下的所有zip文件
func UncompressLogs(directory string) error {
	// 获取所有zip文件
	zipFiles, err := filepath.Glob(filepath.Join(directory, "*.zip"))
	if err != nil {
		return fmt.Errorf("获取zip文件失败: %v", err)
	}

	for _, zipFile := range zipFiles {
		// 先打开zip文件
		reader, err := zip.OpenReader(zipFile)
		if err != nil {
			log.Printf("打开zip文件失败 %s: %v", zipFile, err)
			continue
		}

		// 确保 reader 在函数结束时关闭
		func() {
			defer reader.Close()

			for _, file := range reader.File {
				// 创建日志文件
				logName := strings.TrimSuffix(zipFile, ".zip") + ".log"
				logFile, err := os.Create(logName)
				if err != nil {
					log.Printf("创建日志文件失败 %s: %v", logName, err)
					continue
				}

				// 使用闭包确保文件句柄及时关闭
				func() {
					defer logFile.Close()

					// 打开zip中的文件
					rc, err := file.Open()
					if err != nil {
						log.Printf("打开zip条目失败 %s: %v", file.Name, err)
						return
					}
					defer rc.Close()

					// 复制内容
					if _, err := io.Copy(logFile, rc); err != nil {
						log.Printf("解压文件失败 %s: %v", file.Name, err)
						return
					}
				}()
			}
		}()

		// 等待一小段时间确保所有句柄都已关闭
		time.Sleep(100 * time.Millisecond)

		// 删除zip文件
		for i := 0; i < 3; i++ { // 最多尝试3次
			err := os.Remove(zipFile)
			if err == nil {
				log.Printf("已解压并删除: %s", zipFile)
				break
			}
			if i < 2 { // 如果不是最后一次尝试，则等待后重试
				log.Printf("删除失败，等待重试: %s: %v", zipFile, err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			log.Printf("删除zip文件失败 %s: %v", zipFile, err)
		}
	}

	return nil
}

// 新增：清除日志文件
func ClearLogs(directory string) error {
	var totalSize int64
	count := 0

	// 分别获取 .log 和 .zip 文件
	patterns := []string{
		filepath.Join(directory, "Files-API-*.log"),
		filepath.Join(directory, "Files-API-*.zip"),
	}

	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			log.Printf("获取文件失败 %s: %v", pattern, err)
			continue
		}

		for _, file := range files {
			info, err := os.Stat(file)
			if err == nil {
				totalSize += info.Size()
			}
			if err := os.Remove(file); err != nil {
				log.Printf("删除文件失败 %s: %v", file, err)
				continue
			}
			count++
			log.Printf("已删除: %s", file)
		}
	}

	if count == 0 {
		log.Printf("未找到需要清理的日志文件")
		return nil
	}

	log.Printf("清理完成: 共删除 %d 个日志文件，释放空间 %.2f MB", count, float64(totalSize)/1024/1024)
	return nil
}

// 新增：清除缓存目录
func ClearCache(cacheDir string) error {
	// 检查目录是否存在
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil
	}

	// 计算目录大小
	totalSize := getDirSize(cacheDir)

	// 删除目录
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("清除缓存目录失败: %v", err)
	}

	log.Printf("清理完成: 已删除缓存目录 %s，释放空间 %.2f MB", cacheDir, float64(totalSize)/1024/1024)
	return nil
}

// 辅助函数：计算目录大小
func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
