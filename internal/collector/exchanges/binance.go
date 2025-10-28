package exchanges

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// BinanceCollector Binance交易所数据采集器
type BinanceCollector struct {
	*BaseCollector
	apiKey    string
	apiSecret string
	baseURL   string
}

// NewBinanceCollector 创建Binance采集器
func NewBinanceCollector() collector.Collector {
	return &BinanceCollector{
		BaseCollector: NewBaseCollector("binance", model.CollectorTypeBinance),
		baseURL:       "https://api.binance.com",
	}
}

func (c *BinanceCollector) Collect(ctx context.Context, taskType model.TaskType, params *model.CollectParams) (*model.CollectResult, error) {
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

func (c *BinanceCollector) collectKLines(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟K线数据采集
	now := time.Now()
	
	// 构造模拟K线数据
	klines := []map[string]interface{}{
		{
			"symbol":     params.Symbol,
			"interval":   params.Interval,
			"open_time":  now.Add(-time.Minute).Unix(),
			"close_time": now.Unix(),
			"open":       "50000.00",
			"high":       "50100.00",
			"low":        "49900.00",
			"close":      "50050.00",
			"volume":     "1.23456789",
			"trades":     100,
		},
	}
	
	return &model.CollectResult{
		Data:      klines,
		Count:     len(klines),
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "binance",
			"type":     "klines",
			"symbol":   params.Symbol,
			"interval": params.Interval,
		},
	}, nil
}

func (c *BinanceCollector) collectTicker(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟ticker数据采集
	now := time.Now()
	
	ticker := map[string]interface{}{
		"symbol":             params.Symbol,
		"price_change":       "150.00",
		"price_change_percent": "0.30",
		"weighted_avg_price": "50025.00",
		"prev_close_price":   "49900.00",
		"last_price":         "50050.00",
		"last_qty":           "0.10000000",
		"bid_price":          "50049.00",
		"ask_price":          "50051.00",
		"open_price":         "49900.00",
		"high_price":         "50100.00",
		"low_price":          "49800.00",
		"volume":             "12345.67890000",
		"quote_volume":       "617283950.00000000",
		"open_time":          now.Add(-24*time.Hour).Unix(),
		"close_time":         now.Unix(),
		"count":              98765,
	}
	
	return &model.CollectResult{
		Data:      ticker,
		Count:     1,
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "binance",
			"type":     "ticker",
			"symbol":   params.Symbol,
		},
	}, nil
}

func (c *BinanceCollector) collectOrderBook(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟订单簿数据采集
	now := time.Now()
	
	// 获取深度参数
	limit := 10
	if limitStr, ok := params.Options["limit"].(string); ok {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	
	// 构造模拟订单簿数据
	bids := make([][]string, limit)
	asks := make([][]string, limit)
	
	basePrice := 50000.0
	for i := 0; i < limit; i++ {
		bidPrice := basePrice - float64(i)*0.1
		askPrice := basePrice + float64(i)*0.1
		
		bids[i] = []string{
			fmt.Sprintf("%.2f", bidPrice),
			fmt.Sprintf("%.8f", 1.0+float64(i)*0.1),
		}
		asks[i] = []string{
			fmt.Sprintf("%.2f", askPrice),
			fmt.Sprintf("%.8f", 1.0+float64(i)*0.1),
		}
	}
	
	orderBook := map[string]interface{}{
		"symbol":       params.Symbol,
		"last_update_id": now.UnixNano(),
		"bids":         bids,
		"asks":         asks,
	}
	
	return &model.CollectResult{
		Data:      orderBook,
		Count:     1,
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "binance",
			"type":     "orderbook",
			"symbol":   params.Symbol,
			"limit":    limit,
		},
	}, nil
}

func (c *BinanceCollector) collectTrades(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟交易数据采集
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
		trades[i] = map[string]interface{}{
			"id":           now.UnixNano() + int64(i),
			"price":        fmt.Sprintf("%.2f", 50000.0+float64(i-limit/2)*0.1),
			"qty":          fmt.Sprintf("%.8f", 0.1+float64(i)*0.001),
			"quote_qty":    fmt.Sprintf("%.8f", 5000.0+float64(i)),
			"time":         now.Add(-time.Duration(limit-i)*time.Second).UnixMilli(),
			"is_buyer_maker": i%2 == 0,
		}
	}
	
	return &model.CollectResult{
		Data:      trades,
		Count:     len(trades),
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "binance",
			"type":     "trades",
			"symbol":   params.Symbol,
			"limit":    limit,
		},
	}, nil
}