package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger with JSON output to stdout and level from LOG_LEVEL (debug, info, warn, error). Default is info.
func Init() {
	level := slog.LevelInfo
	if s := strings.TrimSpace(os.Getenv("LOG_LEVEL")); s != "" {
		switch strings.ToLower(s) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}
