package bootstrap

import (
	"context"
	"fmt"

	_ "github.com/mooyang-code/data-collector/internal/collector/binance" // 注册 binance 采集器
	"github.com/mooyang-code/data-collector/internal/taskmgr"
	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go/log"
)

// Services 应用服务集合
type Services struct{}

// StartBackgroundServices 启动所有后台服务
func StartBackgroundServices(ctx context.Context) (*Services, error) {
	log.Info("正在启动后台服务...")

	// 1. 初始化缓存系统（缓存远端任务配置API的结果）
	if err := initConfigCaches(); err != nil {
		log.Errorf("初始化配置缓存系统失败: %v", err)
		return nil, err
	}

	// 2. 初始化任务管理器
	if err := initTaskManager(); err != nil {
		log.Errorf("初始化任务管理器失败: %v", err)
		return nil, err
	}

	log.Info("后台服务启动完成")
	return &Services{}, nil
}

// initTaskManager 初始化任务管理器
func initTaskManager() error {
	log.Info("正在初始化任务管理器...")
	taskmgr.InitManager()
	log.Info("任务管理器初始化完成")
	return nil
}

// initConfigCaches 初始化配置缓存系统
func initConfigCaches() error {
	log.Info("正在初始化缓存系统...")

	// 创建任务实例缓存配置
	taskInstanceCache := config.CollectorTaskInstanceCache{
		AccessUrl: fmt.Sprintf("http://%s/gateway/collectmgr/GetTaskInstanceListInner", config.MooxServerServiceName),
	}

	// 初始化远程配置缓存系统
	if err := config.InitCache(
		map[string]string{
			config.MooxServerServiceName: "", // 初始化的时候为空，没关系；后面服务端心跳探测会更新该映射
		},
		fmt.Sprintf("compass://%s", config.MooxServerServiceName),
		taskInstanceCache); err != nil {
		log.Errorf("初始化任务实例缓存失败: %v", err)
		return err
	}
	log.Info("缓存系统初始化完成")
	return nil
}
