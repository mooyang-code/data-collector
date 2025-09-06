package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	
	"gopkg.in/yaml.v3"
)

// Config 主配置结构
type Config struct {
	System     SystemConfig     `yaml:"system"`
	Logging    LoggingConfig    `yaml:"logging"`
	EventBus   EventBusConfig   `yaml:"event_bus"`
	Storage    StorageConfig    `yaml:"storage"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Sources    SourcesConfig    `yaml:"sources"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
	Timezone    string `yaml:"timezone"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string         `yaml:"level"`
	Format string         `yaml:"format"`
	Output []OutputConfig `yaml:"output"`
}

// OutputConfig 日志输出配置
type OutputConfig struct {
	Type      string `yaml:"type"`
	Level     string `yaml:"level"`
	Path      string `yaml:"path"`
	Rotation  string `yaml:"rotation"`
	Retention int    `yaml:"retention"`
}

// EventBusConfig 事件总线配置
type EventBusConfig struct {
	Type       string                 `yaml:"type"`
	BufferSize int                    `yaml:"buffer_size"`
	Workers    int                    `yaml:"workers"`
	Config     map[string]interface{} `yaml:"config"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Default  string                          `yaml:"default"`
	Backends map[string]StorageBackendConfig `yaml:"backends"`
}

// StorageBackendConfig 存储后端配置
type StorageBackendConfig map[string]interface{}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Prometheus  PrometheusConfig  `yaml:"prometheus"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

// PrometheusConfig Prometheus配置
type PrometheusConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

// SourcesConfig 数据源配置
type SourcesConfig struct {
	Market     []SourceConfig `yaml:"market"`
	Social     []SourceConfig `yaml:"social"`
	News       []SourceConfig `yaml:"news"`
	Blockchain []SourceConfig `yaml:"blockchain"`
}

// SourceConfig 单个数据源配置
type SourceConfig struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
	Config  string `yaml:"config"`
}

// SourceAppConfig 数据源应用配置
type SourceAppConfig struct {
	App        AppConfig                 `yaml:"app"`
	API        APIConfig                 `yaml:"api"`
	Auth       AuthConfig                `yaml:"auth"`
	Collectors map[string]interface{}    `yaml:"collectors"`
	Processing ProcessingConfig          `yaml:"processing"`
	Storage    map[string]interface{}    `yaml:"storage"`
	Monitoring SourceMonitoringConfig    `yaml:"monitoring"`
	Advanced   AdvancedConfig           `yaml:"advanced"`
}

// AppConfig 应用配置
type AppConfig struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
}

// APIConfig API配置
type APIConfig struct {
	BaseURL      string          `yaml:"base_url"`
	WebsocketURL string          `yaml:"websocket_url"`
	Timeout      int             `yaml:"timeout"`
	RateLimit    RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig 限速配置
type RateLimitConfig struct {
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	EnableBurst       bool `yaml:"enable_burst"`
	BurstSize         int  `yaml:"burst_size"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
}

// ProcessingConfig 数据处理配置
type ProcessingConfig struct {
	Validation     ValidationConfig     `yaml:"validation"`
	Transformation TransformationConfig `yaml:"transformation"`
	Aggregation    AggregationConfig    `yaml:"aggregation"`
}

// ValidationConfig 数据验证配置
type ValidationConfig struct {
	Enabled        bool `yaml:"enabled"`
	CheckSequence  bool `yaml:"check_sequence"`
	CheckTimestamp bool `yaml:"check_timestamp"`
}

// TransformationConfig 数据转换配置
type TransformationConfig struct {
	NormalizeSymbol   bool `yaml:"normalize_symbol"`
	ConvertTimestamp  bool `yaml:"convert_timestamp"`
}

// AggregationConfig 数据聚合配置
type AggregationConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Intervals []string `yaml:"intervals"`
}

// SourceMonitoringConfig 源监控配置
type SourceMonitoringConfig struct {
	Metrics     MetricsConfig      `yaml:"metrics"`
	HealthCheck HealthCheckConfig  `yaml:"health_check"`
	Alerts      []AlertConfig      `yaml:"alerts"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled        bool   `yaml:"enabled"`
	ExportInterval string `yaml:"export_interval"`
}

// AlertConfig 告警配置
type AlertConfig struct {
	Name      string `yaml:"name"`
	Condition string `yaml:"condition"`
	Severity  string `yaml:"severity"`
}

// AdvancedConfig 高级配置
type AdvancedConfig struct {
	Retry   RetryConfig   `yaml:"retry"`
	Cache   CacheConfig   `yaml:"cache"`
	Logging LoggingConfig `yaml:"logging"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int    `yaml:"max_attempts"`
	Backoff     string `yaml:"backoff"`
	BaseDelay   string `yaml:"base_delay"`
	MaxDelay    string `yaml:"max_delay"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled bool   `yaml:"enabled"`
	TTL     string `yaml:"ttl"`
	MaxSize int    `yaml:"max_size"`
}

// LoadConfig 加载主配置文件
func LoadConfig(path string) (*Config, error) {
	// 扩展环境变量
	path = os.ExpandEnv(path)
	
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	
	// 替换环境变量
	content := os.ExpandEnv(string(data))
	
	// 解析YAML
	var config Config
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	
	return &config, nil
}

// LoadSourceConfig 加载数据源配置文件
func LoadSourceConfig(path string) (*SourceAppConfig, error) {
	// 扩展环境变量
	path = os.ExpandEnv(path)
	
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取数据源配置文件失败: %w", err)
	}
	
	// 替换环境变量
	content := os.ExpandEnv(string(data))
	
	// 解析YAML
	var config SourceAppConfig
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("解析数据源配置文件失败: %w", err)
	}
	
	return &config, nil
}

// LoadAllConfigs 加载所有配置
func LoadAllConfigs(mainConfigPath string) (*Config, map[string]*SourceAppConfig, error) {
	// 加载主配置
	mainConfig, err := LoadConfig(mainConfigPath)
	if err != nil {
		return nil, nil, err
	}
	
	// 获取配置文件目录
	configDir := filepath.Dir(mainConfigPath)
	
	// 加载所有数据源配置
	sourceConfigs := make(map[string]*SourceAppConfig)
	
	// 加载市场数据源
	for _, source := range mainConfig.Sources.Market {
		if source.Enabled {
			configPath := filepath.Join(configDir, strings.TrimPrefix(source.Config, "configs/"))
			config, err := LoadSourceConfig(configPath)
			if err != nil {
				return nil, nil, fmt.Errorf("加载%s配置失败: %w", source.Name, err)
			}
			sourceConfigs[source.Name] = config
		}
	}
	
	// 加载社交数据源
	for _, source := range mainConfig.Sources.Social {
		if source.Enabled {
			configPath := filepath.Join(configDir, strings.TrimPrefix(source.Config, "configs/"))
			config, err := LoadSourceConfig(configPath)
			if err != nil {
				return nil, nil, fmt.Errorf("加载%s配置失败: %w", source.Name, err)
			}
			sourceConfigs[source.Name] = config
		}
	}
	
	// 加载新闻数据源
	for _, source := range mainConfig.Sources.News {
		if source.Enabled {
			configPath := filepath.Join(configDir, strings.TrimPrefix(source.Config, "configs/"))
			config, err := LoadSourceConfig(configPath)
			if err != nil {
				return nil, nil, fmt.Errorf("加载%s配置失败: %w", source.Name, err)
			}
			sourceConfigs[source.Name] = config
		}
	}
	
	// 加载区块链数据源
	for _, source := range mainConfig.Sources.Blockchain {
		if source.Enabled {
			configPath := filepath.Join(configDir, strings.TrimPrefix(source.Config, "configs/"))
			config, err := LoadSourceConfig(configPath)
			if err != nil {
				return nil, nil, fmt.Errorf("加载%s配置失败: %w", source.Name, err)
			}
			sourceConfigs[source.Name] = config
		}
	}
	
	return mainConfig, sourceConfigs, nil
}

// GetCollectorConfig 获取采集器配置
func (s *SourceAppConfig) GetCollectorConfig(name string) (map[string]interface{}, bool) {
	config, exists := s.Collectors[name]
	if !exists {
		return nil, false
	}
	
	// 尝试转换为map
	if m, ok := config.(map[string]interface{}); ok {
		return m, true
	}
	
	return nil, false
}