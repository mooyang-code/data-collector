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
	*collector.BaseCollector
	instType  string   // 产品类型
	symbol    string   // 交易对
	intervals []string // K线周期列表

	// API 客户端
	binanceClient *binanceapi.Client
	spotAPI       *binanceapi.SpotAPI
	swapAPI       *binanceapi.SwapAPI
}

// init 自注册到采集器注册中心
func init() {
	err := collector.NewBuilder().
		Source("binance", "币安").
		DataType("kline", "K线").
		Description("币安K线数据采集器").
		Creator(NewKlineCollector).
		Register()

	if err != nil {
		panic(fmt.Sprintf("注册K线采集器失败: %v", err))
	}
}

// NewKlineCollector 创建K线采集器
func NewKlineCollector(config map[string]interface{}) (collector.Collector, error) {
	// 解析 inst_type（必填）
	instType, ok := config["inst_type"].(string)
	if !ok || instType == "" {
		return nil, fmt.Errorf("缺少必填参数 inst_type")
	}

	// 验证 inst_type
	if instType != InstTypeSPOT && instType != InstTypeSWAP {
		return nil, fmt.Errorf("无效的产品类型 inst_type: %s，支持: SPOT, SWAP", instType)
	}

	// 解析 symbol（必填）
	symbol, ok := config["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("缺少必填参数 symbol")
	}

	// 解析 intervals（必填）
	intervals, ok := config["intervals"].([]string)
	if !ok || len(intervals) == 0 {
		return nil, fmt.Errorf("缺少必填参数 intervals")
	}

	// 生成唯一的采集器ID
	collectorID := fmt.Sprintf("binance_kline_%s_%s", instType, symbol)

	// 创建币安客户端
	binanceClient := binanceapi.NewClient()

	c := &KlineCollector{
		BaseCollector: collector.NewBaseCollector(collectorID, "market", "kline"),
		instType:      instType,
		symbol:        symbol,
		intervals:     intervals,
		binanceClient: binanceClient,
		spotAPI:       binanceapi.NewSpotAPI(binanceClient),
		swapAPI:       binanceapi.NewSwapAPI(binanceClient),
	}

	return c, nil
}

// Initialize 初始化
func (c *KlineCollector) Initialize(ctx context.Context) error {
	log.InfoContextf(ctx, "K线采集器初始化: inst_type=%s, symbol=%s, intervals=%v", c.instType, c.symbol, c.intervals)

	// 调用基类初始化
	if err := c.BaseCollector.Initialize(ctx); err != nil {
		return err
	}

	// 为每个时间间隔添加定时器
	for _, interval := range c.intervals {
		duration, err := market.IntervalDuration(interval)
		if err != nil {
			return fmt.Errorf("无效的时间间隔 %s: %w", interval, err)
		}

		// 创建定时器
		timerName := fmt.Sprintf("collect_%s", interval)
		handler := c.createCollectHandler(interval)

		if err := c.AddTimer(ctx, timerName, duration, handler); err != nil {
			return fmt.Errorf("添加定时器失败 %s: %w", timerName, err)
		}

		log.InfoContextf(ctx, "K线采集器: 添加定时器 %s, 间隔 %v", timerName, duration)
	}

	log.InfoContextf(ctx, "K线采集器初始化完成")
	return nil
}

// createCollectHandler 创建采集处理函数
func (c *KlineCollector) createCollectHandler(interval string) collector.TimerHandler {
	return func(ctx context.Context) error {
		log.InfoContextf(ctx, "K线采集开始: inst_type=%s, symbol=%s, interval=%s", c.instType, c.symbol, interval)

		// 从币安 API 获取 K 线数据
		klines, err := c.fetchKlines(ctx, interval)
		if err != nil {
			log.ErrorContextf(ctx, "K线采集失败: inst_type=%s, symbol=%s, interval=%s, error=%v",
				c.instType, c.symbol, interval, err)
			return err
		}

		log.InfoContextf(ctx, "K线采集完成: inst_type=%s, symbol=%s, interval=%s, count=%d",
			c.instType, c.symbol, interval, len(klines))

		// TODO: 发布事件或存储数据
		return nil
	}
}

// fetchKlines 从币安 API 获取 K 线数据
func (c *KlineCollector) fetchKlines(ctx context.Context, interval string) ([]*market.Kline, error) {
	req := &exchange.KlineRequest{
		Symbol:   c.symbol,
		Interval: interval,
		Limit:    1, // 只获取最新的一根K线
	}

	var exchangeKlines []*exchange.Kline
	var err error

	// 根据产品类型选择 API
	switch c.instType {
	case InstTypeSPOT:
		exchangeKlines, err = c.spotAPI.GetKline(ctx, req)
	case InstTypeSWAP:
		exchangeKlines, err = c.swapAPI.GetKline(ctx, req)
	default:
		return nil, fmt.Errorf("不支持的产品类型: %s", c.instType)
	}

	if err != nil {
		return nil, err
	}

	// 转换为 market.Kline 格式
	klines := make([]*market.Kline, 0, len(exchangeKlines))
	for _, ek := range exchangeKlines {
		kline := market.NewKline("binance", c.symbol, interval)
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
