package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/internal/reporter"
	"github.com/mooyang-code/data-collector/pkg/config"
	"github.com/mooyang-code/data-collector/pkg/model"
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
					// 异步上报失败状态
					reporter.ReportTaskStatusAsync(ctx, taskCopy.TaskID, reporter.StatusFailed, err.Error())
					// 返回 nil 以便其他任务继续执行
					return nil
				}

				// 异步上报成功状态
				reporter.ReportTaskStatusAsync(ctx, taskCopy.TaskID, reporter.StatusSuccess, "")
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

// ExecuteTaskImmediately 立即执行任务（服务端触发的任务转移）
// 用于任务失败后，服务端将任务转移到其他节点立即执行
// 注意：客户端在上报失败前已经进行了多次重试，这里直接执行即可
func ExecuteTaskImmediately(ctx context.Context, taskEvent *model.TaskExecuteEvent) (string, error) {
	if taskEvent == nil {
		return "", fmt.Errorf("taskEvent is nil")
	}

	log.InfoContextf(ctx, "[ExecuteTaskImmediately] Starting immediate execution: taskID=%s, symbol=%s",
		taskEvent.TaskID, taskEvent.Symbol)

	// 1. 获取采集器
	c, err := collector.GetRegistry().Get(taskEvent.DataSource, taskEvent.DataType)
	if err != nil {
		errMsg := fmt.Sprintf("采集器未找到: source=%s, dataType=%s", taskEvent.DataSource, taskEvent.DataType)
		log.ErrorContextf(ctx, "[ExecuteTaskImmediately] %s", errMsg)
		// 异步上报失败状态
		reporter.ReportTaskStatusAsync(ctx, taskEvent.TaskID, reporter.StatusFailed, errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// 2. 执行所有 interval 的采集任务
	var handlers []func() error
	var hasError bool
	var lastError string

	for _, interval := range taskEvent.Intervals {
		// 捕获变量，避免闭包问题
		intervalCopy := interval
		collectorCopy := c
		taskEventCopy := taskEvent

		handlers = append(handlers, func() error {
			params := &collector.CollectParams{
				InstType: taskEventCopy.InstType,
				Symbol:   taskEventCopy.Symbol,
				Interval: intervalCopy,
			}

			log.InfoContextf(ctx, "[ExecuteTaskImmediately] 执行采集: taskID=%s, source=%s, dataType=%s, symbol=%s, interval=%s",
				taskEventCopy.TaskID, taskEventCopy.DataSource, taskEventCopy.DataType, taskEventCopy.Symbol, intervalCopy)

			if err := collectorCopy.Collect(ctx, params); err != nil {
				log.ErrorContextf(ctx, "[ExecuteTaskImmediately] 采集失败: taskID=%s, interval=%s, error=%v",
					taskEventCopy.TaskID, intervalCopy, err)
				hasError = true
				lastError = err.Error()
				return err
			}

			log.InfoContextf(ctx, "[ExecuteTaskImmediately] 采集成功: taskID=%s, interval=%s",
				taskEventCopy.TaskID, intervalCopy)
			return nil
		})
	}

	if len(handlers) == 0 {
		errMsg := "没有需要执行的interval"
		log.WarnContextf(ctx, "[ExecuteTaskImmediately] %s", errMsg)
		reporter.ReportTaskStatusAsync(ctx, taskEvent.TaskID, reporter.StatusFailed, errMsg)
		return "", fmt.Errorf(errMsg)
	}

	log.InfoContextf(ctx, "[ExecuteTaskImmediately] 并发执行 %d 个采集任务", len(handlers))

	// 3. 并发执行所有采集任务
	_ = trpc.GoAndWait(handlers...)

	// 4. 根据执行结果上报状态
	var resultMsg string
	var status int

	if hasError {
		status = reporter.StatusFailed
		resultMsg = fmt.Sprintf("部分或全部任务执行失败, lastError=%s", lastError)
	} else {
		status = reporter.StatusSuccess
		resultMsg = "所有任务执行成功"
	}

	log.InfoContextf(ctx, "[ExecuteTaskImmediately] 任务执行完成: taskID=%s, status=%d, result=%s",
		taskEvent.TaskID, status, resultMsg)

	// 异步上报任务状态
	reporter.ReportTaskStatusAsync(ctx, taskEvent.TaskID, status, resultMsg)

	return resultMsg, nil
}
