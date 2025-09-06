package event

import "context"

// EventBusAdapter 将 event.EventBus 适配到其他包的 EventBus 接口
type EventBusAdapter struct {
	eventBus EventBus
}

// NewEventBusAdapter 创建适配器
func NewEventBusAdapter(eventBus EventBus) *EventBusAdapter {
	return &EventBusAdapter{
		eventBus: eventBus,
	}
}

// Publish 发布事件（接受 interface{} 类型）
func (a *EventBusAdapter) Publish(event interface{}) error {
	ctx := context.Background()
	// 尝试转换为 Event 接口
	if e, ok := event.(Event); ok {
		return a.eventBus.Publish(ctx, e)
	}

	// 如果不是 Event 类型，包装成 BaseEvent
	baseEvent := NewEvent("generic.event", "adapter", event)
	return a.eventBus.Publish(ctx, baseEvent)
}

// PublishAsync 异步发布事件
func (a *EventBusAdapter) PublishAsync(event interface{}) {
	ctx := context.Background()
	// 尝试转换为 Event 接口
	if e, ok := event.(Event); ok {
		a.eventBus.PublishAsync(ctx, e)
		return
	}

	// 如果不是 Event 类型，包装成 BaseEvent
	baseEvent := NewEvent("generic.event", "adapter", event)
	a.eventBus.PublishAsync(ctx, baseEvent)
}