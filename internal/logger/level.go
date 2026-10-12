package logger

import "log/slog"

// defaultLevelName 构建时可通过 ldflags 覆盖
// 发布构建: -ldflags="-X 'paradise/internal/logger.defaultLevelName=off'"
var defaultLevelName = "debug"

// DefaultLevel 返回当前日志级别（完全由构建时 ldflags 注入决定）
// off   → 不输出日志，不产生日志文件
// debug → 输出所有日志
// info  → 输出 Info 及以上
// warn  → 输出 Warn 及以上
// error → 仅输出 Error
func DefaultLevel() slog.Level {
	switch defaultLevelName {
	case "off":
		return LevelOff
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
