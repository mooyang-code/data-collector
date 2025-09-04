// Package main trpc服务器主程序（最好是中文注释！）
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mooyang-code/data-collector/internal/services"
	pb "github.com/mooyang-code/data-collector/proto/gen"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// 清除unix域套接字文件，避免内部使用unix域套接字的服务启动失败
	clearSocketFiles()

	// 创建trpc服务器
	s := trpc.NewServer()

	// 创建并注册数据采集器服务
	collectorService := services.NewCollectorService()
	pb.RegisterCollectorService(s, collectorService)

	log.Info("数据采集器 trpc 服务已启动")

	// 启动trpc服务器
	if err := s.Serve(); err != nil {
		log.Errorf("trpc服务器出错: %v", err)
	}
}

func clearSocketFiles() {
	files, err := filepath.Glob("./*")
	if err != nil {
		log.Errorf("读取目录失败: %v", err)
		return
	}

	for _, file := range files {
		baseFile := filepath.Base(file)
		if strings.HasPrefix(baseFile, "0.0.0.0") || strings.HasPrefix(baseFile, "127.0.0.1") {
			if err := os.Remove(file); err != nil {
				log.Errorf("删除文件 %s 失败: %v", file, err)
			}
		}
	}
}
