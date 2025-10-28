package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Logger 结构化日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	With(args ...interface{}) Logger
	WithContext(ctx context.Context) Logger
}

// Config 日志配置
type Config struct {
	Level  string `json:"level" yaml:"level"`
	Format string `json:"format" yaml:"format"` // json, text
	NodeID string `json:"node_id" yaml:"node_id"`
}

// structuredLogger 结构化日志实现
type structuredLogger struct {
	logger *slog.Logger
	nodeID string
}

// New 创建新的日志实例
func New(cfg Config) Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		"node_id", cfg.NodeID,
		"component", "data-collector",
	)

	return &structuredLogger{
		logger: logger,
		nodeID: cfg.NodeID,
	}
}

// NewDefault 创建默认日志实例
func NewDefault() Logger {
	return New(Config{
		Level:  "info",
		Format: "json",
		NodeID: "unknown",
	})
}

func (l *structuredLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, l.parseArgs(args...)...)
}

func (l *structuredLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, l.parseArgs(args...)...)
}

func (l *structuredLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, l.parseArgs(args...)...)
}

func (l *structuredLogger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, l.parseArgs(args...)...)
}

func (l *structuredLogger) With(args ...interface{}) Logger {
	return &structuredLogger{
		logger: l.logger.With(l.parseArgs(args...)...),
		nodeID: l.nodeID,
	}
}

func (l *structuredLogger) WithContext(ctx context.Context) Logger {
	// 从context中提取请求ID等信息
	attrs := make([]interface{}, 0)
	
	if requestID := ctx.Value("request_id"); requestID != nil {
		attrs = append(attrs, "request_id", requestID)
	}
	
	if traceID := ctx.Value("trace_id"); traceID != nil {
		attrs = append(attrs, "trace_id", traceID)
	}

	attrs = append(attrs, "timestamp", time.Now().UTC())
	
	return l.With(attrs...)
}

// parseArgs 解析日志参数，支持键值对和单个值
func (l *structuredLogger) parseArgs(args ...interface{}) []interface{} {
	result := make([]interface{}, 0, len(args))
	
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			// 键值对
			result = append(result, args[i], args[i+1])
		} else {
			// 单个值，使用默认键
			result = append(result, "value", args[i])
		}
	}
	
	return result
}

// Global 全局日志实例
var Global Logger = NewDefault()

// SetGlobal 设置全局日志实例
func SetGlobal(logger Logger) {
	Global = logger
}

// 便捷函数
func Debug(msg string, args ...interface{}) {
	Global.Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	Global.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	Global.Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	Global.Error(msg, args...)
}

func With(args ...interface{}) Logger {
	return Global.With(args...)
}

func WithContext(ctx context.Context) Logger {
	return Global.WithContext(ctx)
}

// LogError 记录错误信息的便捷函数
func LogError(err error, msg string, args ...interface{}) {
	if err != nil {
		allArgs := append(args, "error", err.Error())
		Global.Error(msg, allArgs...)
	}
}

// LogDuration 记录函数执行时间
func LogDuration(start time.Time, operation string, args ...interface{}) {
	duration := time.Since(start)
	allArgs := append(args, 
		"operation", operation, 
		"duration_ms", duration.Milliseconds(),
	)
	Global.Info("operation completed", allArgs...)
}

// TimedOperation 带计时的操作包装器
func TimedOperation(operation string, fn func() error, args ...interface{}) error {
	start := time.Now()
	err := fn()
	
	duration := time.Since(start)
	allArgs := append(args,
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
	)
	
	if err != nil {
		allArgs = append(allArgs, "error", err.Error())
		Global.Error("operation failed", allArgs...)
	} else {
		Global.Info("operation completed", allArgs...)
	}
	
	return err
}