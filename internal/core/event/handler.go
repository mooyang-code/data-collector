package event

import (
	"log"
)

// Handler 事件处理器接口
type Handler interface {
	Name() string
	Handle(event Event) error
}

// BaseHandler 基础事件处理器
type BaseHandler struct {
	name string
}

func NewBaseHandler(name string) *BaseHandler {
	return &BaseHandler{
		name: name,
	}
}

func (h *BaseHandler) Name() string {
	return h.name
}

// LogHandler 日志处理器
type LogHandler struct {
	BaseHandler
}

func NewLogHandler() *LogHandler {
	return &LogHandler{
		BaseHandler: *NewBaseHandler("log_handler"),
	}
}

func (h *LogHandler) Handle(event Event) error {
	log.Printf("[%s] 事件: %s, 源: %s, 时间: %v, 数据: %v",
		h.Name(),
		event.Type(),
		event.Source(),
		event.Timestamp().Format("2006-01-02 15:04:05"),
		event.Data(),
	)
	return nil
}

// StorageHandler 存储处理器（示例）
type StorageHandler struct {
	BaseHandler
	// storage Storage // 实际应该注入存储接口
}

func NewStorageHandler() *StorageHandler {
	return &StorageHandler{
		BaseHandler: *NewBaseHandler("storage_handler"),
	}
}

func (h *StorageHandler) Handle(event Event) error {
	// 根据事件类型处理存储逻辑
	switch event.Type() {
	case EventKlineCollected:
		log.Printf("[%s] 存储K线数据: %v", h.Name(), event.Data())
		// 实际存储逻辑
	case EventTickerCollected:
		log.Printf("[%s] 存储行情数据: %v", h.Name(), event.Data())
		// 实际存储逻辑
	default:
		log.Printf("[%s] 未知事件类型: %s", h.Name(), event.Type())
	}
	
	return nil
}

// MonitorHandler 监控处理器
type MonitorHandler struct {
	BaseHandler
	metrics map[string]int64
}

func NewMonitorHandler() *MonitorHandler {
	return &MonitorHandler{
		BaseHandler: *NewBaseHandler("monitor_handler"),
		metrics:     make(map[string]int64),
	}
}

func (h *MonitorHandler) Handle(event Event) error {
	// 更新指标
	h.metrics[event.Type()]++
	
	// 检查是否是错误事件
	if _, ok := event.(*ErrorEvent); ok {
		log.Printf("[%s] 错误事件: %v", h.Name(), event.Data())
		// 可以发送告警
	}
	
	return nil
}

// HandlerChain 处理器链
type HandlerChain struct {
	handlers []Handler
}

func NewHandlerChain() *HandlerChain {
	return &HandlerChain{
		handlers: make([]Handler, 0),
	}
}

func (c *HandlerChain) AddHandler(handler Handler) {
	c.handlers = append(c.handlers, handler)
}

func (c *HandlerChain) Handle(event Event) error {
	for _, handler := range c.handlers {
		if err := handler.Handle(event); err != nil {
			log.Printf("处理器 %s 处理失败: %v", handler.Name(), err)
			// 继续执行其他处理器
		}
	}
	return nil
}