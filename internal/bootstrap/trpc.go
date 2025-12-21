package bootstrap

import (
	"github.com/mooyang-code/data-collector/internal/dnsproxy"
	"github.com/mooyang-code/data-collector/internal/heartbeat"
	"github.com/mooyang-code/data-collector/internal/taskmgr"
	"github.com/mooyang-code/go-commlib/trpc-database/timer"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

// RegisterTRPCServices 注册所有TRPC服务并启动服务
// 包括：心跳定时器服务、任务同步定时器服务、DNS获取定时器服务
func RegisterTRPCServices() error {
	log.Info("正在初始化TRPC服务...")

	// 创建TRPC服务器
	s := trpc.NewServer()

	// 注册心跳定时器
	log.Info("注册心跳定时器...")
	timer.RegisterScheduler("heartbeatSchedule", &timer.DefaultScheduler{})
	timer.RegisterHandlerService(s.Service("trpc.heartbeat.timer"), heartbeat.ScheduledHeartbeat)

	// 注册任务同步定时器
	log.Info("注册任务同步定时器...")
	timer.RegisterScheduler("taskSyncSchedule", &timer.DefaultScheduler{})
	timer.RegisterHandlerService(s.Service("trpc.tasksync.timer"), taskmgr.SyncTasks)

	// 注册 DNS 获取定时器
	log.Info("注册 DNS 获取定时器...")
	timer.RegisterScheduler("dnsFetchSchedule", &timer.DefaultScheduler{})
	timer.RegisterHandlerService(s.Service("trpc.dnsfetch.timer"), dnsproxy.ScheduledFetchDNS)

	// 启动TRPC服务（用go协程包裹）
	go func() {
		log.Info("启动TRPC服务器...")
		if err := s.Serve(); err != nil {
			log.Errorf("TRPC服务器启动失败: %v", err)
		}
	}()
	return nil
}
