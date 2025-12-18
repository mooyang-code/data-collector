package binance

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mooyang-code/data-collector/internal/exchange"
	"github.com/mooyang-code/data-collector/internal/model/common"
)

// CandleStick 币安K线原始数据
// 币安返回的是数组格式：[openTime, open, high, low, close, volume, closeTime, quoteVolume, tradeCount, takerBuyVolume, takerBuyQuoteVolume, ignore]
type CandleStick struct {
	OpenTime    int64  // 开盘时间（毫秒）
	Open        string // 开盘价
	High        string // 最高价
	Low         string // 最低价
	Close       string // 收盘价
	Volume      string // 成交量
	CloseTime   int64  // 收盘时间（毫秒）
	QuoteVolume string // 成交额
	TradeCount  int64  // 成交笔数
}

// UnmarshalJSON 自定义 JSON 解析（处理数组格式）
func (c *CandleStick) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析K线数组失败: %w", err)
	}

	if len(raw) < 9 {
		return fmt.Errorf("K线数据字段不足，期望至少9个，实际%d个", len(raw))
	}

	// 解析各字段
	if err := json.Unmarshal(raw[0], &c.OpenTime); err != nil {
		return fmt.Errorf("解析 openTime 失败: %w", err)
	}
	if err := json.Unmarshal(raw[1], &c.Open); err != nil {
		return fmt.Errorf("解析 open 失败: %w", err)
	}
	if err := json.Unmarshal(raw[2], &c.High); err != nil {
		return fmt.Errorf("解析 high 失败: %w", err)
	}
	if err := json.Unmarshal(raw[3], &c.Low); err != nil {
		return fmt.Errorf("解析 low 失败: %w", err)
	}
	if err := json.Unmarshal(raw[4], &c.Close); err != nil {
		return fmt.Errorf("解析 close 失败: %w", err)
	}
	if err := json.Unmarshal(raw[5], &c.Volume); err != nil {
		return fmt.Errorf("解析 volume 失败: %w", err)
	}
	if err := json.Unmarshal(raw[6], &c.CloseTime); err != nil {
		return fmt.Errorf("解析 closeTime 失败: %w", err)
	}
	if err := json.Unmarshal(raw[7], &c.QuoteVolume); err != nil {
		return fmt.Errorf("解析 quoteVolume 失败: %w", err)
	}

	// tradeCount 可能是数字或字符串
	var tradeCount interface{}
	if err := json.Unmarshal(raw[8], &tradeCount); err != nil {
		return fmt.Errorf("解析 tradeCount 失败: %w", err)
	}
	switch v := tradeCount.(type) {
	case float64:
		c.TradeCount = int64(v)
	case string:
		tc, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("转换 tradeCount 失败: %w", err)
		}
		c.TradeCount = tc
	}

	return nil
}

// ToKline 转换为通用 Kline 结构
func (c *CandleStick) ToKline() (*exchange.Kline, error) {
	return &exchange.Kline{
		OpenTime:    time.UnixMilli(c.OpenTime),
		CloseTime:   time.UnixMilli(c.CloseTime),
		Open:        common.NewDecimal(c.Open),
		High:        common.NewDecimal(c.High),
		Low:         common.NewDecimal(c.Low),
		Close:       common.NewDecimal(c.Close),
		Volume:      common.NewDecimal(c.Volume),
		QuoteVolume: common.NewDecimal(c.QuoteVolume),
		TradeCount:  c.TradeCount,
	}, nil
}

// APIError 币安 API 错误响应
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("币安API错误 [%d]: %s", e.Code, e.Message)
}
