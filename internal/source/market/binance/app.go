package binance

import (
	"context"
	"fmt"
	"time"
	
	"github.com/mooyang-code/data-collector/internal/core/app"
	"github.com/mooyang-code/data-collector/internal/core/collector"
	"github.com/mooyang-code/data-collector/internal/source/market/binance/api"
)

// BinanceApp Binance数据采集应用
type BinanceApp struct {
	*app.BaseApp
	client     *api.Client
	config     *Config
	collectors map[string]interface{} // 采集器配置
}

// Config Binance配置
type Config struct {
	BaseURL   string
	APIKey    string
	APISecret string
}

// init 自注册
func init() {
	err := app.RegisterCreator(
		"binance",
		"币安",
		"币安交易所数据采集应用",
		app.SourceTypeMarket,
		NewBinanceApp,
	)
	if err != nil {
		panic(fmt.Sprintf("注册Binance App失败: %v", err))
	}
}

// NewBinanceApp 创建Binance应用
func NewBinanceApp(config *app.AppConfig) (app.App, error) {
	// 解析配置
	cfg := &Config{
		BaseURL:   "https://api.binance.com",
		APIKey:    "",
		APISecret: "",
	}
	
	// 优先从 api_config 中获取配置
	if apiConfig, ok := config.Settings["api_config"].(map[string]interface{}); ok {
		if baseURL, ok := apiConfig["base_url"].(string); ok {
			cfg.BaseURL = baseURL
		}
	}
	
	// 从 auth 中获取认证信息
	if auth, ok := config.Settings["auth"].(map[string]interface{}); ok {
		if apiKey, ok := auth["api_key"].(string); ok {
			cfg.APIKey = apiKey
		}
		if apiSecret, ok := auth["api_secret"].(string); ok {
			cfg.APISecret = apiSecret
		}
	}
	
	// 兼容旧配置格式
	if baseURL, ok := config.Settings["base_url"].(string); ok {
		cfg.BaseURL = baseURL
	}
	if apiKey, ok := config.Settings["api_key"].(string); ok {
		cfg.APIKey = apiKey
	}
	if apiSecret, ok := config.Settings["api_secret"].(string); ok {
		cfg.APISecret = apiSecret
	}
	
	// 创建API客户端
	client := api.NewClient(cfg.BaseURL, cfg.APIKey, cfg.APISecret)
	
	// 创建App
	binanceApp := &BinanceApp{
		BaseApp: app.NewBaseApp("binance", "币安", app.SourceTypeMarket),
		client:  client,
		config:  cfg,
	}
	
	// 保存采集器配置
	if collectors, ok := config.Settings["collectors"].(map[string]interface{}); ok {
		binanceApp.collectors = collectors
	}
	
	// 事件总线会通过SetEventBus方法设置
	
	return binanceApp, nil
}

// genericEvent 通用事件类型
type genericEvent struct {
	id   string
	typ  string
	src  string
	ts   time.Time
	data interface{}
}

func (g *genericEvent) ID() string         { return g.id }
func (g *genericEvent) Type() string       { return g.typ }
func (g *genericEvent) Source() string     { return g.src }
func (g *genericEvent) Timestamp() time.Time { return g.ts }
func (g *genericEvent) Data() interface{}  { return g.data }

// Initialize 初始化
func (a *BinanceApp) Initialize(ctx context.Context) error {
	// 测试API连接
	if err := a.client.Ping(); err != nil {
		return fmt.Errorf("连接Binance API失败: %w", err)
	}
	
	// 自动发现并注册采集器
	a.registerCollectors()
	
	// 调用基类初始化（会初始化所有已注册的采集器）
	if err := a.BaseApp.Initialize(ctx); err != nil {
		return err
	}
	
	return nil
}

// SetEventBus 设置事件总线
func (a *BinanceApp) SetEventBus(eventBus app.EventBus) {
	a.BaseApp.SetEventBus(eventBus)
}

// registerCollectors 注册采集器
func (a *BinanceApp) registerCollectors() {
	// 获取事件总线
	eventBus := a.BaseApp.GetEventBus()
	
	// 注册K线采集器
	if klineCfg, ok := a.collectors["kline"].(map[string]interface{}); ok && klineCfg["enabled"] == true {
		klineConfig := map[string]interface{}{
			"client":    a.client,
			"event_bus": eventBus,
		}
		
		// 合并配置
		for k, v := range klineCfg {
			klineConfig[k] = v
		}
		
		if klineCollector, err := collector.GetRegistry().CreateCollector("binance", "kline", klineConfig); err == nil {
		// 使用适配器包装采集器
		adaptedCollector := app.NewCollectorAdapter(klineCollector)
		a.RegisterCollector(adaptedCollector)
		if baseCollector, ok := klineCollector.(*collector.BaseCollector); ok && eventBus != nil {
			// 创建适配器
			adapter := collector.NewAppEventBusAdapter(
				func(e interface{}) error {
					if appEvent, ok := e.(app.Event); ok {
						return eventBus.Publish(appEvent)
					}
					// 包装成 app.Event
					return eventBus.Publish(&genericEvent{
						id:   time.Now().Format("20060102150405"),
						typ:  "generic.event",
						src:  "binance",
						ts:   time.Now(),
						data: e,
					})
				},
				func(e interface{}) {
					if appEvent, ok := e.(app.Event); ok {
						eventBus.Publish(appEvent)
					} else {
						eventBus.Publish(&genericEvent{
							id:   time.Now().Format("20060102150405"),
							typ:  "generic.event",
							src:  "binance",
							ts:   time.Now(),
							data: e,
						})
					}
				},
			)
			baseCollector.SetEventBus(adapter)
		}
		}
	}
	
	// 注册行情采集器
	if tickerCfg, ok := a.collectors["ticker"].(map[string]interface{}); ok && tickerCfg["enabled"] == true {
		tickerConfig := map[string]interface{}{
			"client":    a.client,
			"event_bus": eventBus,
		}
		
		// 合并配置
		for k, v := range tickerCfg {
			tickerConfig[k] = v
		}
		
		if tickerCollector, err := collector.GetRegistry().CreateCollector("binance", "ticker", tickerConfig); err == nil {
			// 使用适配器包装采集器
			adaptedCollector := app.NewCollectorAdapter(tickerCollector)
			a.RegisterCollector(adaptedCollector)
			if baseCollector, ok := tickerCollector.(*collector.BaseCollector); ok && eventBus != nil {
				// 创建适配器
				adapter := collector.NewAppEventBusAdapter(
					func(e interface{}) error {
						if appEvent, ok := e.(app.Event); ok {
							return eventBus.Publish(appEvent)
						}
						// 包装成 app.Event
						return eventBus.Publish(&genericEvent{
							id:   time.Now().Format("20060102150405"),
							typ:  "generic.event",
							src:  "binance",
							ts:   time.Now(),
							data: e,
						})
					},
					func(e interface{}) {
						if appEvent, ok := e.(app.Event); ok {
							eventBus.Publish(appEvent)
						} else {
							eventBus.Publish(&genericEvent{
								id:   time.Now().Format("20060102150405"),
								typ:  "generic.event", 
								src:  "binance",
								ts:   time.Now(),
								data: e,
							})
						}
					},
				)
				baseCollector.SetEventBus(adapter)
			}
		}
	}
}