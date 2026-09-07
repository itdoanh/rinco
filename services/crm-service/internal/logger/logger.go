// Package logger provides logging utilities for CRM service.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init initializes the package-level logger.
func Init(service, env, version string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level, AddSource: env == "development"})
	l := slog.New(h).With(
		slog.String("service", service),
		slog.String("env", env),
		slog.String("version", version),
	)
	slog.SetDefault(l)
	return l
}
