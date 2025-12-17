package taskmgr

import (
	"context"

	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go/log"
)

// SyncTasks 框架定时器入口函数 - 定时同步任务配置
// 该函数由 TRPC 定时器框架调用
func SyncTasks(ctx context.Context, _ string) error {
	nodeID, version := config.GetNodeInfo()
	log.WithContextFields(ctx, "func", "SyncTasks", "version", version, "nodeID", nodeID)

	mgr := GetManager()
	if mgr == nil {
		log.ErrorContext(ctx, "TaskManager 未初始化")
		return nil
	}

	if err := mgr.Sync(ctx); err != nil {
		log.ErrorContextf(ctx, "任务同步失败: %v", err)
		return err
	}

	return nil
}
