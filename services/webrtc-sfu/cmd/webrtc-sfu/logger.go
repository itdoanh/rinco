package main

import (
	"log/slog"
	"os"

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/config"
)

func defaultLogger(_ config.Config) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
