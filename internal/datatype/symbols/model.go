package symbols

import (
	"fmt"
	"strconv"
	"time"
)

// SymbolType 交易对类型
type SymbolType string

const (
	TypeSpot     SymbolType = "spot"     // 现货
	TypePerp     SymbolType = "perp"     // 永续合约
	TypeDelivery SymbolType = "delivery" // 交割合约
	TypeOption   SymbolType = "option"   // 期权
)

// SymbolStatus 交易对状态
type SymbolStatus string

const (
	StatusTrading      SymbolStatus = "TRADING"       // 交易中
	StatusOffline      SymbolStatus = "OFFLINE"       // 离线
	StatusListing      SymbolStatus = "LISTING"       // 上市中
	StatusDelisting    SymbolStatus = "DELISTING"     // 下市中
	StatusBreak        SymbolStatus = "BREAK"         // 暂停
	StatusHalt         SymbolStatus = "HALT"          // 停牌
	StatusAuctionMatch SymbolStatus = "AUCTION_MATCH" // 集合竞价
)

// SymbolMeta 统一描述一个可交易"符号"（现货交易对/合约/期权等）
type SymbolMeta struct {
	// 基础信息
	Exchange string     `json:"exchange"` // 交易所
	Symbol   string     `json:"symbol"`   // 交易对符号
	
	// 资产信息
	BaseAsset  string `json:"baseAsset"`  // 基础资产
	QuoteAsset string `json:"quoteAsset"` // 计价资产
	
	// 状态和类型
	Status SymbolStatus `json:"status"` // 交易状态
	Type   SymbolType   `json:"type"`   // 交易对类型
	
	// 合约特有字段
	ContractType string `json:"contractType,omitempty"` // 合约类型 (PERPETUAL/CURRENT_QUARTER等)
	DeliveryTime int64  `json:"deliveryTime,omitempty"` // 交割时间(毫秒)，无则0
	Multiplier   string `json:"multiplier,omitempty"`   // 合约乘数
	
	// 精度信息
	PricePrecision    int `json:"pricePrecision"`    // 价格精度
	QuantityPrecision int `json:"quantityPrecision"` // 数量精度
	
	// 交易限制
	MinQty      string `json:"minQty"`      // 最小数量
	MaxQty      string `json:"maxQty"`      // 最大数量
	MinNotional string `json:"minNotional"` // 最小名义价值
	MaxPrice    string `json:"maxPrice"`    // 最大价格
	MinPrice    string `json:"minPrice"`    // 最小价格
	
	// 步长信息
	TickSize string `json:"tickSize"` // 价格步长
	StepSize string `json:"stepSize"` // 数量步长
	
	// 时间信息
	ListingTime time.Time `json:"listingTime"` // 上市时间
	UpdateTime  time.Time `json:"updateTime"`  // 更新时间
	
	// 权限和过滤器
	Permissions []string                   `json:"permissions"` // 交易权限
	Filters     map[string]interface{}     `json:"filters"`     // 过滤器
	
	// 扩展字段
	Extra map[string]interface{} `json:"extra"` // 扩展字段
	Raw   map[string]interface{} `json:"raw"`   // 原始数据
}

// Key 返回交易对的唯一标识
func (s *SymbolMeta) Key() string {
	return fmt.Sprintf("%s:%s", s.Exchange, s.Symbol)
}

// IsActive 检查交易对是否活跃
func (s *SymbolMeta) IsActive() bool {
	return s.Status == StatusTrading
}

// IsSpot 检查是否为现货交易对
func (s *SymbolMeta) IsSpot() bool {
	return s.Type == TypeSpot
}

// IsContract 检查是否为合约
func (s *SymbolMeta) IsContract() bool {
	return s.Type == TypePerp || s.Type == TypeDelivery
}

// IsOption 检查是否为期权
func (s *SymbolMeta) IsOption() bool {
	return s.Type == TypeOption
}

// GetMinQtyFloat 获取最小数量浮点数
func (s *SymbolMeta) GetMinQtyFloat() (float64, error) {
	if s.MinQty == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s.MinQty, 64)
}

// GetMaxQtyFloat 获取最大数量浮点数
func (s *SymbolMeta) GetMaxQtyFloat() (float64, error) {
	if s.MaxQty == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s.MaxQty, 64)
}

// GetMinNotionalFloat 获取最小名义价值浮点数
func (s *SymbolMeta) GetMinNotionalFloat() (float64, error) {
	if s.MinNotional == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s.MinNotional, 64)
}

// GetTickSizeFloat 获取价格步长浮点数
func (s *SymbolMeta) GetTickSizeFloat() (float64, error) {
	if s.TickSize == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s.TickSize, 64)
}

// GetStepSizeFloat 获取数量步长浮点数
func (s *SymbolMeta) GetStepSizeFloat() (float64, error) {
	if s.StepSize == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s.StepSize, 64)
}

// GetMultiplierFloat 获取合约乘数浮点数
func (s *SymbolMeta) GetMultiplierFloat() (float64, error) {
	if s.Multiplier == "" {
		return 1, nil // 默认乘数为1
	}
	return strconv.ParseFloat(s.Multiplier, 64)
}

// Clone 克隆交易对元数据
func (s *SymbolMeta) Clone() *SymbolMeta {
	clone := *s
	
	// 深拷贝切片和映射
	if s.Permissions != nil {
		clone.Permissions = make([]string, len(s.Permissions))
		copy(clone.Permissions, s.Permissions)
	}
	
	if s.Filters != nil {
		clone.Filters = make(map[string]interface{})
		for k, v := range s.Filters {
			clone.Filters[k] = v
		}
	}
	
	if s.Extra != nil {
		clone.Extra = make(map[string]interface{})
		for k, v := range s.Extra {
			clone.Extra[k] = v
		}
	}
	
	if s.Raw != nil {
		clone.Raw = make(map[string]interface{})
		for k, v := range s.Raw {
			clone.Raw[k] = v
		}
	}
	
	return &clone
}

// SymbolFilter 交易对过滤器（保持向后兼容）
type SymbolFilter struct {
	Exchange    string       `json:"exchange,omitempty"`    // 交易所过滤
	Symbol      string       `json:"symbol,omitempty"`      // 交易对过滤
	BaseAsset   string       `json:"baseAsset,omitempty"`   // 基础资产过滤
	QuoteAsset  string       `json:"quoteAsset,omitempty"`  // 计价资产过滤
	Status      SymbolStatus `json:"status,omitempty"`      // 状态过滤
	Type        SymbolType   `json:"type,omitempty"`        // 类型过滤
	ActiveOnly  bool         `json:"activeOnly"`            // 仅活跃交易对
	Limit       int          `json:"limit,omitempty"`       // 数量限制
}

// Match 检查交易对是否匹配过滤条件
func (f *SymbolFilter) Match(symbol *SymbolMeta) bool {
	if f.Exchange != "" && f.Exchange != symbol.Exchange {
		return false
	}
	
	if f.Symbol != "" && f.Symbol != symbol.Symbol {
		return false
	}
	
	if f.BaseAsset != "" && f.BaseAsset != symbol.BaseAsset {
		return false
	}
	
	if f.QuoteAsset != "" && f.QuoteAsset != symbol.QuoteAsset {
		return false
	}
	
	if f.Status != "" && f.Status != symbol.Status {
		return false
	}
	
	if f.Type != "" && f.Type != symbol.Type {
		return false
	}
	
	if f.ActiveOnly && !symbol.IsActive() {
		return false
	}
	
	return true
}

// SymbolSnapshot 交易对快照（保持向后兼容）
type SymbolSnapshot struct {
	Exchange     string        `json:"exchange"`     // 交易所
	SnapshotTime time.Time     `json:"snapshotTime"` // 快照时间
	SymbolCount  int           `json:"symbolCount"`  // 交易对数量
	Symbols      []*SymbolMeta `json:"symbols"`      // 交易对列表
}

// SymbolStats 交易对统计信息（保持向后兼容）
type SymbolStats struct {
	Exchange      string    `json:"exchange"`      // 交易所
	TotalCount    int       `json:"totalCount"`    // 总数量
	ActiveCount   int       `json:"activeCount"`   // 活跃数量
	SpotCount     int       `json:"spotCount"`     // 现货数量
	ContractCount int       `json:"contractCount"` // 合约数量
	OptionCount   int       `json:"optionCount"`   // 期权数量
	LastUpdate    time.Time `json:"lastUpdate"`    // 最后更新时间
}

// SymbolRange 交易对范围（保持向后兼容）
type SymbolRange struct {
	Exchange  string    `json:"exchange"`  // 交易所
	StartTime time.Time `json:"startTime"` // 开始时间
	EndTime   time.Time `json:"endTime"`   // 结束时间
}

// SymbolGap 交易对缺口（保持向后兼容）
type SymbolGap struct {
	Exchange  string    `json:"exchange"`  // 交易所
	Symbol    string    `json:"symbol"`    // 交易对
	StartTime time.Time `json:"startTime"` // 缺口开始时间
	EndTime   time.Time `json:"endTime"`   // 缺口结束时间
	Duration  time.Duration `json:"duration"` // 缺口持续时间
}
