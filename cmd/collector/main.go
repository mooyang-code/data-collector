package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mooyang-code/data-collector/internal/config"
	"github.com/mooyang-code/data-collector/internal/core/app"
	"github.com/mooyang-code/data-collector/internal/core/event"

	// 导入所有数据源,及其采集器，触发自注册
	_ "github.com/mooyang-code/data-collector/internal/source/market/binance"
	_ "github.com/mooyang-code/data-collector/internal/source/market/binance/collectors"
)

func main() {
	// 解析命令行参数
	var configFile string
	flag.StringVar(&configFile, "config", "../configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 设置日志
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 打印启动信息
	fmt.Println("========================================")
	fmt.Println("     量化数据采集器 v2.0.0")
	fmt.Println("========================================")
	fmt.Printf("配置文件: %s\n", configFile)
	fmt.Println()

	// 加载配置
	mainConfig, sourceConfigs, err := config.LoadAllConfigs(configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("系统配置加载成功: %s v%s", mainConfig.System.Name, mainConfig.System.Version)
	log.Printf("环境: %s", mainConfig.System.Environment)

	// 创建主context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建事件总线
	eventBus := event.NewMemoryEventBus(mainConfig.EventBus.BufferSize, mainConfig.EventBus.Workers)
	if err := eventBus.Start(ctx); err != nil {
		log.Fatalf("启动事件总线失败: %v", err)
	}
	log.Println("事件总线已启动")

	// 注册事件处理器
	registerEventHandlers(eventBus)

	// 创建App管理器
	appManager := app.NewManager(&app.ManagerConfig{
		MaxConcurrent: 10,
		EventBus:      eventBus,
	})

	// 加载并初始化所有App
	if err := loadApps(ctx, mainConfig, sourceConfigs, appManager, eventBus); err != nil {
		log.Fatalf("加载Apps失败: %v", err)
	}

	log.Println("所有组件已启动，系统运行中...")

	// 打印运行状态
	printStatus(appManager)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("收到退出信号，开始关闭系统...")

	// 优雅关闭
	// 停止所有App
	if err := appManager.Shutdown(); err != nil {
		log.Printf("停止Apps时发生错误: %v", err)
	}

	// 停止事件总线
	eventBus.Stop(ctx)

	log.Println("系统已安全关闭")
}

// loadApps 加载所有配置的App
func loadApps(ctx context.Context, mainConfig *config.Config, sourceConfigs map[string]*config.SourceAppConfig,
	manager *app.Manager, eventBus event.EventBus) error {
	// 加载市场数据源
	for _, source := range mainConfig.Sources.Market {
		if !source.Enabled {
			log.Printf("跳过未启用的数据源: %s", source.Name)
			continue
		}

		sourceConfig, ok := sourceConfigs[source.Name]
		if !ok {
			log.Printf("未找到 %s 的配置", source.Name)
			continue
		}

		appConfig := &app.AppConfig{
			ID:      sourceConfig.App.ID,
			Name:    sourceConfig.App.Name,
			Type:    sourceConfig.App.Type,
			Enabled: true,
			Settings: map[string]interface{}{
				"api_config": sourceConfig.API,
				"auth":       sourceConfig.Auth,
				"collectors": sourceConfig.Collectors,
				"processing": sourceConfig.Processing,
				"storage":    sourceConfig.Storage,
				"monitoring": sourceConfig.Monitoring,
			},
			EventBus: eventBus,
		}

		if err := manager.CreateApp(appConfig); err != nil {
			return fmt.Errorf("创建App %s 失败: %w", source.Name, err)
		}

		if err := manager.StartApp(sourceConfig.App.ID); err != nil {
			return fmt.Errorf("启动App %s 失败: %w", source.Name, err)
		}
		log.Printf("成功加载并启动数据源: %s", source.Name)
	}

	// TODO: 加载其他类型的数据源（社交、新闻、区块链）

	return nil
}

// registerEventHandlers 注册事件处理器
func registerEventHandlers(eventBus *event.MemoryEventBus) {
	// 数据事件处理器
	eventBus.Subscribe("data.*", func(e event.Event) error {
		if dataEvent, ok := e.(*event.DataEvent); ok {
			log.Printf("📊 数据事件: %s - 交易所=%s, 交易对=%s, 数量=%d",
				e.Type(),
				dataEvent.Exchange,
				dataEvent.Symbol,
				dataEvent.Count,
			)
		}
		return nil
	})

	// 系统事件处理器
	eventBus.Subscribe("system.*", func(e event.Event) error {
		log.Printf("📢 系统事件: %s", e.Type())
		return nil
	})

	// 错误事件处理器
	eventBus.Subscribe("error.*", func(e event.Event) error {
		if errEvent, ok := e.(*event.ErrorEvent); ok {
			log.Printf("❌ 错误事件: %s - %v", errEvent.Source(), errEvent.Error)
		}
		return nil
	})

	log.Println("事件处理器注册完成")
}

// printStatus 打印系统状态
func printStatus(manager *app.Manager) {
	appInfos := manager.ListApps()

	fmt.Println("\n========== 系统状态 ==========")
	fmt.Printf("运行中的Apps: %d\n", len(appInfos))

	for _, appInfo := range appInfos {
		fmt.Printf("\nApp: %s (%s)\n", appInfo.ID, appInfo.Name)
		fmt.Printf("  类型: %s\n", appInfo.Type)
		fmt.Printf("  状态: %s\n", appInfo.Status)

		// 获取实际的App实例来显示采集器信息
		if app, err := manager.GetApp(appInfo.ID); err == nil {
			collectors := app.ListCollectors()
			fmt.Printf("  采集器数量: %d\n", len(collectors))

			for _, collector := range collectors {
				status := collector.GetStatus()
				fmt.Printf("    - %s (%s) - 运行中: %v\n",
					collector.ID(),
					collector.DataType(),
					status.IsRunning,
				)

				// 打印定时器信息
				if len(status.Timers) > 0 {
					for _, timer := range status.Timers {
						fmt.Printf("      定时器: %s, 间隔: %v, 运行次数: %d, 错误次数: %d\n",
							timer.Name,
							timer.Interval,
							timer.RunCount,
							timer.ErrorCount,
						)
					}
				} else {
					fmt.Printf("      (无定时器)\n")
				}
			}
		}
	}

	fmt.Println("==============================\n")
}
