// Package klines 币安K线适配器实现
package klines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/klines"
	"trpc.group/trpc-go/trpc-go/log"
)

// BinanceKlineAdapter 币安K线适配器
type BinanceKlineAdapter struct {
	baseURL    string
	httpClient *http.Client
}

// NewBinanceKlineAdapter 创建币安K线适配器
func NewBinanceKlineAdapter(baseURL string) *BinanceKlineAdapter {
	if baseURL == "" {
		baseURL = "https://api.binance.com"
	}
	
	return &BinanceKlineAdapter{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetExchange 获取交易所名称
func (a *BinanceKlineAdapter) GetExchange() string {
	return "binance"
}

// SubscribeKlines 订阅实时K线（币安使用WebSocket，这里暂时返回不支持）
func (a *BinanceKlineAdapter) SubscribeKlines(ctx context.Context, subscriptions []*klines.KlineSubscription, eventCh chan<- *klines.KlineEvent) error {
	// 币安的实时K线需要WebSocket实现，这里暂时返回不支持
	// 实际项目中可以在这里实现WebSocket连接逻辑
	return fmt.Errorf("real-time subscription not implemented yet")
}

// UnsubscribeKlines 取消订阅
func (a *BinanceKlineAdapter) UnsubscribeKlines(ctx context.Context, subscriptions []*klines.KlineSubscription) error {
	// WebSocket取消订阅逻辑
	return fmt.Errorf("real-time unsubscription not implemented yet")
}

// FetchHistoryKlines 获取历史K线
func (a *BinanceKlineAdapter) FetchHistoryKlines(ctx context.Context, symbol string, interval klines.Interval, startTime, endTime time.Time, limit int) ([]*klines.Kline, error) {
	// 构建请求URL - 根据baseURL判断是现货还是合约
	var url string
	if a.baseURL == "https://fapi.binance.com" {
		url = fmt.Sprintf("%s/fapi/v1/klines", a.baseURL) // 合约API
	} else {
		url = fmt.Sprintf("%s/api/v3/klines", a.baseURL)  // 现货API
	}
	
	// 构建请求参数
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	q := req.URL.Query()
	q.Set("symbol", symbol)
	q.Set("interval", string(interval))
	
	if !startTime.IsZero() {
		q.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		q.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	
	req.URL.RawQuery = q.Encode()
	
	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var rawKlines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawKlines); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	
	// 转换为标准格式
	result := make([]*klines.Kline, 0, len(rawKlines))
	for _, raw := range rawKlines {
		if len(raw) < 11 {
			continue
		}
		
		kline, err := a.parseRawKline(raw, symbol, interval)
		if err != nil {
			log.Warnf("解析K线数据失败: %v", err)
			continue
		}
		
		result = append(result, kline)
	}
	
	return result, nil
}

// parseRawKline 解析原始K线数据
func (a *BinanceKlineAdapter) parseRawKline(raw []interface{}, symbol string, interval klines.Interval) (*klines.Kline, error) {
	// 币安K线数据格式：
	// [
	//   1499040000000,      // 开盘时间
	//   "0.01634790",       // 开盘价
	//   "0.80000000",       // 最高价
	//   "0.01575800",       // 最低价
	//   "0.01577100",       // 收盘价
	//   "148976.11427815",  // 成交量
	//   1499644799999,      // 收盘时间
	//   "2434.19055334",    // 成交额
	//   308,                // 成交笔数
	//   "1756.87402397",    // 主动买入成交量
	//   "28.46694368",      // 主动买入成交额
	//   "17928899.62484339" // 忽略
	// ]
	
	openTimeMs, ok := raw[0].(float64)
	if !ok {
		return nil, fmt.Errorf("无效的开盘时间")
	}
	
	closeTimeMs, ok := raw[6].(float64)
	if !ok {
		return nil, fmt.Errorf("无效的收盘时间")
	}
	
	open, ok := raw[1].(string)
	if !ok {
		return nil, fmt.Errorf("无效的开盘价")
	}
	
	high, ok := raw[2].(string)
	if !ok {
		return nil, fmt.Errorf("无效的最高价")
	}
	
	low, ok := raw[3].(string)
	if !ok {
		return nil, fmt.Errorf("无效的最低价")
	}
	
	close, ok := raw[4].(string)
	if !ok {
		return nil, fmt.Errorf("无效的收盘价")
	}
	
	volume, ok := raw[5].(string)
	if !ok {
		return nil, fmt.Errorf("无效的成交量")
	}
	
	quoteVolume, ok := raw[7].(string)
	if !ok {
		return nil, fmt.Errorf("无效的成交额")
	}
	
	tradeCount, ok := raw[8].(float64)
	if !ok {
		return nil, fmt.Errorf("无效的成交笔数")
	}
	
	return &klines.Kline{
		Exchange:    a.GetExchange(),
		Symbol:      symbol,
		Interval:    interval,
		OpenTime:    time.UnixMilli(int64(openTimeMs)),
		CloseTime:   time.UnixMilli(int64(closeTimeMs)),
		Open:        open,
		High:        high,
		Low:         low,
		Close:       close,
		Volume:      volume,
		QuoteVolume: quoteVolume,
		TradeCount:  int64(tradeCount),
		IsClosed:    true, // 历史数据都是已闭合的
		Source:      "binance_api",
		Timestamp:   time.Now(),
	}, nil
}

// GetSupportedIntervals 获取支持的周期
func (a *BinanceKlineAdapter) GetSupportedIntervals() []klines.Interval {
	return []klines.Interval{
		klines.Interval1m,
		klines.Interval3m,
		klines.Interval5m,
		klines.Interval15m,
		klines.Interval30m,
		klines.Interval1h,
		klines.Interval2h,
		klines.Interval4h,
		klines.Interval6h,
		klines.Interval8h,
		klines.Interval12h,
		klines.Interval1d,
		klines.Interval3d,
		klines.Interval1w,
		klines.Interval1M,
	}
}

// IsSymbolSupported 检查是否支持该交易对
func (a *BinanceKlineAdapter) IsSymbolSupported(symbol string) bool {
	// 这里可以实现更复杂的逻辑，比如从API获取支持的交易对列表
	// 暂时简单返回true，表示支持所有交易对
	return symbol != ""
}
