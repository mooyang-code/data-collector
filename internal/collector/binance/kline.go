package binance

import (
	"context"
	"fmt"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/internal/exchange"
	binanceapi "github.com/mooyang-code/data-collector/internal/exchange/binance"
	"github.com/mooyang-code/data-collector/internal/model/market"
	"trpc.group/trpc-go/trpc-go/log"
)

// 产品类型常量
const (
	InstTypeSPOT = "SPOT" // 现货
	InstTypeSWAP = "SWAP" // 永续合约
)

// KlineCollector K线数据采集器
type KlineCollector struct {
	client  *binanceapi.Client
	spotAPI *binanceapi.SpotAPI
	swapAPI *binanceapi.SwapAPI
}

// init 自注册到采集器注册中心
func init() {
	// 创建采集器实例
	client := binanceapi.NewClient()
	c := &KlineCollector{
		client:  client,
		spotAPI: binanceapi.NewSpotAPI(client),
		swapAPI: binanceapi.NewSwapAPI(client),
	}

	// 注册到全局注册中心
	err := collector.NewBuilder().
		Source("binance", "币安").
		DataType("kline", "K线").
		Description("币安K线数据采集器").
		Collector(c).
		Register()

	if err != nil {
		panic(fmt.Sprintf("注册K线采集器失败: %v", err))
	}
}

// Source 返回数据源标识
func (c *KlineCollector) Source() string {
	return "binance"
}

// DataType 返回数据类型标识
func (c *KlineCollector) DataType() string {
	return "kline"
}

// Collect 执行一次K线采集
func (c *KlineCollector) Collect(ctx context.Context, params *collector.CollectParams) error {
	log.InfoContextf(ctx, "K线采集开始: inst_type=%s, symbol=%s, interval=%s",
		params.InstType, params.Symbol, params.Interval)

	// 从币安 API 获取 K 线数据
	klines, err := c.fetchKlines(ctx, params)
	if err != nil {
		log.ErrorContextf(ctx, "K线采集失败: inst_type=%s, symbol=%s, interval=%s, error=%v",
			params.InstType, params.Symbol, params.Interval, err)
		return err
	}

	if len(klines) > 0 {
		log.InfoContextf(ctx, "K线采集完成: inst_type=%s, symbol=%s, interval=%s, count=%d, latest=%+v",
			params.InstType, params.Symbol, params.Interval, len(klines), klines[0])
	}

	// TODO: 发布事件或存储数据
	return nil
}

// fetchKlines 从币安 API 获取 K 线数据
func (c *KlineCollector) fetchKlines(ctx context.Context, params *collector.CollectParams) ([]*market.Kline, error) {
	req := &exchange.KlineRequest{
		Symbol:   params.Symbol,
		Interval: params.Interval,
		Limit:    5, // 只获取最新的5根K线
	}

	var exchangeKlines []*exchange.Kline
	var err error

	// 根据产品类型选择 API
	switch params.InstType {
	case InstTypeSPOT:
		exchangeKlines, err = c.spotAPI.GetKline(ctx, req)
	case InstTypeSWAP:
		exchangeKlines, err = c.swapAPI.GetKline(ctx, req)
	default:
		return nil, fmt.Errorf("不支持的产品类型: %s", params.InstType)
	}

	if err != nil {
		return nil, err
	}

	// 转换为 market.Kline 格式
	klines := make([]*market.Kline, 0, len(exchangeKlines))
	for _, ek := range exchangeKlines {
		kline := market.NewKline("binance", params.Symbol, params.Interval)
		kline.OpenTime = ek.OpenTime
		kline.CloseTime = ek.CloseTime
		kline.Open = ek.Open
		kline.High = ek.High
		kline.Low = ek.Low
		kline.Close = ek.Close
		kline.Volume = ek.Volume
		kline.QuoteVolume = ek.QuoteVolume
		kline.TradeCount = ek.TradeCount
		klines = append(klines, kline)
	}
	return klines, nil
}
