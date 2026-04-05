package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the global structured JSON logger.
// level: "debug", "info", "warn", "error" (default: "info").
// Returns the configured logger instance.
func Setup(level string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	log := slog.New(handler)
	slog.SetDefault(log)
	return log
}
