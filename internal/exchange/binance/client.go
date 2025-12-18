package binance

import (
	"time"

	"github.com/mooyang-code/data-collector/pkg/httpclient"
)

// API 基础 URL
const (
	SpotBaseURL = "https://api.binance.com" // 现货
	SwapBaseURL = "https://fapi.binance.com" // U本位永续合约
)

// API 端点
const (
	SpotKlineEndpoint = "/api/v3/klines"  // 现货K线
	SwapKlineEndpoint = "/fapi/v1/klines" // 永续合约K线
)

// Client 币安客户端
type Client struct {
	spotClient *httpclient.Client // 现货客户端
	swapClient *httpclient.Client // 永续合约客户端
}

// NewClient 创建币安客户端
func NewClient() *Client {
	return &Client{
		spotClient: httpclient.New(
			httpclient.WithBaseURL(SpotBaseURL),
			httpclient.WithTimeout(30*time.Second),
		),
		swapClient: httpclient.New(
			httpclient.WithBaseURL(SwapBaseURL),
			httpclient.WithTimeout(30*time.Second),
		),
	}
}

// SpotClient 获取现货客户端
func (c *Client) SpotClient() *httpclient.Client {
	return c.spotClient
}

// SwapClient 获取永续合约客户端
func (c *Client) SwapClient() *httpclient.Client {
	return c.swapClient
}
