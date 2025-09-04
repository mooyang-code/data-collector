// Package main 自注册机制演示（最好是中文注释！）
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mooyang-code/data-collector/internal/app"
)

func main() {
	fmt.Println("=== 采集器自注册机制演示 ===")

	// 1. 显示注册中心的状态
	fmt.Println("\n=== 注册中心状态 ===")
	registry := app.GetGlobalRegistry()
	
	// 显示所有已注册的采集器类型
	supportedTypes := registry.GetSupportedTypes()
	fmt.Printf("已注册的采集器类型数量: %d\n", len(supportedTypes))
	
	for _, collectorType := range supportedTypes {
		fmt.Printf("  - %s\n", collectorType)
	}

	// 2. 按交易所分组显示
	fmt.Println("\n=== 按交易所分组 ===")
	exchanges := []string{"binance", "okx", "bybit"}
	
	for _, exchange := range exchanges {
		collectors := registry.GetCollectorsByExchange(exchange)
		fmt.Printf("%s 交易所: %d 个采集器\n", exchange, len(collectors))
		
		for key, entry := range collectors {
			fmt.Printf("  - %s: %s.%s\n", key, entry.DataType, entry.MarketType)
		}
	}

	// 3. 按数据类型分组显示
	fmt.Println("\n=== 按数据类型分组 ===")
	dataTypes := []string{"symbols", "klines"}
	
	for _, dataType := range dataTypes {
		collectors := registry.GetCollectorsByDataType(dataType)
		fmt.Printf("%s 数据类型: %d 个采集器\n", dataType, len(collectors))
		
		for key, entry := range collectors {
			fmt.Printf("  - %s: %s.%s\n", key, entry.Exchange, entry.MarketType)
		}
	}

	// 4. 按市场类型分组显示
	fmt.Println("\n=== 按市场类型分组 ===")
	marketTypes := []string{"spot", "futures"}
	
	for _, marketType := range marketTypes {
		collectors := registry.GetCollectorsByMarketType(marketType)
		fmt.Printf("%s 市场: %d 个采集器\n", marketType, len(collectors))
		
		for key, entry := range collectors {
			fmt.Printf("  - %s: %s.%s\n", key, entry.Exchange, entry.DataType)
		}
	}

	// 5. 测试采集器创建
	fmt.Println("\n=== 测试采集器创建 ===")
	factory := app.NewCollectorFactory()
	
	// 创建一个模拟配置
	mockConfig := &app.MockCollectorConfig{
		DataType:   "symbols",
		MarketType: "spot",
		Schedule: app.MockScheduleConfig{
			TriggerInterval:   "5m",
			EnableAutoRefresh: true,
			MaxRetries:        3,
			RetryInterval:     "30s",
		},
		Config: app.MockConfig{
			Filters:     []string{"USDT", "BTC"},
			BufferSize:  1000,
		},
	}

	// 测试创建不同的采集器
	testCases := []struct {
		appName       string
		collectorName string
		description   string
	}{
		{"binance", "spot_symbols_collector", "币安现货交易对采集器"},
		{"binance", "futures_symbols_collector", "币安合约交易对采集器"},
		{"binance", "spot_klines_collector", "币安现货K线采集器"},
		{"binance", "futures_klines_collector", "币安合约K线采集器"},
	}

	for _, testCase := range testCases {
		fmt.Printf("\n测试创建: %s (%s)\n", testCase.description, testCase.collectorName)
		
		// 根据采集器名称调整配置
		if testCase.collectorName == "futures_symbols_collector" || testCase.collectorName == "futures_klines_collector" {
			mockConfig.MarketType = "futures"
		} else {
			mockConfig.MarketType = "spot"
		}
		
		if testCase.collectorName == "spot_klines_collector" || testCase.collectorName == "futures_klines_collector" {
			mockConfig.DataType = "klines"
		} else {
			mockConfig.DataType = "symbols"
		}

		collector, err := factory.CreateCollector(testCase.appName, testCase.collectorName, mockConfig)
		if err != nil {
			fmt.Printf("  ❌ 创建失败: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ 创建成功\n")
		fmt.Printf("     采集器ID: %s\n", collector.GetID())
		fmt.Printf("     采集器类型: %s\n", collector.GetType())
		fmt.Printf("     数据类型: %s\n", collector.GetDataType())
		fmt.Printf("     运行状态: %v\n", collector.IsRunning())

		// 测试初始化和启动
		ctx := context.Background()
		
		if err := collector.Initialize(ctx); err != nil {
			fmt.Printf("     ❌ 初始化失败: %v\n", err)
			continue
		}
		fmt.Printf("     ✅ 初始化成功\n")

		if err := collector.StartCollection(ctx); err != nil {
			fmt.Printf("     ❌ 启动失败: %v\n", err)
			continue
		}
		fmt.Printf("     ✅ 启动成功\n")
		fmt.Printf("     运行状态: %v\n", collector.IsRunning())

		// 停止采集器
		if err := collector.StopCollection(ctx); err != nil {
			fmt.Printf("     ❌ 停止失败: %v\n", err)
		} else {
			fmt.Printf("     ✅ 停止成功\n")
			fmt.Printf("     运行状态: %v\n", collector.IsRunning())
		}
	}

	// 6. 演示如何检查支持的采集器类型
	fmt.Println("\n=== 支持性检查 ===")
	checkCases := []struct {
		exchange   string
		dataType   string
		marketType string
	}{
		{"binance", "symbols", "spot"},
		{"binance", "klines", "futures"},
		{"okx", "symbols", "spot"},
		{"bybit", "symbols", "spot"},
		{"unknown", "symbols", "spot"},
	}

	for _, checkCase := range checkCases {
		supported := registry.IsSupported(checkCase.exchange, checkCase.dataType, checkCase.marketType)
		status := "❌ 不支持"
		if supported {
			status = "✅ 支持"
		}
		fmt.Printf("%s.%s.%s: %s\n", checkCase.exchange, checkCase.dataType, checkCase.marketType, status)
	}

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("总结:")
	fmt.Println("1. 采集器通过 init() 函数自动注册到全局注册中心")
	fmt.Println("2. 新增采集器只需在自己的包中添加 register.go 文件")
	fmt.Println("3. 无需修改外层的工厂代码")
	fmt.Println("4. 支持按交易所、数据类型、市场类型查询和创建")
	fmt.Println("5. 注册中心提供完整的采集器管理功能")
}
