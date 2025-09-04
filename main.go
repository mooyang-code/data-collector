// Package main 重构后的量化数据采集器主程序（最好是中文注释！）
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/mooyang-code/data-collector/internal/app"
	"github.com/mooyang-code/data-collector/internal/services"
	pb "github.com/mooyang-code/data-collector/proto/gen"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"

	// 导入采集器包以触发自注册
	_ "github.com/mooyang-code/data-collector/internal/app/binance/klines"
	_ "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
	_ "github.com/mooyang-code/data-collector/internal/app/okx/klines"
	_ "github.com/mooyang-code/data-collector/internal/app/okx/symbols"
)

// 命令行参数
var (
	configPath = flag.String("config", "configs", "配置文件目录路径")
)

func main() {
	flag.Parse()
	fmt.Println("=== 量化数据采集器 ===")
	fmt.Printf("配置目录: %s\n", *configPath)


	// 创建App工厂
	appFactory := app.NewAppFactory()

	// 创建App管理器
	appManager := app.NewAppManager(appFactory)

	// 加载配置
	if err := appManager.LoadConfig(*configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 启动App管理器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := appManager.Start(ctx); err != nil {
		log.Fatalf("启动App管理器失败: %v", err)
	}

	// 启动trpc服务器
	startTrpcServer()
}

// startTrpcServer 启动trpc服务器
func startTrpcServer() {
	fmt.Println("🚀 启动trpc服务器...")

	// 创建trpc服务器
	s := trpc.NewServer()

	// 空服务
	collectorService := services.NewCollectorService()
	pb.RegisterCollectorService(s, collectorService)

	if err := s.Serve(); err != nil {
		log.Errorf("trpc服务器出错: %v", err)
	}
	log.Info("✅ trpc服务器已启动")
}
