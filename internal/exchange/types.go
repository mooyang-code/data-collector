package exchange

import (
	"time"

	"github.com/mooyang-code/data-collector/internal/model/common"
)

// KlineRequest K线请求参数
type KlineRequest struct {
	Symbol    string    // 交易对，如 BTC-USDT
	Interval  string    // K线周期，如 1m, 5m, 1h
	Limit     int       // 返回数量限制
	StartTime time.Time // 开始时间（可选）
	EndTime   time.Time // 结束时间（可选）
}

// Kline K线数据
type Kline struct {
	OpenTime    time.Time      // 开盘时间
	CloseTime   time.Time      // 收盘时间
	Open        common.Decimal // 开盘价
	High        common.Decimal // 最高价
	Low         common.Decimal // 最低价
	Close       common.Decimal // 收盘价
	Volume      common.Decimal // 成交量
	QuoteVolume common.Decimal // 成交额
	TradeCount  int64          // 成交笔数
}
