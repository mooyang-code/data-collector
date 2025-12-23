package executor

import (
	"context"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

// ScheduledExecute 定时执行采集任务（由 TRPC 定时器调用）
// 该函数每分钟整点触发，检查所有任务是否该执行
func ScheduledExecute(c context.Context, _ string) error {
	ctx := trpc.CloneContext(c)
	now := time.Now()

	nodeID, version := config.GetNodeInfo()
	log.WithContextFields(ctx, "func", "ScheduledExecute", "version", version, "nodeID", nodeID)

	if nodeID == "" {
		log.DebugContext(ctx, "NodeID 为空，跳过本次执行")
		return nil
	}

	// 获取本节点的任务配置
	tasks := config.GetTaskInstancesByNode(nodeID)
	if len(tasks) == 0 {
		log.DebugContext(ctx, "没有需要执行的任务")
		return nil
	}

	log.InfoContextf(ctx, "开始执行采集任务，当前时间: %s, 任务数: %d", now.Format("15:04:05"), len(tasks))

	// 收集所有需要执行的任务
	var handlers []func() error
	for _, task := range tasks {
		// 获取采集器
		c, err := collector.GetRegistry().Get(task.DataSource, task.DataType)
		if err != nil {
			log.WarnContextf(ctx, "未找到采集器: source=%s, dataType=%s, taskID=%s",
				task.DataSource, task.DataType, task.TaskID)
			continue
		}

		// 为每个需要执行的 interval 创建一个 handler
		for _, interval := range task.Intervals {
			if !shouldExecute(interval, now) {
				continue
			}

			// 捕获变量，避免闭包问题
			taskCopy := task
			intervalCopy := interval
			collectorCopy := c

			handlers = append(handlers, func() error {
				params := &collector.CollectParams{
					InstType: taskCopy.InstType,
					Symbol:   taskCopy.Symbol,
					Interval: intervalCopy,
				}

				log.InfoContextf(ctx, "执行采集: taskID=%s, source=%s, dataType=%s, symbol=%s, interval=%s",
					taskCopy.TaskID, taskCopy.DataSource, taskCopy.DataType, taskCopy.Symbol, intervalCopy)

				if err := collectorCopy.Collect(ctx, params); err != nil {
					log.ErrorContextf(ctx, "采集失败: taskID=%s, interval=%s, error=%v",
						taskCopy.TaskID, intervalCopy, err)
					// 返回 nil 以便其他任务继续执行
					return nil
				}
				return nil
			})
		}
	}
	if len(handlers) == 0 {
		log.DebugContextf(ctx, "当前时刻没有需要执行的任务")
		return nil
	}

	log.InfoContextf(ctx, "并发执行 %d 个采集任务", len(handlers))

	// 使用 trpc.GoAndWait 并发执行所有采集任务
	_ = trpc.GoAndWait(handlers...)

	log.InfoContextf(ctx, "本轮采集任务执行完成")
	return nil
}

// shouldExecute 判断当前时刻是否应该执行指定周期的任务
// interval: K线周期，如 "1m", "5m", "1h" 等
// now: 当前时间
func shouldExecute(interval string, now time.Time) bool {
	minute := now.Minute()
	hour := now.Hour()

	switch interval {
	case "1m":
		// 每分钟执行
		return true
	case "3m":
		// 每3分钟执行（0, 3, 6, 9, ...）
		return minute%3 == 0
	case "5m":
		// 每5分钟执行（0, 5, 10, 15, ...）
		return minute%5 == 0
	case "15m":
		// 每15分钟执行（0, 15, 30, 45）
		return minute%15 == 0
	case "30m":
		// 每30分钟执行（0, 30）
		return minute%30 == 0
	case "1h":
		// 每小时整点执行
		return minute == 0
	case "2h":
		// 每2小时整点执行
		return minute == 0 && hour%2 == 0
	case "4h":
		// 每4小时整点执行
		return minute == 0 && hour%4 == 0
	case "6h":
		// 每6小时整点执行
		return minute == 0 && hour%6 == 0
	case "12h":
		// 每12小时整点执行（0点、12点）
		return minute == 0 && hour%12 == 0
	case "1d":
		// 每天0点执行
		return minute == 0 && hour == 0
	case "1w":
		// 每周一0点执行
		return minute == 0 && hour == 0 && now.Weekday() == time.Monday
	case "1M":
		// 每月1号0点执行
		return minute == 0 && hour == 0 && now.Day() == 1
	default:
		log.Warnf("未知的时间周期: %s", interval)
		return false
	}
}
