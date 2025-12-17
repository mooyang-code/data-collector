package binance

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/internal/model/common"
	"github.com/mooyang-code/data-collector/internal/model/market"
	"trpc.group/trpc-go/trpc-go/log"
)

// KlineCollector K线数据采集器
type KlineCollector struct {
	*collector.BaseCollector
	symbol    string
	intervals []string
}

// init 自注册到采集器注册中心
func init() {
	err := collector.NewBuilder().
		Source("binance", "币安").
		DataType("kline", "K线").
		MarketType("spot", "现货").
		Description("币安现货K线数据采集器").
		Creator(NewKlineCollector).
		Register()

	if err != nil {
		panic(fmt.Sprintf("注册K线采集器失败: %v", err))
	}
}

// NewKlineCollector 创建K线采集器
func NewKlineCollector(config map[string]interface{}) (collector.Collector, error) {
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
	collectorID := fmt.Sprintf("binance_kline_%s", symbol)

	c := &KlineCollector{
		BaseCollector: collector.NewBaseCollector(collectorID, "market", "kline"),
		symbol:        symbol,
		intervals:     intervals,
	}

	return c, nil
}

// Initialize 初始化
func (c *KlineCollector) Initialize(ctx context.Context) error {
	log.InfoContextf(ctx, "K线采集器初始化: symbol=%s, intervals=%v", c.symbol, c.intervals)

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
		log.InfoContextf(ctx, "K线采集开始: symbol=%s, interval=%s", c.symbol, interval)

		// 生成假数据
		klines := c.generateMockKlines(interval, 10)

		log.InfoContextf(ctx, "K线采集完成: symbol=%s, interval=%s, count=%d",
			c.symbol, interval, len(klines))

		// TODO: 发布事件或存储数据
		return nil
	}
}

// generateMockKlines 生成假的K线数据
func (c *KlineCollector) generateMockKlines(interval string, count int) []*market.Kline {
	klines := make([]*market.Kline, 0, count)

	// 获取间隔时长
	duration, _ := market.IntervalDuration(interval)
	if duration == 0 {
		duration = time.Minute
	}

	// 基础价格（模拟 BTC 价格）
	basePrice := 50000.0 + rand.Float64()*10000

	now := time.Now().Truncate(duration)

	for i := 0; i < count; i++ {
		openTime := now.Add(-duration * time.Duration(count-i))
		closeTime := openTime.Add(duration)

		// 生成随机价格波动
		priceChange := (rand.Float64() - 0.5) * 200
		open := basePrice + priceChange
		close := open + (rand.Float64()-0.5)*100
		high := max(open, close) + rand.Float64()*50
		low := min(open, close) - rand.Float64()*50
		volume := 100 + rand.Float64()*1000

		kline := market.NewKline("binance", c.symbol, interval)
		kline.OpenTime = openTime
		kline.CloseTime = closeTime
		kline.Open = common.NewDecimalFromFloat(open)
		kline.High = common.NewDecimalFromFloat(high)
		kline.Low = common.NewDecimalFromFloat(low)
		kline.Close = common.NewDecimalFromFloat(close)
		kline.Volume = common.NewDecimalFromFloat(volume)
		kline.QuoteVolume = common.NewDecimalFromFloat(volume * open)
		kline.TradeCount = int64(rand.Intn(1000) + 100)

		klines = append(klines, kline)

		// 更新基础价格
		basePrice = close
	}

	return klines
}
