package binance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/internal/exchange"
	binanceapi "github.com/mooyang-code/data-collector/internal/exchange/binance"
	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go/log"
)

// SymbolCollector 标的同步采集器
type SymbolCollector struct {
	client  *binanceapi.Client
	spotAPI *binanceapi.SpotAPI
	swapAPI *binanceapi.SwapAPI
}

// Source 返回数据源标识
func (c *SymbolCollector) Source() string {
	return "binance"
}

// DataType 返回数据类型标识
func (c *SymbolCollector) DataType() string {
	return "symbol"
}

func init() {
	client := binanceapi.NewClient()
	c := &SymbolCollector{
		client:  client,
		spotAPI: binanceapi.NewSpotAPI(client),
		swapAPI: binanceapi.NewSwapAPI(client),
	}

	// 注册到全局注册中心
	err := collector.NewBuilder().
		Source("binance", "币安").
		DataType("symbol", "标的").
		Description("币安交易所标的同步采集器").
		Collector(c).
		Register()

	if err != nil {
		log.Errorf("注册币安标的采集器失败: %v", err)
	}
}

// Collect 执行标的同步采集
func (c *SymbolCollector) Collect(ctx context.Context, params *collector.CollectParams) error {
	log.InfoContextf(ctx, "[SymbolCollector] 开始采集标的, InstType=%s", params.InstType)

	// 根据产品类型获取标的列表
	symbols, err := c.fetchSymbols(ctx, params)
	if err != nil {
		log.ErrorContextf(ctx, "[SymbolCollector] 获取标的失败: %v", err)
		return err
	}

	log.InfoContextf(ctx, "[SymbolCollector] 获取标的成功（过滤前）, count=%d, InstType=%s",
		len(symbols), params.InstType)

	// 过滤标的：仅保留 QuoteAsset 为 USDT 且 Status 为 active 的数据
	filteredSymbols := c.filterSymbols(symbols)
	log.InfoContextf(ctx, "[SymbolCollector] 过滤后标的数量, count=%d (过滤前: %d), InstType=%s",
		len(filteredSymbols), len(symbols), params.InstType)

	// 上报标的到 Server
	if err := c.reportSymbols(ctx, params.InstType, filteredSymbols); err != nil {
		log.ErrorContextf(ctx, "[SymbolCollector] 上报标的失败: %v", err)
		return err
	}

	log.InfoContextf(ctx, "[SymbolCollector] 标的采集完成, InstType=%s", params.InstType)
	return nil
}

// fetchSymbols 获取标的列表
func (c *SymbolCollector) fetchSymbols(ctx context.Context, params *collector.CollectParams) ([]*exchange.SymbolInfo, error) {
	switch params.InstType {
	case InstTypeSPOT:
		return c.spotAPI.GetExchangeInfo(ctx)
	case InstTypeSWAP:
		return c.swapAPI.GetExchangeInfo(ctx)
	default:
		return nil, fmt.Errorf("不支持的产品类型: %s", params.InstType)
	}
}

// filterSymbols 过滤标的列表，仅保留 QuoteAsset 为 USDT 且 Status 为 active 的数据
func (c *SymbolCollector) filterSymbols(symbols []*exchange.SymbolInfo) []*exchange.SymbolInfo {
	filtered := make([]*exchange.SymbolInfo, 0, len(symbols))

	for _, symbol := range symbols {
		// 仅保留 QuoteAsset 为 USDT 且 Status 为 active 的标的
		if symbol.QuoteAsset == "USDT" && symbol.Status == "active" {
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// reportSymbols 上报标的到 Server
func (c *SymbolCollector) reportSymbols(ctx context.Context, instType string, symbols []*exchange.SymbolInfo) error {
	// 构建上报请求
	request := &SyncSymbolsRequest{
		Exchange: "binance",
		InstType: instType,
		Symbols:  symbols,
		SyncTime: time.Now().UnixMilli(),
	}

	// 序列化为 JSON
	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("序列化上报数据失败: %w", err)
	}

	// 获取 Server 地址
	serverURL := config.GetServerURL()
	if serverURL == "" {
		return fmt.Errorf("未配置 Server 地址")
	}

	// 发送 HTTP POST 请求
	url := fmt.Sprintf("%s/api/v1/collector/symbols/sync", serverURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	var respBody bytes.Buffer
	_, _ = respBody.ReadFrom(resp.Body)
	respData := respBody.String()

	// 检查响应
	if resp.StatusCode != http.StatusOK {
		log.ErrorContextf(ctx, "[SymbolCollector] 上报失败，HTTP状态码: %d, 响应内容: %s",
			resp.StatusCode, respData)
		return fmt.Errorf("上报失败，HTTP状态码: %d", resp.StatusCode)
	}

	log.InfoContextf(ctx, "[SymbolCollector] 上报标的成功, count=%d, 响应内容: %s",
		len(symbols), respData)
	return nil
}

// SyncSymbolsRequest 同步标的请求
type SyncSymbolsRequest struct {
	Exchange string                 `json:"exchange"`  // 交易所
	InstType string                 `json:"inst_type"` // 产品类型
	Symbols  []*exchange.SymbolInfo `json:"symbols"`   // 标的列表
	SyncTime int64                  `json:"sync_time"` // 同步时间戳
}
