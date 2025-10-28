package exchanges

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// HuobiCollector Huobi交易所数据采集器
type HuobiCollector struct {
	*BaseCollector
	apiKey    string
	apiSecret string
	baseURL   string
}

// NewHuobiCollector 创建Huobi采集器
func NewHuobiCollector() collector.Collector {
	return &HuobiCollector{
		BaseCollector: NewBaseCollector("huobi", model.CollectorTypeHuobi),
		baseURL:       "https://api.huobi.pro",
	}
}

func (c *HuobiCollector) Collect(ctx context.Context, taskType model.TaskType, params *model.CollectParams) (*model.CollectResult, error) {
	if c.GetState() != collector.StateRunning {
		return nil, fmt.Errorf("collector not running")
	}

	switch taskType {
	case model.TaskTypeKLine:
		return c.collectKLines(ctx, params)
	case model.TaskTypeTicker:
		return c.collectTicker(ctx, params)
	case model.TaskTypeOrderBook:
		return c.collectOrderBook(ctx, params)
	case model.TaskTypeTrade:
		return c.collectTrades(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", taskType)
	}
}

func (c *HuobiCollector) collectKLines(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟Huobi K线数据采集
	now := time.Now()
	
	// Huobi的K线数据格式
	klines := []map[string]interface{}{
		{
			"id":     now.Add(-time.Minute).Unix(),
			"open":   50000.0,
			"close":  50050.0,
			"low":    49900.0,
			"high":   50100.0,
			"amount": 1.23456789,
			"vol":    61728.95,
			"count":  100,
		},
	}
	
	return &model.CollectResult{
		Data:      klines,
		Count:     len(klines),
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "huobi",
			"type":     "klines",
			"symbol":   params.Symbol,
			"interval": params.Interval,
		},
	}, nil
}

func (c *HuobiCollector) collectTicker(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟Huobi ticker数据采集
	now := time.Now()
	
	ticker := map[string]interface{}{
		"symbol": params.Symbol,
		"open":   49900.0,
		"high":   50100.0,
		"low":    49800.0,
		"close":  50050.0,
		"amount": 12345.67890,
		"vol":    617283950.0,
		"count":  98765,
		"bid":    50049.0,
		"bidSize": 2.3,
		"ask":    50051.0,
		"askSize": 1.5,
	}
	
	return &model.CollectResult{
		Data:      ticker,
		Count:     1,
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "huobi",
			"type":     "ticker",
			"symbol":   params.Symbol,
		},
	}, nil
}

func (c *HuobiCollector) collectOrderBook(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟Huobi订单簿数据采集
	now := time.Now()
	
	// 获取深度参数
	limit := 10
	if limitStr, ok := params.Options["limit"].(string); ok {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	
	// 构造模拟订单簿数据
	bids := make([][]float64, limit)
	asks := make([][]float64, limit)
	
	basePrice := 50000.0
	for i := 0; i < limit; i++ {
		bidPrice := basePrice - float64(i)*0.1
		askPrice := basePrice + float64(i)*0.1
		
		bids[i] = []float64{bidPrice, 1.0 + float64(i)*0.1}
		asks[i] = []float64{askPrice, 1.0 + float64(i)*0.1}
	}
	
	orderBook := map[string]interface{}{
		"bids": bids,
		"asks": asks,
		"ts":   now.UnixMilli(),
		"version": now.UnixNano(),
	}
	
	return &model.CollectResult{
		Data:      orderBook,
		Count:     1,
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "huobi",
			"type":     "orderbook",
			"symbol":   params.Symbol,
			"limit":    limit,
		},
	}, nil
}

func (c *HuobiCollector) collectTrades(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟Huobi交易数据采集
	now := time.Now()
	
	// 获取数量限制
	limit := 100
	if limitStr, ok := params.Options["limit"].(string); ok {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	
	// 构造模拟交易数据
	trades := make([]map[string]interface{}, limit)
	for i := 0; i < limit; i++ {
		direction := "buy"
		if i%2 == 0 {
			direction = "sell"
		}
		
		trades[i] = map[string]interface{}{
			"id":        now.UnixNano() + int64(i),
			"price":     50000.0 + float64(i-limit/2)*0.1,
			"amount":    0.1 + float64(i)*0.001,
			"direction": direction,
			"ts":        now.Add(-time.Duration(limit-i)*time.Second).UnixMilli(),
		}
	}
	
	return &model.CollectResult{
		Data:      trades,
		Count:     len(trades),
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "huobi",
			"type":     "trades",
			"symbol":   params.Symbol,
			"limit":    limit,
		},
	}, nil
}