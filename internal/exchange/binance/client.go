package binance

import "github.com/mooyang-code/data-collector/pkg/httpclient"

// 域名常量
const (
	SpotDomain = "api.binance.com"  // 现货域名
	SwapDomain = "fapi.binance.com" // U本位永续合约域名
)

// API 端点
const (
	SpotKlineEndpoint = "/api/v3/klines"  // 现货K线
	SwapKlineEndpoint = "/fapi/v1/klines" // 永续合约K线
)

// Client 币安客户端
type Client struct {
	*httpclient.HTTPClient
}

// NewClient 创建币安客户端
func NewClient() *Client {
	return &Client{
		HTTPClient: httpclient.NewHTTPClient(),
	}
}
