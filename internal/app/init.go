// Package app 采集器初始化（最好是中文注释！）
package app

import (
	// 导入所有采集器包以触发自注册
	// 这个文件专门用于导入采集器包，避免循环导入
	_ "github.com/mooyang-code/data-collector/internal/app/binance/klines"
	_ "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
	_ "github.com/mooyang-code/data-collector/internal/app/okx/symbols"
	
	// 未来可以在这里添加新的采集器包
	// _ "github.com/mooyang-code/data-collector/internal/app/okx/klines"
	// _ "github.com/mooyang-code/data-collector/internal/app/bybit/klines"
	// _ "github.com/mooyang-code/data-collector/internal/app/bybit/symbols"
)

// InitCollectors 初始化所有采集器
// 这个函数确保所有采集器包被导入并注册
func InitCollectors() {
	// 这个函数的存在确保了上面的导入语句被执行
	// 从而触发各个采集器包的 init() 函数
	// 实际上不需要做任何事情，导入语句已经足够了
}
