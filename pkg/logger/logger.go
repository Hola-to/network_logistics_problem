package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *slog.Logger

// Config конфигурация логгера
type Config struct {
	Level      string
	Format     string // json, text
	Output     string // stdout, stderr, file
	FilePath   string
	MaxSize    int // MB
	MaxBackups int
	MaxAge     int // days
	Compress   bool
}

// Init инициализирует логгер
func Init(level string) {
	InitWithConfig(Config{
		Level:  level,
		Format: "json",
		Output: "stdout",
	})
}

// InitWithConfig инициализирует логгер с полной конфигурацией
func InitWithConfig(cfg Config) {
	var lvl slog.Level
	switch cfg.Level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	// Выбираем writer
	var writer io.Writer
	switch cfg.Output {
	case "stderr":
		writer = os.Stderr
	case "file":
		if cfg.FilePath == "" {
			cfg.FilePath = "logs/app.log"
		}
		// Создаём директорию
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			writer = os.Stdout
		} else {
			// Используем lumberjack для ротации
			writer = &lumberjack.Logger{
				Filename:   cfg.FilePath,
				MaxSize:    cfg.MaxSize,
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge,
				Compress:   cfg.Compress,
			}
		}
	default:
		writer = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: lvl == slog.LevelDebug,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	Log = slog.New(handler)
}

// WithContext добавляет контекстные данные
func WithContext(ctx context.Context, args ...any) *slog.Logger {
	return Log.With(args...)
}

// WithRequestID добавляет request ID
func WithRequestID(requestID string) *slog.Logger {
	return Log.With("request_id", requestID)
}

// WithService добавляет имя сервиса
func WithService(service string) *slog.Logger {
	return Log.With("service", service)
}

// Debug логирует debug сообщение
func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

// Info логирует info сообщение
func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

// Warn логирует warning сообщение
func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

// Error логирует error сообщение
func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

// Fatal логирует fatal сообщение и завершает программу
func Fatal(msg string, args ...any) {
	Log.Error(msg, args...)
	os.Exit(1)
}
