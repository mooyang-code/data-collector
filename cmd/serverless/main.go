package main

import (
	"github.com/mooyang-code/data-collector/internal/bootstrap"
	"github.com/mooyang-code/data-collector/internal/handler"
)

func main() {
	// 创建默认启动器配置
	cfg := bootstrap.DefaultConfig()

	// 创建启动器
	bs := bootstrap.New(cfg)

	// 注册并启动云函数
	// 这里会调用 cloudfunction.Start() 并阻塞等待请求
	// 节点信息将在首次请求时从 functioncontext 动态获取和更新
	handler.RegisterCloudFunction(bs)
}
