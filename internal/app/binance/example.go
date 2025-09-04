// +build ignore

// Package binance 币安交易所示例用法
package binance

import (
	"context"
	"time"

	binanceKlines "github.com/mooyang-code/data-collector/internal/app/binance/klines"
	"github.com/mooyang-code/data-collector/internal/app/binance/klines/futures"
	"github.com/mooyang-code/data-collector/internal/app/binance/klines/spot"
	binanceSymbols "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
	"github.com/mooyang-code/data-collector/internal/datatype/klines"
	"github.com/mooyang-code/data-collector/internal/datatype/symbols"
	"trpc.group/trpc-go/trpc-go/log"
)

// ExampleUsage 展示如何使用新的币安采集器架构
func ExampleUsage() {
	ctx := context.Background()

	// 1. 使用币安现货K线采集器
	log.Info("=== 币安现货K线采集器示例 ===")
	spotKlineExample(ctx)

	// 2. 使用币安合约K线采集器
	log.Info("=== 币安合约K线采集器示例 ===")
	futuresKlineExample(ctx)

	// 3. 使用币安交易对采集器
	log.Info("=== 币安交易对采集器示例 ===")
	symbolsExample(ctx)

	// 4. 使用应用级别的封装
	log.Info("=== 应用级别封装示例 ===")
	appLevelExample(ctx)
}

// spotKlineExample 现货K线采集器示例
func spotKlineExample(ctx context.Context) {
	// 创建现货K线配置
	config := &binanceKlines.BinanceKlineConfig{
		Exchange: "binance_spot",
		BaseURL:  "https://api.binance.com",
		Symbols:  []string{"BTCUSDT", "ETHUSDT"},
		Intervals: []klines.Interval{
			klines.Interval1m,
			klines.Interval5m,
			klines.Interval1h,
		},
		TimerConfigs:      binanceKlines.DefaultBinanceKlineConfig().TimerConfigs,
		HTTPTimeout:       30 * time.Second,
		EnableBackfill:    true,
		BackfillDays:      1,
		EnablePersistence: true,
		BufferSize:        1000,
	}

	// 创建采集器
	collector, err := binanceKlines.NewBinanceKlineCollector(config)
	if err != nil {
		log.Errorf("创建现货K线采集器失败: %v", err)
		return
	}

	// 初始化并启动
	if err := collector.Initialize(ctx); err != nil {
		log.Errorf("初始化现货K线采集器失败: %v", err)
		return
	}

	if err := collector.StartCollection(ctx); err != nil {
		log.Errorf("启动现货K线采集失败: %v", err)
		return
	}

	// 监听事件
	go func() {
		events := collector.Events()
		for event := range events {
			log.Infof("收到现货K线事件: symbol=%s, interval=%s, source=%s",
				event.Record.Symbol, event.Record.Interval, event.Source)
		}
	}()

	// 运行一段时间后停止
	time.Sleep(10 * time.Second)
	if err := collector.StopCollection(ctx); err != nil {
		log.Errorf("停止现货K线采集失败: %v", err)
	}
}

// futuresKlineExample 合约K线采集器示例
func futuresKlineExample(ctx context.Context) {
	// 创建合约K线配置
	config := &binanceKlines.BinanceKlineConfig{
		Exchange: "binance_futures",
		BaseURL:  "https://fapi.binance.com", // 合约API
		Symbols:  []string{"BTCUSDT", "ETHUSDT"},
		Intervals: []klines.Interval{
			klines.Interval1m,
			klines.Interval15m,
			klines.Interval1h,
		},
		TimerConfigs:      binanceKlines.DefaultBinanceKlineConfig().TimerConfigs,
		HTTPTimeout:       30 * time.Second,
		RequestLimit:      2400, // 合约API限制更高
		EnableBackfill:    true,
		BackfillDays:      1,
		EnablePersistence: true,
		BufferSize:        1000,
	}

	// 创建采集器
	collector, err := binanceKlines.NewBinanceKlineCollector(config)
	if err != nil {
		log.Errorf("创建合约K线采集器失败: %v", err)
		return
	}

	// 初始化并启动
	if err := collector.Initialize(ctx); err != nil {
		log.Errorf("初始化合约K线采集器失败: %v", err)
		return
	}

	if err := collector.StartCollection(ctx); err != nil {
		log.Errorf("启动合约K线采集失败: %v", err)
		return
	}

	// 监听事件
	go func() {
		events := collector.Events()
		for event := range events {
			log.Infof("收到合约K线事件: symbol=%s, interval=%s, source=%s",
				event.Record.Symbol, event.Record.Interval, event.Source)
		}
	}()

	// 运行一段时间后停止
	time.Sleep(10 * time.Second)
	if err := collector.StopCollection(ctx); err != nil {
		log.Errorf("停止合约K线采集失败: %v", err)
	}
}

// symbolsExample 交易对采集器示例
func symbolsExample(ctx context.Context) {
	// 创建现货交易对配置
	config := binanceSymbols.DefaultBinanceSymbolConfig()
	config.RefreshInterval = 1 * time.Minute
	config.SymbolFilter.AllowedQuoteAssets = []string{"USDT", "BTC"}

	// 创建采集器
	collector, err := binanceSymbols.NewBinanceSymbolCollector(config)
	if err != nil {
		log.Errorf("创建交易对采集器失败: %v", err)
		return
	}

	// 初始化并启动
	if err := collector.Initialize(ctx); err != nil {
		log.Errorf("初始化交易对采集器失败: %v", err)
		return
	}

	if err := collector.StartCollection(ctx); err != nil {
		log.Errorf("启动交易对采集失败: %v", err)
		return
	}

	// 监听事件
	go func() {
		events := collector.Events()
		for event := range events {
			switch event.Type {
			case symbols.SymbolAdded:
				log.Infof("新增交易对: %s", event.Symbol.Symbol)
			case symbols.SymbolUpdated:
				log.Infof("更新交易对: %s", event.Symbol.Symbol)
			case symbols.SymbolRemoved:
				log.Infof("移除交易对: %s", event.Symbol.Symbol)
			case symbols.SnapshotEnd:
				log.Infof("交易对快照完成")
			}
		}
	}()

	// 运行一段时间后查询数据
	time.Sleep(5 * time.Second)

	// 获取统计信息
	stats := collector.GetStats()
	log.Infof("交易对统计: 总数=%d, 活跃=%d, 现货=%d",
		stats.TotalCount, stats.ActiveCount, stats.SpotCount)

	// 获取USDT交易对
	usdtSymbols := collector.GetSymbolsByQuoteAsset("USDT")
	log.Infof("USDT交易对数量: %d", len(usdtSymbols))

	// 停止采集
	if err := collector.StopCollection(ctx); err != nil {
		log.Errorf("停止交易对采集失败: %v", err)
	}
}

// appLevelExample 应用级别封装示例
func appLevelExample(ctx context.Context) {
	// 使用现货K线应用
	spotApp := spot.NewBinanceSpotKlineApp("binance_spot_demo")

	// 模拟配置（实际使用中从配置文件加载）
	// config := &configs.AppConfig{...}
	// if err := spotApp.Initialize(config); err != nil {
	//     log.Errorf("初始化现货应用失败: %v", err)
	//     return
	// }

	// if err := spotApp.Start(ctx); err != nil {
	//     log.Errorf("启动现货应用失败: %v", err)
	//     return
	// }

	log.Infof("现货应用创建成功: %s", spotApp.GetID())

	// 使用合约K线应用
	futuresApp := futures.NewBinanceFuturesKlineApp("binance_futures_demo")

	// 模拟配置
	// if err := futuresApp.Initialize(config); err != nil {
	//     log.Errorf("初始化合约应用失败: %v", err)
	//     return
	// }

	// if err := futuresApp.Start(ctx); err != nil {
	//     log.Errorf("启动合约应用失败: %v", err)
	//     return
	// }

	log.Infof("合约应用创建成功: %s", futuresApp.GetID())

	// 停止应用
	// futuresApp.Stop(ctx)
	// spotApp.Stop(ctx)
}

// DirectAdapterUsage 直接使用适配器的示例
func DirectAdapterUsage() {
	ctx := context.Background()

	// 直接使用K线适配器
	klineAdapter := binanceKlines.NewBinanceKlineAdapter("https://api.binance.com")

	// 获取历史K线
	klineData, err := klineAdapter.FetchHistoryKlines(
		ctx,
		"BTCUSDT",
		klines.Interval1h,
		time.Now().Add(-24*time.Hour),
		time.Now(),
		24,
	)
	if err != nil {
		log.Errorf("获取历史K线失败: %v", err)
		return
	}

	log.Infof("获取到 %d 条K线数据", len(klineData))

	// 直接使用交易对适配器
	symbolAdapter := binanceSymbols.NewBinanceSymbolAdapter("https://api.binance.com", "binance")

	// 获取所有交易对
	allSymbols, err := symbolAdapter.FetchAll(ctx)
	if err != nil {
		log.Errorf("获取交易对失败: %v", err)
		return
	}

	log.Infof("获取到 %d 个交易对", len(allSymbols))

	// 获取单个交易对
	btcSymbol, err := symbolAdapter.FetchSymbol(ctx, "BTCUSDT")
	if err != nil {
		log.Errorf("获取BTCUSDT交易对失败: %v", err)
		return
	}

	log.Infof("BTCUSDT交易对信息: 状态=%s, 类型=%s", btcSymbol.Status, btcSymbol.Type)
}
