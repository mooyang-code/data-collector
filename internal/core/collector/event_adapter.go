package collector

// AppEventBusAdapter 将 app.EventBus 适配到 collector.EventBus
type AppEventBusAdapter struct {
	publish      func(interface{}) error
	publishAsync func(interface{})
}

// NewAppEventBusAdapter 创建适配器
func NewAppEventBusAdapter(publish func(interface{}) error, publishAsync func(interface{})) EventBus {
	return &AppEventBusAdapter{
		publish:      publish,
		publishAsync: publishAsync,
	}
}

// Publish 发布事件
func (a *AppEventBusAdapter) Publish(event interface{}) error {
	return a.publish(event)
}

// PublishAsync 异步发布事件
func (a *AppEventBusAdapter) PublishAsync(event interface{}) {
	a.publishAsync(event)
}