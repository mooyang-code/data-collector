// Package klines 提供K线数据管理的核心功能
package klines

import (
	"fmt"
	"strconv"
	"time"
)

// Interval K线周期
type Interval string

const (
	Interval1m  Interval = "1m"
	Interval3m  Interval = "3m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval30m Interval = "30m"
	Interval1h  Interval = "1h"
	Interval2h  Interval = "2h"
	Interval4h  Interval = "4h"
	Interval6h  Interval = "6h"
	Interval8h  Interval = "8h"
	Interval12h Interval = "12h"
	Interval1d  Interval = "1d"
	Interval3d  Interval = "3d"
	Interval1w  Interval = "1w"
	Interval1M  Interval = "1M"
)

// ToMinutes 将周期转换为分钟数
func (i Interval) ToMinutes() int {
	switch i {
	case Interval1m:
		return 1
	case Interval3m:
		return 3
	case Interval5m:
		return 5
	case Interval15m:
		return 15
	case Interval30m:
		return 30
	case Interval1h:
		return 60
	case Interval2h:
		return 120
	case Interval4h:
		return 240
	case Interval6h:
		return 360
	case Interval8h:
		return 480
	case Interval12h:
		return 720
	case Interval1d:
		return 1440
	case Interval3d:
		return 4320
	case Interval1w:
		return 10080
	case Interval1M:
		return 43200 // 30天
	default:
		return 0
	}
}

// ToDuration 将周期转换为时间间隔
func (i Interval) ToDuration() time.Duration {
	return time.Duration(i.ToMinutes()) * time.Minute
}

// IsValid 检查周期是否有效
func (i Interval) IsValid() bool {
	return i.ToMinutes() > 0
}

// Kline K线数据
type Kline struct {
	// 基础信息
	Exchange string   `json:"exchange"` // 交易所
	Symbol   string   `json:"symbol"`   // 交易对
	Interval Interval `json:"interval"` // 周期

	// 时间信息
	OpenTime  time.Time `json:"openTime"`  // 开盘时间
	CloseTime time.Time `json:"closeTime"` // 收盘时间

	// OHLCV数据
	Open   string `json:"open"`   // 开盘价
	High   string `json:"high"`   // 最高价
	Low    string `json:"low"`    // 最低价
	Close  string `json:"close"`  // 收盘价
	Volume string `json:"volume"` // 成交量

	// 扩展数据
	QuoteVolume string `json:"quoteVolume"` // 成交额
	TradeCount  int64  `json:"tradeCount"`  // 成交笔数

	// 状态信息
	IsClosed bool `json:"isClosed"` // 是否已闭合

	// 元数据
	Source    string                 `json:"source"`    // 数据源
	Timestamp time.Time              `json:"timestamp"` // 数据时间戳
	Extra     map[string]interface{} `json:"extra"`     // 扩展字段
}

// Key 返回K线的唯一标识
func (k *Kline) Key() string {
	return fmt.Sprintf("%s:%s:%s:%d", k.Exchange, k.Symbol, k.Interval, k.OpenTime.Unix())
}

// ToKlineRecord 转换为新的KlineRecord格式
func (k *Kline) ToKlineRecord() *KlineRecord {
	return &KlineRecord{
		Exchange:    k.Exchange,
		Symbol:      k.Symbol,
		Interval:    string(k.Interval),
		OpenTimeMs:  k.OpenTime.UnixMilli(),
		CloseTimeMs: k.CloseTime.UnixMilli(),
		Open:        k.Open,
		High:        k.High,
		Low:         k.Low,
		Close:       k.Close,
		Volume:      k.Volume,
		Closed:      k.IsClosed,
	}
}

// GetOpenFloat 获取开盘价浮点数
func (k *Kline) GetOpenFloat() (float64, error) {
	return strconv.ParseFloat(k.Open, 64)
}

// GetHighFloat 获取最高价浮点数
func (k *Kline) GetHighFloat() (float64, error) {
	return strconv.ParseFloat(k.High, 64)
}

// GetLowFloat 获取最低价浮点数
func (k *Kline) GetLowFloat() (float64, error) {
	return strconv.ParseFloat(k.Low, 64)
}

// GetCloseFloat 获取收盘价浮点数
func (k *Kline) GetCloseFloat() (float64, error) {
	return strconv.ParseFloat(k.Close, 64)
}

// GetVolumeFloat 获取成交量浮点数
func (k *Kline) GetVolumeFloat() (float64, error) {
	return strconv.ParseFloat(k.Volume, 64)
}

// GetQuoteVolumeFloat 获取成交额浮点数
func (k *Kline) GetQuoteVolumeFloat() (float64, error) {
	return strconv.ParseFloat(k.QuoteVolume, 64)
}

// Clone 克隆K线数据
func (k *Kline) Clone() *Kline {
	clone := *k
	if k.Extra != nil {
		clone.Extra = make(map[string]interface{})
		for key, value := range k.Extra {
			clone.Extra[key] = value
		}
	}
	return &clone
}

// KlineSubscription K线订阅（保持向后兼容）
type KlineSubscription struct {
	Exchange string   `json:"exchange"` // 交易所
	Symbol   string   `json:"symbol"`   // 交易对
	Interval Interval `json:"interval"` // 周期
}

// Key 返回订阅的唯一标识
func (s *KlineSubscription) Key() string {
	return fmt.Sprintf("%s:%s:%s", s.Exchange, s.Symbol, s.Interval)
}

// ToSymbolInterval 转换为新的SymbolInterval格式
func (s *KlineSubscription) ToSymbolInterval() SymbolInterval {
	return SymbolInterval{
		Symbol:   s.Symbol,
		Interval: string(s.Interval),
	}
}

// FromSymbolInterval 从SymbolInterval创建KlineSubscription
func FromSymbolInterval(si SymbolInterval, exchange string) *KlineSubscription {
	return &KlineSubscription{
		Exchange: exchange,
		Symbol:   si.Symbol,
		Interval: Interval(si.Interval),
	}
}

// KlineFilter K线过滤器（保持向后兼容）
type KlineFilter struct {
	Exchange   string    `json:"exchange,omitempty"`  // 交易所过滤
	Symbol     string    `json:"symbol,omitempty"`    // 交易对过滤
	Interval   Interval  `json:"interval,omitempty"`  // 周期过滤
	StartTime  time.Time `json:"startTime,omitempty"` // 开始时间
	EndTime    time.Time `json:"endTime,omitempty"`   // 结束时间
	Limit      int       `json:"limit,omitempty"`     // 数量限制
	ClosedOnly bool      `json:"closedOnly"`          // 仅闭合K线
}

// ToKlineQuery 转换为新的KlineQuery格式
func (f *KlineFilter) ToKlineQuery() KlineQuery {
	query := KlineQuery{
		Symbol:   f.Symbol,
		Interval: string(f.Interval),
		Limit:    f.Limit,
	}

	if !f.StartTime.IsZero() {
		query.StartMs = f.StartTime.UnixMilli()
	}

	if !f.EndTime.IsZero() {
		query.EndMs = f.EndTime.UnixMilli()
	}

	return query
}

// Match 检查K线是否匹配过滤条件
func (f *KlineFilter) Match(kline *Kline) bool {
	if f.Exchange != "" && f.Exchange != kline.Exchange {
		return false
	}

	if f.Symbol != "" && f.Symbol != kline.Symbol {
		return false
	}

	if f.Interval != "" && f.Interval != kline.Interval {
		return false
	}

	if !f.StartTime.IsZero() && kline.OpenTime.Before(f.StartTime) {
		return false
	}

	if !f.EndTime.IsZero() && kline.OpenTime.After(f.EndTime) {
		return false
	}

	if f.ClosedOnly && !kline.IsClosed {
		return false
	}

	return true
}
