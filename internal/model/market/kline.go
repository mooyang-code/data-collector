package market

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/model/common"
)

// Kline K线数据
type Kline struct {
	common.BaseDataPoint
	Symbol      string         `json:"symbol"`
	Exchange    string         `json:"exchange"`
	Interval    string         `json:"interval"`
	OpenTime    time.Time      `json:"open_time"`
	CloseTime   time.Time      `json:"close_time"`
	Open        common.Decimal `json:"open"`
	High        common.Decimal `json:"high"`
	Low         common.Decimal `json:"low"`
	Close       common.Decimal `json:"close"`
	Volume      common.Decimal `json:"volume"`
	QuoteVolume common.Decimal `json:"quote_volume"`
	TradeCount  int64          `json:"trade_count"`
}

// NewKline 创建K线数据
func NewKline(exchange, symbol, interval string) *Kline {
	return &Kline{
		BaseDataPoint: common.NewBaseDataPoint(exchange, "kline"),
		Exchange:      exchange,
		Symbol:        symbol,
		Interval:      interval,
	}
}

// Validate 验证K线数据
func (k *Kline) Validate() error {
	if k.Symbol == "" {
		return fmt.Errorf("交易对不能为空")
	}
	if k.Exchange == "" {
		return fmt.Errorf("交易所不能为空")
	}
	if k.Interval == "" {
		return fmt.Errorf("时间间隔不能为空")
	}
	if k.OpenTime.IsZero() || k.CloseTime.IsZero() {
		return fmt.Errorf("时间不能为空")
	}
	if k.OpenTime.After(k.CloseTime) {
		return fmt.Errorf("开盘时间不能晚于收盘时间")
	}

	// 验证价格
	high, _ := k.High.Float64()
	low, _ := k.Low.Float64()
	if high < low {
		return fmt.Errorf("最高价不能低于最低价")
	}
	return nil
}

// Marshal 序列化
func (k *Kline) Marshal() ([]byte, error) {
	return json.Marshal(k)
}

// Unmarshal 反序列化
func (k *Kline) Unmarshal(data []byte) error {
	return json.Unmarshal(data, k)
}

// KlineBatch K线批量数据
type KlineBatch struct {
	Exchange string   `json:"exchange"`
	Symbol   string   `json:"symbol"`
	Interval string   `json:"interval"`
	Klines   []*Kline `json:"klines"`
	Count    int      `json:"count"`
}

// NewKlineBatch 创建K线批量数据
func NewKlineBatch(exchange, symbol, interval string) *KlineBatch {
	return &KlineBatch{
		Exchange: exchange,
		Symbol:   symbol,
		Interval: interval,
		Klines:   make([]*Kline, 0),
	}
}

// AddKline 添加K线
func (kb *KlineBatch) AddKline(kline *Kline) {
	kb.Klines = append(kb.Klines, kline)
	kb.Count = len(kb.Klines)
}

const (
	Interval1m  = "1m"  // Interval1m 1分钟时间间隔
	Interval5m  = "5m"  // Interval5m 5分钟时间间隔
	Interval15m = "15m" // Interval15m 15分钟时间间隔
	Interval30m = "30m" // Interval30m 30分钟时间间隔
	Interval1h  = "1h"  // Interval1h 1小时时间间隔
	Interval4h  = "4h"  // Interval4h 4小时时间间隔
	Interval1d  = "1d"  // Interval1d 1天时间间隔
	Interval1w  = "1w"  // Interval1w 1周时间间隔
)

// IntervalDuration 获取间隔对应的时间长度
func IntervalDuration(interval string) (time.Duration, error) {
	switch interval {
	case Interval1m:
		return 1 * time.Minute, nil
	case Interval5m:
		return 5 * time.Minute, nil
	case Interval15m:
		return 15 * time.Minute, nil
	case Interval30m:
		return 30 * time.Minute, nil
	case Interval1h:
		return 1 * time.Hour, nil
	case Interval4h:
		return 4 * time.Hour, nil
	case Interval1d:
		return 24 * time.Hour, nil
	case Interval1w:
		return 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("未知的时间间隔: %s", interval)
	}
}
