// Package configs 配置管理（最好是中文注释！）
package configs

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	System SystemConfig `yaml:"system"`
	Apps   AppsConfig   `yaml:"apps"`

	// 保留原有的持久化配置
	Persistence struct {
		Backend string `yaml:"backend"`
		Parquet struct {
			BasePath      string `yaml:"basePath"`
			Compression   string `yaml:"compression"`
			RowGroupSize  int    `yaml:"rowGroupSize"`
			PageSize      int    `yaml:"pageSize"`
			FlushInterval string `yaml:"flushInterval"`
		} `yaml:"parquet"`
	} `yaml:"persistence"`
}

// SystemConfig 系统配置结构
type SystemConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
	Debug       bool   `yaml:"debug"`
}

// AppsConfig 应用配置结构
type AppsConfig struct {
	Binance AppConfig `yaml:"binance"`
	OKX     AppConfig `yaml:"okx"`
	Bybit   AppConfig `yaml:"bybit"`
}

// AppConfig 单个应用配置结构
type AppConfig struct {
	Name       string               `yaml:"name"`
	Enabled    bool                 `yaml:"enabled"`
	BaseConfig BaseConfig           `yaml:"baseConfig"`
	Collectors map[string]Collector `yaml:"collectors"`
}

// BaseConfig 基础配置结构
type BaseConfig struct {
	RestBaseURL string `yaml:"restBaseURL"`
	WsBaseURL   string `yaml:"wsBaseURL"`
	LogLevel    string `yaml:"logLevel"`
}

// Collector 采集器配置结构
type Collector struct {
	Name        string          `yaml:"name"`
	Enabled     bool            `yaml:"enabled"`
	DataType    string          `yaml:"dataType"`
	MarketType  string          `yaml:"marketType"`
	RestBaseURL string          `yaml:"restBaseURL"`
	Schedule    ScheduleConfig  `yaml:"schedule"`
	Config      CollectorConfig `yaml:"config"`
}

// ScheduleConfig 调度配置结构
type ScheduleConfig struct {
	TriggerInterval   string `yaml:"triggerInterval"`
	EnableAutoRefresh bool   `yaml:"enableAutoRefresh"`
	EnableRealtime    bool   `yaml:"enableRealtime"`
	MaxRetries        int    `yaml:"maxRetries"`
	RetryInterval     string `yaml:"retryInterval"`
}

// CollectorConfig 采集器具体配置结构
type CollectorConfig struct {
	// 通用配置
	EnablePersistence bool     `yaml:"enablePersistence"`
	PersistenceFile   string   `yaml:"persistenceFile"`
	Filters           []string `yaml:"filters"`

	// K线特有配置
	Intervals            []string      `yaml:"intervals"`
	BufferSize           int           `yaml:"bufferSize"`
	MaxKlinesPerSymbol   int           `yaml:"maxKlinesPerSymbol"`
	EnableAggregation    bool          `yaml:"enableAggregation"`
	EnableGapDetection   bool          `yaml:"enableGapDetection"`
	GapDetectionInterval string        `yaml:"gapDetectionInterval"`
	EnableBackfill       bool          `yaml:"enableBackfill"`
	BackfillLookback     string        `yaml:"backfillLookback"`
	History              HistoryConfig `yaml:"history"`
}

// HistoryConfig 历史数据配置结构
type HistoryConfig struct {
	Enable        bool   `yaml:"enable"`
	LookbackHours int    `yaml:"lookbackHours"`
	BatchSize     int    `yaml:"batchSize"`
	Interval      string `yaml:"interval"`
}

// Load 加载配置文件
func Load(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if err := setDefaults(&config); err != nil {
		return nil, fmt.Errorf("设置默认配置失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	return &config, nil
}

// setDefaults 设置配置默认值
func setDefaults(config *Config) error {
	// 系统默认值
	if config.System.Name == "" {
		config.System.Name = "data-collector"
	}
	if config.System.Version == "" {
		config.System.Version = "1.0.0"
	}
	if config.System.Environment == "" {
		config.System.Environment = "development"
	}

	// Binance应用默认值
	setAppDefaults(&config.Apps.Binance, "binance-app", "https://api.binance.com", "wss://stream.binance.com:9443/ws")

	// OKX应用默认值
	setAppDefaults(&config.Apps.OKX, "okx-app", "https://www.okx.com", "wss://ws.okx.com:8443/ws/v5/public")

	// Bybit应用默认值
	setAppDefaults(&config.Apps.Bybit, "bybit-app", "https://api.bybit.com", "wss://stream.bybit.com/v5/public/spot")

	return nil
}

// setAppDefaults 设置单个应用的默认值
func setAppDefaults(app *AppConfig, defaultName, defaultRestURL, defaultWsURL string) {
	if app.Name == "" {
		app.Name = defaultName
	}

	if app.BaseConfig.RestBaseURL == "" {
		app.BaseConfig.RestBaseURL = defaultRestURL
	}

	if app.BaseConfig.WsBaseURL == "" {
		app.BaseConfig.WsBaseURL = defaultWsURL
	}

	if app.BaseConfig.LogLevel == "" {
		app.BaseConfig.LogLevel = "info"
	}

	// 为每个采集器设置默认值
	for name, collector := range app.Collectors {
		setCollectorDefaults(&collector, name)
		app.Collectors[name] = collector
	}
}

// setCollectorDefaults 设置采集器默认值
func setCollectorDefaults(collector *Collector, name string) {
	if collector.Name == "" {
		collector.Name = name
	}

	// 调度默认值
	if collector.Schedule.TriggerInterval == "" {
		if collector.DataType == "symbols" {
			collector.Schedule.TriggerInterval = "5m"
		} else if collector.DataType == "klines" {
			collector.Schedule.TriggerInterval = "1m"
		}
	}

	if collector.Schedule.MaxRetries == 0 {
		collector.Schedule.MaxRetries = 3
	}

	if collector.Schedule.RetryInterval == "" {
		collector.Schedule.RetryInterval = "30s"
	}

	// K线采集器特有默认值
	if collector.DataType == "klines" {
		if len(collector.Config.Intervals) == 0 {
			collector.Config.Intervals = []string{"1m", "5m", "1h"}
		}

		if collector.Config.BufferSize == 0 {
			collector.Config.BufferSize = 1000
		}

		if collector.Config.MaxKlinesPerSymbol == 0 {
			collector.Config.MaxKlinesPerSymbol = 10000
		}

		if collector.Config.GapDetectionInterval == "" {
			collector.Config.GapDetectionInterval = "5m"
		}

		if collector.Config.BackfillLookback == "" {
			collector.Config.BackfillLookback = "24h"
		}

		if collector.Config.History.BatchSize == 0 {
			collector.Config.History.BatchSize = 1000
		}

		if collector.Config.History.LookbackHours == 0 {
			collector.Config.History.LookbackHours = 8
		}

		if collector.Config.History.Interval == "" {
			collector.Config.History.Interval = "1h"
		}
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证系统配置
	if c.System.Name == "" {
		return fmt.Errorf("system name is required")
	}

	// 验证应用配置
	if err := c.validateApp("binance", &c.Apps.Binance); err != nil {
		return fmt.Errorf("binance app validation failed: %w", err)
	}

	if err := c.validateApp("okx", &c.Apps.OKX); err != nil {
		return fmt.Errorf("okx app validation failed: %w", err)
	}

	if err := c.validateApp("bybit", &c.Apps.Bybit); err != nil {
		return fmt.Errorf("bybit app validation failed: %w", err)
	}

	return nil
}

// validateApp 验证单个应用配置
func (c *Config) validateApp(appName string, app *AppConfig) error {
	if !app.Enabled {
		return nil // 未启用的应用跳过验证
	}

	if app.BaseConfig.RestBaseURL == "" {
		return fmt.Errorf("restBaseURL is required for app %s", appName)
	}

	// 验证采集器配置
	for collectorName, collector := range app.Collectors {
		if err := c.validateCollector(appName, collectorName, &collector); err != nil {
			return fmt.Errorf("collector %s validation failed: %w", collectorName, err)
		}
	}

	return nil
}

// validateCollector 验证采集器配置
func (c *Config) validateCollector(appName, collectorName string, collector *Collector) error {
	if !collector.Enabled {
		return nil // 未启用的采集器跳过验证
	}

	// 验证数据类型
	if collector.DataType != "symbols" && collector.DataType != "klines" {
		return fmt.Errorf("invalid dataType %s, must be 'symbols' or 'klines'", collector.DataType)
	}

	// 验证市场类型
	if collector.MarketType != "spot" && collector.MarketType != "futures" {
		return fmt.Errorf("invalid marketType %s, must be 'spot' or 'futures'", collector.MarketType)
	}

	// 验证时间间隔格式
	if collector.Schedule.TriggerInterval != "" {
		if _, err := time.ParseDuration(collector.Schedule.TriggerInterval); err != nil {
			return fmt.Errorf("invalid triggerInterval: %w", err)
		}
	}

	if collector.Schedule.RetryInterval != "" {
		if _, err := time.ParseDuration(collector.Schedule.RetryInterval); err != nil {
			return fmt.Errorf("invalid retryInterval: %w", err)
		}
	}

	// 验证K线特有配置
	if collector.DataType == "klines" {
		if collector.Config.GapDetectionInterval != "" {
			if _, err := time.ParseDuration(collector.Config.GapDetectionInterval); err != nil {
				return fmt.Errorf("invalid gapDetectionInterval: %w", err)
			}
		}

		if collector.Config.BackfillLookback != "" {
			if _, err := time.ParseDuration(collector.Config.BackfillLookback); err != nil {
				return fmt.Errorf("invalid backfillLookback: %w", err)
			}
		}

		if collector.Config.History.Interval != "" {
			if _, err := time.ParseDuration(collector.Config.History.Interval); err != nil {
				return fmt.Errorf("invalid history interval: %w", err)
			}
		}
	}

	return nil
}

// GetAppConfig 获取指定应用配置
func (c *Config) GetAppConfig(appName string) (*AppConfig, error) {
	switch appName {
	case "binance":
		return &c.Apps.Binance, nil
	case "okx":
		return &c.Apps.OKX, nil
	case "bybit":
		return &c.Apps.Bybit, nil
	default:
		return nil, fmt.Errorf("unknown app: %s", appName)
	}
}

// GetCollector 获取指定应用的指定采集器配置
func (c *Config) GetCollector(appName, collectorName string) (*Collector, error) {
	app, err := c.GetAppConfig(appName)
	if err != nil {
		return nil, err
	}

	collector, exists := app.Collectors[collectorName]
	if !exists {
		return nil, fmt.Errorf("collector %s not found in app %s", collectorName, appName)
	}

	return &collector, nil
}

// GetEnabledApps 获取所有启用的应用
func (c *Config) GetEnabledApps() map[string]*AppConfig {
	enabled := make(map[string]*AppConfig)

	if c.Apps.Binance.Enabled {
		enabled["binance"] = &c.Apps.Binance
	}
	if c.Apps.OKX.Enabled {
		enabled["okx"] = &c.Apps.OKX
	}
	if c.Apps.Bybit.Enabled {
		enabled["bybit"] = &c.Apps.Bybit
	}

	return enabled
}

// GetEnabledCollectors 获取指定应用的所有启用的采集器
func (c *Config) GetEnabledCollectors(appName string) (map[string]*Collector, error) {
	app, err := c.GetAppConfig(appName)
	if err != nil {
		return nil, err
	}

	enabled := make(map[string]*Collector)
	for name, collector := range app.Collectors {
		if collector.Enabled {
			collectorCopy := collector
			enabled[name] = &collectorCopy
		}
	}

	return enabled, nil
}

// GetCollectorsByType 获取指定数据类型的所有启用采集器
func (c *Config) GetCollectorsByType(dataType string) map[string]map[string]*Collector {
	result := make(map[string]map[string]*Collector)

	apps := c.GetEnabledApps()
	for appName, app := range apps {
		appCollectors := make(map[string]*Collector)
		for collectorName, collector := range app.Collectors {
			if collector.Enabled && collector.DataType == dataType {
				collectorCopy := collector
				appCollectors[collectorName] = &collectorCopy
			}
		}
		if len(appCollectors) > 0 {
			result[appName] = appCollectors
		}
	}

	return result
}

// GetCollectorsByMarketType 获取指定市场类型的所有启用采集器
func (c *Config) GetCollectorsByMarketType(marketType string) map[string]map[string]*Collector {
	result := make(map[string]map[string]*Collector)

	apps := c.GetEnabledApps()
	for appName, app := range apps {
		appCollectors := make(map[string]*Collector)
		for collectorName, collector := range app.Collectors {
			if collector.Enabled && collector.MarketType == marketType {
				collectorCopy := collector
				appCollectors[collectorName] = &collectorCopy
			}
		}
		if len(appCollectors) > 0 {
			result[appName] = appCollectors
		}
	}

	return result
}

// GetCollectorRestURL 获取采集器的REST API URL
func (c *Collector) GetCollectorRestURL(baseURL string) string {
	if c.RestBaseURL != "" {
		return c.RestBaseURL
	}
	return baseURL
}

// IsSymbolsCollector 判断是否为交易对采集器
func (c *Collector) IsSymbolsCollector() bool {
	return c.DataType == "symbols"
}

// IsKlinesCollector 判断是否为K线采集器
func (c *Collector) IsKlinesCollector() bool {
	return c.DataType == "klines"
}

// IsSpotCollector 判断是否为现货采集器
func (c *Collector) IsSpotCollector() bool {
	return c.MarketType == "spot"
}

// IsFuturesCollector 判断是否为合约采集器
func (c *Collector) IsFuturesCollector() bool {
	return c.MarketType == "futures"
}

// GetTriggerDuration 获取触发间隔的Duration
func (c *Collector) GetTriggerDuration() (time.Duration, error) {
	return time.ParseDuration(c.Schedule.TriggerInterval)
}

// GetRetryDuration 获取重试间隔的Duration
func (c *Collector) GetRetryDuration() (time.Duration, error) {
	return time.ParseDuration(c.Schedule.RetryInterval)
}
