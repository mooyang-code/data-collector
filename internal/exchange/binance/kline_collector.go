// Package binance 币安K线采集器实现
package binance

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector/kline"
	"github.com/mooyang-code/data-collector/internal/datatype/klines"
	"trpc.group/trpc-go/trpc-go/log"
)

// KlineCollector 币安K线采集器
type KlineCollector struct {
	*kline.BaseKlineCollector
	client *APIClient
}

// NewKlineCollector 创建币安K线采集器
func NewKlineCollector(config kline.Config) (*KlineCollector, error) {
	// 创建API客户端
	client := NewAPIClient(&ClientConfig{
		BaseURL: "https://api.binance.com",
		Timeout: 30 * time.Second,
	})
	
	// 创建数据存储（这里使用内存存储作为示例）
	store := NewMemoryKlineStore()
	
	// 创建币安API适配器
	api := &binanceAPI{client: client}
	
	// 创建采集器ID
	id := fmt.Sprintf("binance_kline_%d", time.Now().Unix())
	
	// 创建基础采集器
	base := kline.NewBaseKlineCollector(id, "binance", config, api, store)
	
	return &KlineCollector{
		BaseKlineCollector: base,
		client:             client,
	}, nil
}

// binanceAPI 币安API适配器
type binanceAPI struct {
	client *APIClient
}

// GetKlines 获取K线数据
func (api *binanceAPI) GetKlines(ctx context.Context, symbol string, interval klines.Interval, start, end time.Time) ([]*klines.Kline, error) {
	// 构建请求参数
	params := map[string]interface{}{
		"symbol":    symbol,
		"interval":  api.intervalToString(interval),
		"startTime": start.UnixMilli(),
		"endTime":   end.UnixMilli(),
		"limit":     1000,
	}
	
	// 调用API
	var response [][]interface{}
	err := api.client.Get(ctx, "/api/v3/klines", params, &response)
	if err != nil {
		return nil, fmt.Errorf("API调用失败: %w", err)
	}
	
	// 转换数据
	result := make([]*klines.Kline, 0, len(response))
	for _, item := range response {
		if len(item) < 11 {
			continue
		}
		
		kline := &klines.Kline{
			Symbol:      symbol,
			Interval:    interval,
			OpenTime:    time.UnixMilli(int64(item[0].(float64))),
			Open:        item[1].(string),
			High:        item[2].(string),
			Low:         item[3].(string),
			Close:       item[4].(string),
			Volume:      item[5].(string),
			CloseTime:   time.UnixMilli(int64(item[6].(float64))),
			QuoteVolume: item[7].(string),
			Trades:      int64(item[8].(float64)),
		}
		result = append(result, kline)
	}
	
	return result, nil
}

// GetExchangeTime 获取交易所时间
func (api *binanceAPI) GetExchangeTime(ctx context.Context) (time.Time, error) {
	var response struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err := api.client.Get(ctx, "/api/v3/time", nil, &response)
	if err != nil {
		return time.Time{}, err
	}
	
	return time.UnixMilli(response.ServerTime), nil
}

// intervalToString 将K线周期转换为字符串
func (api *binanceAPI) intervalToString(interval klines.Interval) string {
	switch interval {
	case klines.Interval1m:
		return "1m"
	case klines.Interval5m:
		return "5m"
	case klines.Interval15m:
		return "15m"
	case klines.Interval30m:
		return "30m"
	case klines.Interval1h:
		return "1h"
	case klines.Interval4h:
		return "4h"
	case klines.Interval1d:
		return "1d"
	default:
		return "1h"
	}
}

// MemoryKlineStore 内存K线存储（示例实现）
type MemoryKlineStore struct {
	data map[string][]*klines.Kline
}

// NewMemoryKlineStore 创建内存存储
func NewMemoryKlineStore() *MemoryKlineStore {
	return &MemoryKlineStore{
		data: make(map[string][]*klines.Kline),
	}
}

// SaveKlines 保存K线数据
func (s *MemoryKlineStore) SaveKlines(symbol string, interval klines.Interval, data []*klines.Kline) error {
	key := fmt.Sprintf("%s_%s", symbol, interval)
	
	// 简单追加（实际应该去重和排序）
	s.data[key] = append(s.data[key], data...)
	
	log.Debugf("保存K线到内存: %s, 数量: %d", key, len(data))
	return nil
}

// GetLastKline 获取最后一条K线
func (s *MemoryKlineStore) GetLastKline(symbol string, interval klines.Interval) (*klines.Kline, error) {
	key := fmt.Sprintf("%s_%s", symbol, interval)
	
	klines := s.data[key]
	if len(klines) == 0 {
		return nil, fmt.Errorf("not found")
	}
	
	return klines[len(klines)-1], nil
}