package exchanges

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// OKXCollector OKX交易所数据采集器
type OKXCollector struct {
	*BaseCollector
	apiKey     string
	apiSecret  string
	passphrase string
	baseURL    string
}

// NewOKXCollector 创建OKX采集器
func NewOKXCollector() collector.Collector {
	return &OKXCollector{
		BaseCollector: NewBaseCollector("okx", model.CollectorTypeOKX),
		baseURL:       "https://www.okx.com",
	}
}

func (c *OKXCollector) Collect(ctx context.Context, taskType model.TaskType, params *model.CollectParams) (*model.CollectResult, error) {
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

func (c *OKXCollector) collectKLines(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟OKX K线数据采集
	now := time.Now()
	
	// OKX的K线数据格式
	klines := [][]string{
		{
			fmt.Sprintf("%d", now.Add(-time.Minute).UnixMilli()), // ts
			"50000.0",  // o (open)
			"50100.0",  // h (high)
			"49900.0",  // l (low)
			"50050.0",  // c (close)
			"1.23456",  // vol
			"61728.95", // volCcy
			"0",        // volCcyQuote
			"1",        // confirm
		},
	}
	
	return &model.CollectResult{
		Data:      klines,
		Count:     len(klines),
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "okx",
			"type":     "klines",
			"symbol":   params.Symbol,
			"interval": params.Interval,
		},
	}, nil
}

func (c *OKXCollector) collectTicker(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟OKX ticker数据采集
	now := time.Now()
	
	ticker := map[string]interface{}{
		"instType": "SPOT",
		"instId":   params.Symbol,
		"last":     "50050.0",
		"lastSz":   "0.10000000",
		"askPx":    "50051.0",
		"askSz":    "1.5",
		"bidPx":    "50049.0",
		"bidSz":    "2.3",
		"open24h":  "49900.0",
		"high24h":  "50100.0",
		"low24h":   "49800.0",
		"volCcy24h": "617283950.0",
		"vol24h":   "12345.67890",
		"ts":       fmt.Sprintf("%d", now.UnixMilli()),
		"sodUtc0":  "49900.0",
		"sodUtc8":  "49900.0",
	}
	
	return &model.CollectResult{
		Data:      ticker,
		Count:     1,
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "okx",
			"type":     "ticker",
			"symbol":   params.Symbol,
		},
	}, nil
}

func (c *OKXCollector) collectOrderBook(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟OKX订单簿数据采集
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
			fmt.Sprintf("%.1f", bidPrice),
			fmt.Sprintf("%.8f", 1.0+float64(i)*0.1),
			"0", // deprecated field
			fmt.Sprintf("%d", i+1), // number of orders
		}
		asks[i] = []string{
			fmt.Sprintf("%.1f", askPrice),
			fmt.Sprintf("%.8f", 1.0+float64(i)*0.1),
			"0", // deprecated field
			fmt.Sprintf("%d", i+1), // number of orders
		}
	}
	
	orderBook := map[string]interface{}{
		"asks": asks,
		"bids": bids,
		"ts":   fmt.Sprintf("%d", now.UnixMilli()),
	}
	
	return &model.CollectResult{
		Data:      orderBook,
		Count:     1,
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "okx",
			"type":     "orderbook",
			"symbol":   params.Symbol,
			"limit":    limit,
		},
	}, nil
}

func (c *OKXCollector) collectTrades(ctx context.Context, params *model.CollectParams) (*model.CollectResult, error) {
	// 模拟OKX交易数据采集
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
		side := "buy"
		if i%2 == 0 {
			side = "sell"
		}
		
		trades[i] = map[string]interface{}{
			"instId":  params.Symbol,
			"tradeId": fmt.Sprintf("%d", now.UnixNano()+int64(i)),
			"px":      fmt.Sprintf("%.1f", 50000.0+float64(i-limit/2)*0.1),
			"sz":      fmt.Sprintf("%.8f", 0.1+float64(i)*0.001),
			"side":    side,
			"ts":      fmt.Sprintf("%d", now.Add(-time.Duration(limit-i)*time.Second).UnixMilli()),
		}
	}
	
	return &model.CollectResult{
		Data:      trades,
		Count:     len(trades),
		Timestamp: now,
		Metadata: map[string]interface{}{
			"exchange": "okx",
			"type":     "trades",
			"symbol":   params.Symbol,
			"limit":    limit,
		},
	}, nil
}