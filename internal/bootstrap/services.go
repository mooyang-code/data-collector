package bootstrap

import (
	"context"
	"fmt"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go/log"

	_ "github.com/mooyang-code/data-collector/internal/collector/exchanges" // 注册采集器
)

// Services 应用服务集合
type Services struct {
	// 采集器管理器（核心服务）
	CollectorManager collector.Manager
}

// StartBackgroundServices 启动所有后台服务
// 简化版：只启动采集器相关服务
func StartBackgroundServices(ctx context.Context) (*Services, error) {
	log.Info("正在启动后台服务...")

	// 1. 初始化缓存系统
	if err := initConfigCaches(); err != nil {
		log.Errorf("初始化配置缓存系统失败: %v", err)
		return nil, err
	}

	// 2. 创建采集器管理器
	collectorManager, err := createCollectorManager()
	if err != nil {
		log.Errorf("创建采集器管理器失败: %v", err)
		return nil, err
	}

	log.Info("后台服务启动完成")
	return &Services{
		CollectorManager: collectorManager,
	}, nil
}

// createCollectorManager 创建采集器管理器
func createCollectorManager() (collector.Manager, error) {
	log.Info("正在创建采集器管理器...")

	// 创建采集器管理器
	collectorManager := collector.NewManager(nil)

	log.Info("采集器管理器创建成功")
	return collectorManager, nil
}

// initConfigCaches 初始化配置缓存系统
func initConfigCaches() error {
	log.Info("正在初始化缓存系统...")

	// 创建任务实例缓存配置
	taskInstanceCache := config.CollectorTaskInstanceCache{
		AccessUrl: fmt.Sprintf("http://%s/gateway/collector/GetTaskInstanceListInner", config.MooxServerServiceName),
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
