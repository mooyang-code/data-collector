package collectors

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/mooyang-code/data-collector/internal/core/collector"
	"github.com/mooyang-code/data-collector/internal/core/event"
	"github.com/mooyang-code/data-collector/internal/model/market"
	"github.com/mooyang-code/data-collector/internal/source/market/binance/api"
)

// TickerCollector 行情数据采集器
type TickerCollector struct {
	*collector.BaseCollector
	client    *api.Client
	symbols   []string
	eventBus  event.EventBus
	updateInterval time.Duration
}

// init 自注册
func init() {
	err := collector.NewBuilder().
		Source("binance", "币安").
		DataType("ticker", "行情").
		MarketType("spot", "现货").
		Description("币安24小时行情数据采集器").
		Creator(NewTickerCollector).
		Register()
	
	if err != nil {
		panic(fmt.Sprintf("注册行情采集器失败: %v", err))
	}
}

// NewTickerCollector 创建行情采集器
func NewTickerCollector(config map[string]interface{}) (collector.Collector, error) {
	client, ok := config["client"].(*api.Client)
	if !ok {
		return nil, fmt.Errorf("API客户端未提供")
	}
	
	c := &TickerCollector{
		BaseCollector:  collector.NewBaseCollector("binance_ticker", "market", "ticker"),
		client:         client,
		symbols:        []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		updateInterval: 5 * time.Second,
	}
	
	if symbols, ok := config["symbols"].([]string); ok {
		c.symbols = symbols
	}
	if interval, ok := config["update_interval"].(time.Duration); ok {
		c.updateInterval = interval
	}
	if eventBus, ok := config["event_bus"].(event.EventBus); ok {
		c.eventBus = eventBus
		// 使用适配器包装事件总线
		adapter := event.NewEventBusAdapter(eventBus)
		c.BaseCollector.SetEventBus(adapter)
	}
	
	return c, nil
}

// Initialize 初始化
func (c *TickerCollector) Initialize(ctx context.Context) error {
	if err := c.BaseCollector.Initialize(ctx); err != nil {
		return err
	}
	
	// 添加采集定时器
	c.AddTimer("collect_tickers", c.updateInterval, c.collectTickers)
	
	log.Printf("行情采集器: 已初始化，更新间隔: %v", c.updateInterval)
	
	return nil
}

// collectTickers 采集行情数据
func (c *TickerCollector) collectTickers(ctx context.Context) error {
	log.Printf("行情采集器: 开始采集行情数据")
	
	// 获取所有行情
	tickers, err := c.client.GetAllTickers()
	if err != nil {
		return fmt.Errorf("获取行情失败: %w", err)
	}
	
	// 创建批量数据
	batch := market.NewTickerBatch("binance")
	for _, ticker := range tickers {
		// 只采集配置的交易对
		for _, symbol := range c.symbols {
			if ticker.Symbol == symbol {
				batch.AddTicker(ticker)
				break
			}
		}
	}
	
	log.Printf("行情采集器: 成功采集 %d 个交易对行情", batch.Count)
	
	// 发布事件
	if c.eventBus != nil {
		dataEvent := &event.DataEvent{
			BaseEvent: *event.NewEvent(event.EventTickerCollected, c.ID(), batch).(*event.BaseEvent),
			Exchange:  "binance",
			DataType:  "ticker",
			Count:     batch.Count,
			RawData:   batch,
		}
		
		c.eventBus.PublishAsync(dataEvent)
	}
	
	return nil
}