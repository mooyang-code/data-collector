package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mooyang-code/data-collector/internal/core/app"
	"github.com/mooyang-code/data-collector/internal/core/event"
	"github.com/mooyang-code/data-collector/internal/model/market"

	// 导入数据源和采集器，触发自注册
	_ "github.com/mooyang-code/data-collector/internal/source/market/binance"
	_ "github.com/mooyang-code/data-collector/internal/source/market/binance/collectors"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 启动币安K线数据采集演示")

	// 创建事件总线
	eventBus := event.NewMemoryEventBus(1000, 5)

	// 启动事件总线
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eventBus.Start(ctx); err != nil {
		log.Fatalf("启动事件总线失败: %v", err)
	}

	// 订阅K线数据事件
	subscribeToEvents(eventBus)

	// 创建App管理器
	config := &app.ManagerConfig{
		MaxConcurrent: 10,
		EventBus:      eventBus,
	}

	appManager := app.NewManager(config)

	// 创建币安App配置
	appConfig := &app.AppConfig{
		ID:      "binance",
		Name:    "币安",
		Type:    "binance",
		Enabled: true,
		Settings: map[string]interface{}{
			"base_url": "https://api.binance.com",
			"timeout":  10,
			"collectors": map[string]interface{}{
				"kline": map[string]interface{}{
					"enabled":   true,
					"symbols":   []string{"BTCUSDT", "ETHUSDT"},
					"intervals": []string{"1m", "5m"}, // 只使用1分钟和5分钟间隔
				},
			},
		},
		EventBus: eventBus,
	}

	// 创建并启动App
	log.Println("📱 创建币安数据采集应用...")
	if err := appManager.CreateApp(appConfig); err != nil {
		log.Fatalf("创建App失败: %v", err)
	}

	log.Println("▶️ 启动数据采集...")
	if err := appManager.StartApp("binance"); err != nil {
		log.Fatalf("启动App失败: %v", err)
	}

	// 显示运行状态
	go displayStatus(appManager)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("✅ 数据采集器正在运行，按 Ctrl+C 退出...")
	log.Println("💡 提示：K线数据每10秒(1m)和30秒(5m)采集一次")

	<-sigChan

	// 优雅关闭
	log.Println("\n🛑 正在关闭数据采集器...")

	if err := appManager.StopApp("binance"); err != nil {
		log.Printf("停止App失败: %v", err)
	}

	if err := appManager.Shutdown(); err != nil {
		log.Printf("关闭管理器失败: %v", err)
	}

	eventBus.Stop(ctx)
	log.Println("👋 数据采集器已关闭")
}

// subscribeToEvents 订阅事件【即采集器发出的结果】
func subscribeToEvents(eventBus *event.MemoryEventBus) {
	// 订阅K线数据事件
	eventBus.Subscribe("data.kline.*", func(e event.Event) error {
		if dataEvent, ok := e.(*event.DataEvent); ok {
			log.Printf("📊 收到K线数据事件: 交易所=%s, 交易对=%s, 数据类型=%s, 数量=%d",
				dataEvent.Exchange,
				dataEvent.Symbol,
				dataEvent.DataType,
				dataEvent.Count,
			)

			// 打印部分K线数据
			if batch, ok := dataEvent.RawData.(*market.KlineBatch); ok {
				klines := batch.Klines
				if len(klines) > 0 {
					// 只打印最新的K线
					latest := klines[len(klines)-1]
					log.Printf("   最新K线: 时间=%s, 开=%s, 高=%s, 低=%s, 收=%s, 量=%s",
						latest.OpenTime.Format("15:04:05"),
						latest.Open,
						latest.High,
						latest.Low,
						latest.Close,
						latest.Volume,
					)
				}
			}
		}
		return nil
	})

	// 订阅系统事件
	eventBus.Subscribe("system.*", func(e event.Event) error {
		log.Printf("📢 系统事件: %s - %v", e.Type(), e.Data())
		return nil
	})

	// 订阅错误事件
	eventBus.Subscribe("error.*", func(e event.Event) error {
		log.Printf("❌ 错误事件: %s - %v", e.Type(), e.Data())
		return nil
	})
}

// displayStatus 显示运行状态
func displayStatus(manager *app.Manager) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n===== 📊 运行状态 =====")

		// 获取所有App
		apps := manager.ListApps()
		for _, appInfo := range apps {
			status := "已停止"
			if appInfo.Status == app.AppStatusRunning {
				status = "运行中"
			}

			fmt.Printf("App: %s (%s) - 状态: %s\n", appInfo.Name, appInfo.ID, status)

			// 获取App实例
			if appInstance, err := manager.GetApp(appInfo.ID); err == nil {
				collectors := appInstance.ListCollectors()
				fmt.Printf("  采集器数量: %d\n", len(collectors))

				// 显示每个采集器的状态
				for _, collector := range collectors {
					collectorStatus := collector.GetStatus()
					fmt.Printf("  - %s: %s (运行中: %v)\n",
						collectorStatus.ID,
						collectorStatus.DataType,
						collectorStatus.IsRunning,
					)

					// 显示定时器状态
					for timerName, timerStatus := range collectorStatus.Timers {
						fmt.Printf("    • %s: 运行%d次, 错误%d次, 下次运行: %s\n",
							timerName,
							timerStatus.RunCount,
							timerStatus.ErrorCount,
							timerStatus.NextRun.Format("15:04:05"),
						)
					}
				}
			}
		}

		fmt.Println("==================\n")
	}
}
