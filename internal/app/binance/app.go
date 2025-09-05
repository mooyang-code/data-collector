// Package binance 币安交易所应用
package binance

import (
	"github.com/mooyang-code/data-collector/internal/app"
	"trpc.group/trpc-go/trpc-go/log"
)

// BinanceApp 币安应用实现
type BinanceApp struct {
	app.App
}

// NewBinanceApp 创建币安应用
func NewBinanceApp(config *app.AppConfig) (app.App, error) {
	// 创建基础的数据采集器应用
	baseApp, err := app.NewDataCollectorApp(config)
	if err != nil {
		return nil, err
	}

	return &BinanceApp{
		App: baseApp,
	}, nil
}

// init 自动注册币安应用
func init() {
	err := app.RegisterCreator(
		"binance",        // 应用名称
		"币安",           // 显示名称
		"币安数据采集应用", // 描述
		NewBinanceApp,    // 创建函数
	)
	if err != nil {
		log.Errorf("注册币安应用失败: %v", err)
	}
}