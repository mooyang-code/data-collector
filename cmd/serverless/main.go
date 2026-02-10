package main

import (
	"log"
	"os"
	"time"

	"github.com/mooyang-code/scf-framework"
	"github.com/mooyang-code/scf-framework/plugin"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go"
	_ "trpc.group/trpc-go/trpc-log-cls"
)

// loadSupportedCollectors 从 config.yaml 的 plugin.supported_collectors 读取支持的采集器类型列表。
func loadSupportedCollectors(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	var cfg struct {
		Plugin struct {
			SupportedCollectors []string `yaml:"supported_collectors"`
		} `yaml:"plugin"`
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	if len(cfg.Plugin.SupportedCollectors) == 0 {
		log.Fatalf("配置文件中 plugin.supported_collectors 为空")
	}

	return cfg.Plugin.SupportedCollectors
}

func main() {
	collectors := loadSupportedCollectors("./configs/config.yaml")

	p := plugin.NewHTTPPluginAdapter(
		"data-collector",
		"http://127.0.0.1:9001",
		plugin.WithReadyTimeout(60*time.Second),
		plugin.WithHeartbeatExtra(map[string]interface{}{
			"supported_collectors": collectors,
		}),
	)

	app := scf.New(p,
		scf.WithConfigPath("./configs/config.yaml"),
		scf.WithGatewayService("trpc.collector.gateway.stdhttp"),
	)

	if err := app.Run(trpc.BackgroundContext()); err != nil {
		log.Fatalf("data-collector exited: %v", err)
	}
}
