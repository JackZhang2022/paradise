// Package logger 提供基于 slog 的结构化日志系统。
//
// 特性：
//   - 开发模式：JSON 格式输出到文件，同时支持 stderr 降级
//   - 发布模式：通过 LevelOff 完全关闭日志，不产生日志文件
//   - 线程安全：使用 sync.Once 保证单次初始化
//   - 环境变量控制：PARADISE_LOG_LEVEL 可覆盖默认日志级别
//
// 构建时注入：
//
//	发布构建: -ldflags="-X 'paradise/internal/logger.defaultLevelName=off'"
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	once  sync.Once
	lgr   *slog.Logger
	level slog.Level
)

const logFileName = "paradise.log"

// LevelOff 关闭所有日志输出的级别
const LevelOff = slog.Level(99)

// Init 初始化日志系统
// logDir: 日志文件存放目录
// lv: 日志级别，低于此级别的日志不会输出；传 LevelOff 则不产生日志文件
func Init(logDir string, lv slog.Level) {
	once.Do(func() {
		level = lv

		// 发布模式：不创建日志文件，不输出任何日志
		if level >= LevelOff {
			lgr = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: level}))
			return
		}

		path := filepath.Join(logDir, logFileName)

		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			lgr = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
			lgr.Warn("无法打开日志文件，将输出到 stderr", "path", path, "error", err)
			return
		}

		opts := &slog.HandlerOptions{
			Level: level,
		}
		lgr = slog.New(slog.NewJSONHandler(file, opts))
		lgr.Info("日志系统初始化完成", "log_file", path, "level", level.String())
	})
}

func ensure() {
	if lgr == nil {
		level = slog.LevelDebug
		lgr = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	}
}

func Info(msg string, args ...any) {
	ensure()
	lgr.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	ensure()
	lgr.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	ensure()
	lgr.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	ensure()
	lgr.Debug(msg, args...)
}
