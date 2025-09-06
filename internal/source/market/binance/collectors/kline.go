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

// KlineCollector K线数据采集器
type KlineCollector struct {
	*collector.BaseCollector
	client    *api.Client
	symbols   []string
	intervals []string
	eventBus  event.EventBus
}

// KlineConfig K线采集器配置
type KlineConfig struct {
	Symbols   []string
	Intervals []string
	EventBus  event.EventBus
}

// init 自注册到采集器注册中心
func init() {
	err := collector.NewBuilder().
		Source("binance", "币安").
		DataType("kline", "K线").
		MarketType("spot", "现货").
		Description("币安现货K线数据采集器").
		Creator(NewKlineCollector).
		Register()
	
	if err != nil {
		panic(fmt.Sprintf("注册K线采集器失败: %v", err))
	}
}

// NewKlineCollector 创建K线采集器
func NewKlineCollector(config map[string]interface{}) (collector.Collector, error) {
	// 解析配置
	cfg := &KlineConfig{
		Symbols:   []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		Intervals: []string{market.Interval1m, market.Interval5m, market.Interval1h},
	}
	
	if symbols, ok := config["symbols"].([]string); ok {
		cfg.Symbols = symbols
	}
	if intervals, ok := config["intervals"].([]string); ok {
		cfg.Intervals = intervals
	}
	if eventBus, ok := config["event_bus"].(event.EventBus); ok {
		cfg.EventBus = eventBus
	}
	if client, ok := config["client"].(*api.Client); ok {
		c := &KlineCollector{
			BaseCollector: collector.NewBaseCollector("binance_kline", "market", "kline"),
			client:        client,
			symbols:       cfg.Symbols,
			intervals:     cfg.Intervals,
			eventBus:      cfg.EventBus,
		}
		
		if cfg.EventBus != nil {
			// 使用适配器包装事件总线
			adapter := event.NewEventBusAdapter(cfg.EventBus)
			c.BaseCollector.SetEventBus(adapter)
		}
		
		return c, nil
	}
	
	return nil, fmt.Errorf("API客户端未提供")
}

// Initialize 初始化
func (c *KlineCollector) Initialize(ctx context.Context) error {
	log.Printf("K线采集器: 开始初始化")
	
	// 调用基类初始化
	if err := c.BaseCollector.Initialize(ctx); err != nil {
		return err
	}
	
	// 为每个时间间隔添加定时器
	for _, interval := range c.intervals {
		duration, err := market.IntervalDuration(interval)
		if err != nil {
			return fmt.Errorf("无效的时间间隔 %s: %w", interval, err)
		}
		
		// 创建定时器
		timerName := fmt.Sprintf("collect_%s", interval)
		handler := c.createCollectHandler(interval)
		
		if err := c.AddTimer(timerName, duration, handler); err != nil {
			return fmt.Errorf("添加定时器失败 %s: %w", timerName, err)
		}
		
		log.Printf("K线采集器: 添加定时器 %s, 间隔 %v", timerName, duration)
	}
	
	// 添加数据清理定时器
	c.AddTimer("cleanup", 24*time.Hour, c.cleanupOldData)
	
	return nil
}

// createCollectHandler 创建采集处理函数
func (c *KlineCollector) createCollectHandler(interval string) collector.TimerHandler {
	return func(ctx context.Context) error {
		log.Printf("K线采集器: 开始采集 %s K线数据", interval)
		
		for _, symbol := range c.symbols {
			// 采集数据
			klines, err := c.client.GetKlines(symbol, interval, 10)
			if err != nil {
				log.Printf("采集 %s %s K线失败: %v", symbol, interval, err)
				continue
			}
			
			log.Printf("K线采集器: 成功采集 %s %s K线 %d 条", symbol, interval, len(klines))
			
			// 创建批量数据
			batch := market.NewKlineBatch("binance", symbol, interval)
			for _, kline := range klines {
				batch.AddKline(kline)
			}
			
			// 发布事件
			if c.eventBus != nil {
				dataEvent := &event.DataEvent{
					BaseEvent: *event.NewEvent(event.EventKlineCollected, c.ID(), batch).(*event.BaseEvent),
					Exchange:  "binance",
					Symbol:    symbol,
					DataType:  "kline",
					Count:     len(klines),
					RawData:   batch,
				}
				
				c.eventBus.PublishAsync(dataEvent)
			}
		}
		
		return nil
	}
}

// cleanupOldData 清理旧数据
func (c *KlineCollector) cleanupOldData(ctx context.Context) error {
	log.Printf("K线采集器: 执行数据清理任务")
	// 这里可以实现清理逻辑
	return nil
}