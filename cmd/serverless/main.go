package main

import (
	"context"

	_ "trpc.group/trpc-go/trpc-log-cls"

	"github.com/mooyang-code/data-collector/internal/bootstrap"
	"github.com/mooyang-code/data-collector/pkg/config"
	"github.com/mooyang-code/data-collector/internal/cloudfunction"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// 创建默认启动器配置
	cfg := config.DefaultConfig()

	// 创建启动器
	bs := bootstrap.New(cfg)

	// 初始化启动器（统一初始化流程：配置加载 → 服务启动 → 服务注册 → 定时器注册）
	if err := bs.Initialize(context.Background()); err != nil {
		panic("failed to initialize bootstrap: " + err.Error())
	}

	// 注册并启动云函数
	cloudfunction.RegisterCloudFunction()

	// 保持运行
	log.Info("数据采集器 启动完成")
	select {}
}
