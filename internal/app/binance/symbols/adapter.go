// Package symbols 币安交易对适配器实现
package symbols

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/symbols"
	"trpc.group/trpc-go/trpc-go/log"
)

// BinanceSymbolAdapter 币安交易对适配器
type BinanceSymbolAdapter struct {
	baseURL    string
	httpClient *http.Client
	exchange   string
}

// NewBinanceSymbolAdapter 创建币安交易对适配器
func NewBinanceSymbolAdapter(baseURL, exchange string) *BinanceSymbolAdapter {
	if baseURL == "" {
		baseURL = "https://api.binance.com"
	}
	if exchange == "" {
		exchange = "binance"
	}
	
	return &BinanceSymbolAdapter{
		baseURL:  baseURL,
		exchange: exchange,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetExchange 获取交易所名称
func (a *BinanceSymbolAdapter) GetExchange() string {
	return a.exchange
}

// FetchAll 获取所有交易对
func (a *BinanceSymbolAdapter) FetchAll(ctx context.Context) ([]*symbols.SymbolMeta, error) {
	// 构建请求URL - 根据baseURL判断是现货还是合约
	var url string
	if a.baseURL == "https://fapi.binance.com" {
		url = fmt.Sprintf("%s/fapi/v1/exchangeInfo", a.baseURL) // 合约API
	} else {
		url = fmt.Sprintf("%s/api/v3/exchangeInfo", a.baseURL)  // 现货API
	}
	
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var exchangeInfo ExchangeInfo
	if err := json.NewDecoder(resp.Body).Decode(&exchangeInfo); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	
	// 转换为标准格式
	result := make([]*symbols.SymbolMeta, 0, len(exchangeInfo.Symbols))
	for _, rawSymbol := range exchangeInfo.Symbols {
		symbolMeta, err := a.parseRawSymbol(rawSymbol)
		if err != nil {
			log.Warnf("解析交易对数据失败: %v", err)
			continue
		}
		
		result = append(result, symbolMeta)
	}
	
	return result, nil
}

// FetchSymbol 获取单个交易对
func (a *BinanceSymbolAdapter) FetchSymbol(ctx context.Context, symbol string) (*symbols.SymbolMeta, error) {
	// 币安没有单独获取单个交易对的API，需要获取全部然后过滤
	allSymbols, err := a.FetchAll(ctx)
	if err != nil {
		return nil, err
	}
	
	for _, s := range allSymbols {
		if s.Symbol == symbol {
			return s, nil
		}
	}
	
	return nil, fmt.Errorf("交易对 %s 不存在", symbol)
}

// IsSupported 检查是否支持该交易对
func (a *BinanceSymbolAdapter) IsSupported(symbol string) bool {
	// 这里可以实现更复杂的逻辑，比如缓存支持的交易对列表
	// 暂时简单返回true，表示支持所有交易对
	return symbol != ""
}

// parseRawSymbol 解析原始交易对数据
func (a *BinanceSymbolAdapter) parseRawSymbol(raw RawSymbol) (*symbols.SymbolMeta, error) {
	if raw.Symbol == "" {
		return nil, fmt.Errorf("交易对符号为空")
	}
	
	// 确定交易对类型
	var symbolType symbols.SymbolType
	if a.baseURL == "https://fapi.binance.com" {
		if raw.ContractType == "PERPETUAL" {
			symbolType = symbols.TypePerp
		} else {
			symbolType = symbols.TypeDelivery
		}
	} else {
		symbolType = symbols.TypeSpot
	}
	
	// 解析状态
	status := symbols.SymbolStatus(raw.Status)
	
	// 解析时间
	var listingTime time.Time
	if raw.OnboardDate > 0 {
		listingTime = time.UnixMilli(raw.OnboardDate)
	}
	
	// 解析过滤器
	filters := make(map[string]interface{})
	for _, filter := range raw.Filters {
		filters[filter.FilterType] = filter
	}
	
	// 解析精度
	pricePrecision := 8 // 默认值
	quantityPrecision := 8 // 默认值
	if raw.PricePrecision != nil {
		pricePrecision = *raw.PricePrecision
	}
	if raw.QuantityPrecision != nil {
		quantityPrecision = *raw.QuantityPrecision
	}
	
	symbolMeta := &symbols.SymbolMeta{
		Exchange:   a.exchange,
		Symbol:     raw.Symbol,
		BaseAsset:  raw.BaseAsset,
		QuoteAsset: raw.QuoteAsset,
		Status:     status,
		Type:       symbolType,
		
		// 合约特有字段
		ContractType: raw.ContractType,
		DeliveryTime: raw.DeliveryDate,
		Multiplier:   raw.Multiplier,
		
		// 精度信息
		PricePrecision:    pricePrecision,
		QuantityPrecision: quantityPrecision,
		
		// 时间信息
		ListingTime: listingTime,
		UpdateTime:  time.Now(),
		
		// 权限和过滤器
		Permissions: raw.Permissions,
		Filters:     filters,
		
		// 扩展字段
		Extra: map[string]interface{}{
			"orderTypes":         raw.OrderTypes,
			"timeInForce":        raw.TimeInForce,
			"icebergAllowed":     raw.IcebergAllowed,
			"ocoAllowed":         raw.OcoAllowed,
			"quoteOrderQtyMarketAllowed": raw.QuoteOrderQtyMarketAllowed,
			"allowTrailingStop":  raw.AllowTrailingStop,
			"cancelReplaceAllowed": raw.CancelReplaceAllowed,
		},
		Raw: map[string]interface{}{
			"original": raw,
		},
	}
	
	// 从过滤器中提取交易限制
	a.extractTradingLimits(symbolMeta, raw.Filters)
	
	return symbolMeta, nil
}

// extractTradingLimits 从过滤器中提取交易限制
func (a *BinanceSymbolAdapter) extractTradingLimits(meta *symbols.SymbolMeta, filters []Filter) {
	for _, filter := range filters {
		switch filter.FilterType {
		case "PRICE_FILTER":
			meta.MinPrice = filter.MinPrice
			meta.MaxPrice = filter.MaxPrice
			meta.TickSize = filter.TickSize
		case "LOT_SIZE":
			meta.MinQty = filter.MinQty
			meta.MaxQty = filter.MaxQty
			meta.StepSize = filter.StepSize
		case "MIN_NOTIONAL":
			meta.MinNotional = filter.MinNotional
		case "NOTIONAL":
			if meta.MinNotional == "" {
				meta.MinNotional = filter.MinNotional
			}
		}
	}
}

// ExchangeInfo 交易所信息响应结构
type ExchangeInfo struct {
	Timezone   string      `json:"timezone"`
	ServerTime int64       `json:"serverTime"`
	Symbols    []RawSymbol `json:"symbols"`
}

// RawSymbol 原始交易对数据结构
type RawSymbol struct {
	Symbol                     string   `json:"symbol"`
	Status                     string   `json:"status"`
	BaseAsset                  string   `json:"baseAsset"`
	QuoteAsset                 string   `json:"quoteAsset"`
	BaseAssetPrecision         *int     `json:"baseAssetPrecision"`
	QuoteAssetPrecision        *int     `json:"quoteAssetPrecision"`
	PricePrecision             *int     `json:"pricePrecision"`
	QuantityPrecision          *int     `json:"quantityPrecision"`
	OrderTypes                 []string `json:"orderTypes"`
	IcebergAllowed             bool     `json:"icebergAllowed"`
	OcoAllowed                 bool     `json:"ocoAllowed"`
	QuoteOrderQtyMarketAllowed bool     `json:"quoteOrderQtyMarketAllowed"`
	AllowTrailingStop          bool     `json:"allowTrailingStop"`
	CancelReplaceAllowed       bool     `json:"cancelReplaceAllowed"`
	IsSpotTradingAllowed       bool     `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool     `json:"isMarginTradingAllowed"`
	Filters                    []Filter `json:"filters"`
	Permissions                []string `json:"permissions"`
	TimeInForce                []string `json:"timeInForce"`
	
	// 合约特有字段
	ContractType string `json:"contractType,omitempty"`
	DeliveryDate int64  `json:"deliveryDate,omitempty"`
	OnboardDate  int64  `json:"onboardDate,omitempty"`
	Multiplier   string `json:"multiplier,omitempty"`
}

// Filter 过滤器结构
type Filter struct {
	FilterType       string `json:"filterType"`
	MinPrice         string `json:"minPrice,omitempty"`
	MaxPrice         string `json:"maxPrice,omitempty"`
	TickSize         string `json:"tickSize,omitempty"`
	MinQty           string `json:"minQty,omitempty"`
	MaxQty           string `json:"maxQty,omitempty"`
	StepSize         string `json:"stepSize,omitempty"`
	MinNotional      string `json:"minNotional,omitempty"`
	ApplyToMarket    bool   `json:"applyToMarket,omitempty"`
	AvgPriceMins     int    `json:"avgPriceMins,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	MaxNumOrders     int    `json:"maxNumOrders,omitempty"`
	MaxNumAlgoOrders int    `json:"maxNumAlgoOrders,omitempty"`
}
