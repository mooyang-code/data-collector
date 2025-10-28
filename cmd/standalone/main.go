package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mooyang-code/data-collector/internal/bootstrap"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("     量化数据采集器 v2.0.0     ")
	fmt.Println("========================================")

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建启动器配置
	cfg := bootstrap.DefaultConfig()

	// 在独立运行模式下，这些配置将使用默认值
	// 在云函数模式下，这些信息会在运行时从 functioncontext 动态获取

	// 创建启动器
	bs := bootstrap.New(cfg)

	// 初始化系统
	if err := bs.Init(ctx); err != nil {
		log.Fatalf("系统初始化失败: %v", err)
	}

	// 启动系统
	if err := bs.Start(ctx); err != nil {
		log.Fatalf("系统启动失败: %v", err)
	}

	fmt.Printf("系统启动成功，状态: %s\n", bs.GetState())
	fmt.Printf("节点信息: %+v\n", bs.GetNodeInfo())

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n收到退出信号，开始关闭系统...")

	// 优雅关闭
	if err := bs.Stop(ctx); err != nil {
		log.Printf("系统关闭时发生错误: %v", err)
	}

	fmt.Println("系统已安全关闭")
}