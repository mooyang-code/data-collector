package exchanges

import (
	"github.com/mooyang-code/data-collector/internal/collector"
)

// RegisterAll 注册所有交易所采集器
func RegisterAll() error {
	// 注册Binance采集器
	if err := collector.Register("binance", NewBinanceCollector); err != nil {
		return err
	}
	
	// 注册OKX采集器
	if err := collector.Register("okx", NewOKXCollector); err != nil {
		return err
	}
	
	// 注册Huobi采集器
	if err := collector.Register("huobi", NewHuobiCollector); err != nil {
		return err
	}
	
	return nil
}

// init 包初始化时自动注册所有采集器
func init() {
	if err := RegisterAll(); err != nil {
		panic("failed to register collectors: " + err.Error())
	}
}