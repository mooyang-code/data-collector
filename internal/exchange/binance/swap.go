package binance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/avast/retry-go"
	"github.com/mooyang-code/data-collector/internal/exchange"
	"trpc.group/trpc-go/trpc-go/log"
)

// SwapAPI 永续合约 API
type SwapAPI struct {
	client *Client
}

// NewSwapAPI 创建永续合约 API
func NewSwapAPI(client *Client) *SwapAPI {
	return &SwapAPI{client: client}
}

// GetKline 获取永续合约K线数据
// API: GET https://fapi.binance.com/fapi/v1/klines
func (api *SwapAPI) GetKline(ctx context.Context, req *exchange.KlineRequest) ([]*exchange.Kline, error) {
	params := url.Values{}

	// 转换交易对格式
	symbol := FormatSymbol(req.Symbol)
	params.Set("symbol", symbol)
	params.Set("interval", req.Interval)

	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}

	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}

	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}

	// 发送请求（带重试）
	var rawKlines []CandleStick
	err := retry.Do(
		func() error {
			return api.client.Get(ctx, SwapDomain, SwapKlineEndpoint, params, &rawKlines)
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			log.WarnContextf(ctx, "[SwapAPI] 获取K线重试 #%d, symbol=%s, interval=%s, err=%v", n+1, symbol, req.Interval, err)
		}),
		retry.Context(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("获取永续合约K线失败: %w", err)
	}

	// 转换为通用格式
	klines := make([]*exchange.Kline, 0, len(rawKlines))
	for _, raw := range rawKlines {
		kline, err := raw.ToKline()
		if err != nil {
			return nil, fmt.Errorf("转换K线数据失败: %w", err)
		}
		klines = append(klines, kline)
	}

	return klines, nil
}
