// Package main 自注册机制简单演示（最好是中文注释！）
package main

import (
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
	
	if len(supportedTypes) == 0 {
		fmt.Println("⚠️  没有发现已注册的采集器")
		fmt.Println("这可能是因为采集器包没有被导入")
		fmt.Println("请确保在 collector_factory.go 中导入了采集器包")
	} else {
		fmt.Println("已注册的采集器类型:")
		for _, collectorType := range supportedTypes {
			fmt.Printf("  - %s\n", collectorType)
		}
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

	// 5. 演示如何检查支持的采集器类型
	fmt.Println("\n=== 支持性检查 ===")
	checkCases := []struct {
		exchange   string
		dataType   string
		marketType string
	}{
		{"binance", "symbols", "spot"},
		{"binance", "symbols", "futures"},
		{"binance", "klines", "spot"},
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

	// 6. 显示工厂支持的类型
	fmt.Println("\n=== 工厂支持的类型 ===")
	factory := app.NewCollectorFactory()
	factoryTypes := factory.GetSupportedTypes()
	
	fmt.Printf("工厂支持的类型数量: %d\n", len(factoryTypes))
	for _, factoryType := range factoryTypes {
		fmt.Printf("  - %s\n", factoryType)
	}

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("自注册机制的优势:")
	fmt.Println("1. ✅ 新增采集器只需在自己的包中添加 register.go 文件")
	fmt.Println("2. ✅ 无需修改外层的工厂代码")
	fmt.Println("3. ✅ 通过 init() 函数自动注册")
	fmt.Println("4. ✅ 支持按交易所、数据类型、市场类型查询")
	fmt.Println("5. ✅ 注册中心提供完整的管理功能")
	
	fmt.Println("\n如何添加新的采集器:")
	fmt.Println("1. 在 internal/app/{exchange}/{datatype}/ 目录下创建采集器实现")
	fmt.Println("2. 创建 register.go 文件，在 init() 函数中调用 app.RegisterCollectorCreator()")
	fmt.Println("3. 在 internal/app/collectors/import.go 中添加包导入")
	fmt.Println("4. 新采集器会自动被注册和识别")
}
