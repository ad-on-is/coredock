package internal

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

type Logger struct {
	*slog.Logger
}

func New(logger *slog.Logger) *Logger {
	return &Logger{Logger: logger}
}

func (l *Logger) Debugf(format string, args ...any) {
	l.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

var logger = InitLogger()

func InitLogger() *Logger {
	level := os.Getenv("COREDOCK_DEBUG_LEVEL")
	l := slog.LevelInfo
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	}

	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      l,
		TimeFormat: time.RFC3339,
	}))
	return New(logger)
}
