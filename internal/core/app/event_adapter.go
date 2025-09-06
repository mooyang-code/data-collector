package app

import (
	"context"
	"time"

	"github.com/mooyang-code/data-collector/internal/core/event"
)

// EventBusAdapter 将 event.EventBus 适配到 app.EventBus
type EventBusAdapter struct {
	eventBus event.EventBus
}

// NewEventBusAdapter 创建适配器
func NewEventBusAdapter(eventBus event.EventBus) EventBus {
	return &EventBusAdapter{eventBus: eventBus}
}

// Publish 实现 app.EventBus 接口
func (a *EventBusAdapter) Publish(event Event) error {
	// 将 app.Event 转换为 event.Event
	ctx := context.Background()
	return a.eventBus.Publish(ctx, &eventAdapter{appEvent: event})
}

// Subscribe 实现 app.EventBus 接口
func (a *EventBusAdapter) Subscribe(pattern string, handler func(Event) error) error {
	// 将处理器适配
	return a.eventBus.Subscribe(pattern, func(ctx context.Context, e event.Event) error {
		// 将 event.Event 转换为 app.Event
		appEvent := &appEventAdapter{e: e}
		return handler(appEvent)
	})
}

// appEventAdapter 将 event.Event 适配到 app.Event
type appEventAdapter struct {
	e event.Event
}

func (a *appEventAdapter) ID() string           { return a.e.ID() }
func (a *appEventAdapter) Type() string         { return a.e.Type() }
func (a *appEventAdapter) Source() string       { return a.e.Source() }
func (a *appEventAdapter) Timestamp() time.Time { return a.e.Timestamp() }
func (a *appEventAdapter) Data() interface{}    { return a.e.Data() }

// eventAdapter 将 app.Event 适配到 event.Event
type eventAdapter struct {
	appEvent Event
}

func (e *eventAdapter) ID() string           { return e.appEvent.ID() }
func (e *eventAdapter) Type() string         { return e.appEvent.Type() }
func (e *eventAdapter) Source() string       { return e.appEvent.Source() }
func (e *eventAdapter) Timestamp() time.Time { return e.appEvent.Timestamp() }
func (e *eventAdapter) Data() interface{}    { return e.appEvent.Data() }
