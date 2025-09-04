// Package ws 提供WebSocket连接管理功能
package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"trpc.group/trpc-go/trpc-go/log"
)

// ConnectionState 连接状态
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateClosed
)

// String 返回状态字符串
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Config WebSocket连接配置
type Config struct {
	URL                string            `yaml:"url"`                // WebSocket URL
	Headers            map[string]string `yaml:"headers"`            // 请求头
	HandshakeTimeout   time.Duration     `yaml:"handshakeTimeout"`   // 握手超时
	ReadTimeout        time.Duration     `yaml:"readTimeout"`        // 读取超时
	WriteTimeout       time.Duration     `yaml:"writeTimeout"`       // 写入超时
	PingInterval       time.Duration     `yaml:"pingInterval"`       // Ping间隔
	PongTimeout        time.Duration     `yaml:"pongTimeout"`        // Pong超时
	ReconnectInterval  time.Duration     `yaml:"reconnectInterval"`  // 重连间隔
	MaxReconnectTries  int               `yaml:"maxReconnectTries"`  // 最大重连次数
	EnableCompression  bool              `yaml:"enableCompression"`  // 启用压缩
	ReadBufferSize     int               `yaml:"readBufferSize"`     // 读缓冲区大小
	WriteBufferSize    int               `yaml:"writeBufferSize"`    // 写缓冲区大小
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		HandshakeTimeout:  10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		PingInterval:      30 * time.Second,
		PongTimeout:       10 * time.Second,
		ReconnectInterval: 5 * time.Second,
		MaxReconnectTries: 10,
		EnableCompression: true,
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
	}
}

// EventType 事件类型
type EventType string

const (
	EventConnected    EventType = "connected"
	EventDisconnected EventType = "disconnected"
	EventMessage      EventType = "message"
	EventError        EventType = "error"
	EventReconnecting EventType = "reconnecting"
)

// Event WebSocket事件
type Event struct {
	Type      EventType   `json:"type"`      // 事件类型
	Data      interface{} `json:"data"`      // 事件数据
	Error     error       `json:"error"`     // 错误信息
	Timestamp time.Time   `json:"timestamp"` // 事件时间
}

// MessageHandler 消息处理器
type MessageHandler func(messageType int, data []byte) error

// EventHandler 事件处理器
type EventHandler func(event *Event)

// Manager WebSocket连接管理器
type Manager struct {
	config  *Config
	dialer  *websocket.Dialer
	
	// 连接状态
	mu    sync.RWMutex
	conn  *websocket.Conn
	state ConnectionState
	
	// 控制通道
	ctx       context.Context
	cancel    context.CancelFunc
	closeCh   chan struct{}
	reconnectCh chan struct{}
	
	// 处理器
	messageHandler MessageHandler
	eventHandlers  []EventHandler
	
	// 统计信息
	stats *Stats
	
	// 重连控制
	reconnectTries int
	lastError      error
}

// Stats 统计信息
type Stats struct {
	ConnectedAt      time.Time `json:"connectedAt"`      // 连接时间
	DisconnectedAt   time.Time `json:"disconnectedAt"`   // 断开时间
	MessagesSent     int64     `json:"messagesSent"`     // 发送消息数
	MessagesReceived int64     `json:"messagesReceived"` // 接收消息数
	ReconnectCount   int       `json:"reconnectCount"`   // 重连次数
	ErrorCount       int64     `json:"errorCount"`       // 错误次数
	LastError        string    `json:"lastError"`        // 最后错误
}

// NewManager 创建WebSocket管理器
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	dialer := &websocket.Dialer{
		HandshakeTimeout: config.HandshakeTimeout,
		ReadBufferSize:   config.ReadBufferSize,
		WriteBufferSize:  config.WriteBufferSize,
		EnableCompression: config.EnableCompression,
	}
	
	return &Manager{
		config:      config,
		dialer:      dialer,
		ctx:         ctx,
		cancel:      cancel,
		closeCh:     make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
		state:       StateDisconnected,
		stats:       &Stats{},
	}
}

// SetMessageHandler 设置消息处理器
func (m *Manager) SetMessageHandler(handler MessageHandler) {
	m.messageHandler = handler
}

// AddEventHandler 添加事件处理器
func (m *Manager) AddEventHandler(handler EventHandler) {
	m.eventHandlers = append(m.eventHandlers, handler)
}

// Connect 连接WebSocket
func (m *Manager) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.state == StateConnected || m.state == StateConnecting {
		return fmt.Errorf("already connected or connecting")
	}
	
	m.setState(StateConnecting)
	
	// 设置请求头
	headers := http.Header{}
	for key, value := range m.config.Headers {
		headers.Set(key, value)
	}
	
	// 建立连接
	conn, _, err := m.dialer.Dial(m.config.URL, headers)
	if err != nil {
		m.setState(StateDisconnected)
		m.recordError(err)
		return fmt.Errorf("dial failed: %w", err)
	}
	
	m.conn = conn
	m.setState(StateConnected)
	m.stats.ConnectedAt = time.Now()
	m.reconnectTries = 0
	
	// 启动消息处理协程
	go m.readLoop()
	go m.pingLoop()
	go m.reconnectLoop()
	
	m.publishEvent(&Event{
		Type:      EventConnected,
		Timestamp: time.Now(),
	})

	log.Infof("WebSocket connected - url: %s", m.config.URL)

	return nil
}

// Disconnect 断开连接
func (m *Manager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.state == StateDisconnected || m.state == StateClosed {
		return nil
	}
	
	m.setState(StateClosed)
	
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	
	m.stats.DisconnectedAt = time.Now()
	
	m.publishEvent(&Event{
		Type:      EventDisconnected,
		Timestamp: time.Now(),
	})

	log.Info("WebSocket disconnected")

	return nil
}

// Close 关闭管理器
func (m *Manager) Close() error {
	m.cancel()
	close(m.closeCh)
	return m.Disconnect()
}

// SendMessage 发送消息
func (m *Manager) SendMessage(messageType int, data []byte) error {
	m.mu.RLock()
	conn := m.conn
	state := m.state
	m.mu.RUnlock()
	
	if state != StateConnected || conn == nil {
		return fmt.Errorf("not connected")
	}
	
	if m.config.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(m.config.WriteTimeout))
	}
	
	if err := conn.WriteMessage(messageType, data); err != nil {
		m.recordError(err)
		m.triggerReconnect()
		return fmt.Errorf("write message failed: %w", err)
	}
	
	m.stats.MessagesSent++
	
	return nil
}

// SendJSON 发送JSON消息
func (m *Manager) SendJSON(v interface{}) error {
	m.mu.RLock()
	conn := m.conn
	state := m.state
	m.mu.RUnlock()
	
	if state != StateConnected || conn == nil {
		return fmt.Errorf("not connected")
	}
	
	if m.config.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(m.config.WriteTimeout))
	}
	
	if err := conn.WriteJSON(v); err != nil {
		m.recordError(err)
		m.triggerReconnect()
		return fmt.Errorf("write json failed: %w", err)
	}
	
	m.stats.MessagesSent++
	
	return nil
}

// GetState 获取连接状态
func (m *Manager) GetState() ConnectionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// GetStats 获取统计信息
func (m *Manager) GetStats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 返回副本
	stats := *m.stats
	return &stats
}

// IsConnected 检查是否已连接
func (m *Manager) IsConnected() bool {
	return m.GetState() == StateConnected
}

// readLoop 读取消息循环
func (m *Manager) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Read loop panic: %v", r)
		}
	}()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.closeCh:
			return
		default:
			m.mu.RLock()
			conn := m.conn
			state := m.state
			m.mu.RUnlock()
			
			if state != StateConnected || conn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			if m.config.ReadTimeout > 0 {
				conn.SetReadDeadline(time.Now().Add(m.config.ReadTimeout))
			}
			
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				m.recordError(err)
				m.triggerReconnect()
				continue
			}
			
			m.stats.MessagesReceived++
			
			// 处理消息
			if m.messageHandler != nil {
				if err := m.messageHandler(messageType, data); err != nil {
					log.Errorf("Message handler error: %v", err)
				}
			}
			
			// 发布消息事件
			m.publishEvent(&Event{
				Type:      EventMessage,
				Data:      data,
				Timestamp: time.Now(),
			})
		}
	}
}

// pingLoop Ping循环
func (m *Manager) pingLoop() {
	if m.config.PingInterval <= 0 {
		return
	}
	
	ticker := time.NewTicker(m.config.PingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.closeCh:
			return
		case <-ticker.C:
			m.mu.RLock()
			conn := m.conn
			state := m.state
			m.mu.RUnlock()
			
			if state != StateConnected || conn == nil {
				continue
			}
			
			if m.config.WriteTimeout > 0 {
				conn.SetWriteDeadline(time.Now().Add(m.config.WriteTimeout))
			}
			
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				m.recordError(err)
				m.triggerReconnect()
			}
		}
	}
}

// reconnectLoop 重连循环
func (m *Manager) reconnectLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.closeCh:
			return
		case <-m.reconnectCh:
			m.handleReconnect()
		}
	}
}

// handleReconnect 处理重连
func (m *Manager) handleReconnect() {
	m.mu.Lock()
	if m.state == StateClosed {
		m.mu.Unlock()
		return
	}
	
	if m.reconnectTries >= m.config.MaxReconnectTries {
		log.Error("Max reconnect tries exceeded")
		m.setState(StateDisconnected)
		m.mu.Unlock()
		return
	}
	
	m.setState(StateReconnecting)
	m.reconnectTries++
	m.stats.ReconnectCount++
	m.mu.Unlock()
	
	m.publishEvent(&Event{
		Type:      EventReconnecting,
		Timestamp: time.Now(),
	})

	log.Infof("Reconnecting WebSocket - tries: %d", m.reconnectTries)
	
	// 等待重连间隔
	time.Sleep(m.config.ReconnectInterval)
	
	// 关闭旧连接
	m.mu.Lock()
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	m.mu.Unlock()
	
	// 尝试重连
	if err := m.Connect(); err != nil {
		log.Errorf("Reconnect failed: %v", err)
		// 继续重连
		m.triggerReconnect()
	}
}

// setState 设置状态
func (m *Manager) setState(state ConnectionState) {
	m.state = state
}

// recordError 记录错误
func (m *Manager) recordError(err error) {
	m.stats.ErrorCount++
	m.stats.LastError = err.Error()
	m.lastError = err
	
	m.publishEvent(&Event{
		Type:      EventError,
		Error:     err,
		Timestamp: time.Now(),
	})
}

// triggerReconnect 触发重连
func (m *Manager) triggerReconnect() {
	select {
	case m.reconnectCh <- struct{}{}:
	default:
		// 重连已在进行中
	}
}

// publishEvent 发布事件
func (m *Manager) publishEvent(event *Event) {
	for _, handler := range m.eventHandlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Event handler panic: %v", r)
				}
			}()
			h(event)
		}(handler)
	}
}
