// Package okx 欧易交易所应用
package okx

import (
	"github.com/mooyang-code/data-collector/internal/app"
	"trpc.group/trpc-go/trpc-go/log"
)

// OKXApp 欧易应用实现
type OKXApp struct {
	app.App
}

// NewOKXApp 创建欧易应用
func NewOKXApp(config *app.AppConfig) (app.App, error) {
	// 创建基础的数据采集器应用
	baseApp, err := app.NewDataCollectorApp(config)
	if err != nil {
		return nil, err
	}

	return &OKXApp{
		App: baseApp,
	}, nil
}

// init 自动注册欧易应用
func init() {
	err := app.RegisterCreator(
		"okx",            // 应用名称
		"欧易",           // 显示名称
		"欧易数据采集应用", // 描述
		NewOKXApp,        // 创建函数
	)
	if err != nil {
		log.Errorf("注册欧易应用失败: %v", err)
	}
}