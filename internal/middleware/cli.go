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
)

type CliFlags struct {
	Help      bool
	Skip      bool
	ZipLogs   bool
	UnzipLogs bool
	Sync      bool // 新增：执行单次同步
}

func ParseFlags() *CliFlags {
	flags := &CliFlags{}

	flag.BoolVar(&flags.Help, "h", false, "显示帮助信息")
	flag.BoolVar(&flags.Help, "help", false, "显示帮助信息")
	flag.BoolVar(&flags.Skip, "skip", false, "跳过首次同步，等待下一个检查周期")
	flag.BoolVar(&flags.ZipLogs, "zip-logs", false, "压缩所有日志文件")
	flag.BoolVar(&flags.UnzipLogs, "unzip-logs", false, "解压所有日志文件")
	flag.BoolVar(&flags.Sync, "sync", false, "执行单次同步检查") // 新增

	flag.Usage = showHelp
	flag.Parse()

	return flags
}

func showHelp() {
	fmt.Print(`Files-API 文件同步服务

用法:
  Files-API [选项]

选项:
  -h, --help       显示帮助信息
  --skip           跳过首次同步，等待下一个检查周期
  --sync          执行单次同步检查后退出
  --zip-logs      压缩所有日志文件为zip格式
  --unzip-logs    解压所有zip格式的日志文件

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
		reader, err := zip.OpenReader(zipFile)
		if err != nil {
			log.Printf("打开zip文件失败 %s: %v", zipFile, err)
			continue
		}

		for _, file := range reader.File {
			// 创建日志文件
			logName := strings.TrimSuffix(zipFile, ".zip") + ".log"
			logFile, err := os.Create(logName)
			if err != nil {
				log.Printf("创建日志文件失败 %s: %v", logName, err)
				continue
			}

			// 打开zip中的文件
			rc, err := file.Open()
			if err != nil {
				log.Printf("打开zip条目失败 %s: %v", file.Name, err)
				logFile.Close()
				continue
			}

			// 复制内容
			if _, err := io.Copy(logFile, rc); err != nil {
				log.Printf("解压文件失败 %s: %v", file.Name, err)
				rc.Close()
				logFile.Close()
				continue
			}

			rc.Close()
			logFile.Close()

			// 删除zip文件
			if err := os.Remove(zipFile); err != nil {
				log.Printf("删除zip文件失败 %s: %v", zipFile, err)
				continue
			}

			log.Printf("已解压日志文件: %s -> %s", zipFile, logName)
		}
		reader.Close()
	}

	return nil
}
