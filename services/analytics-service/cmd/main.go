package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/itdoanh/rinco/services/analytics-service/internal/handler"
	"github.com/itdoanh/rinco/services/analytics-service/internal/repository"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	// ClickHouse connection
	chURL := envOr("CLICKHOUSE_URL", "clickhouse://localhost:9000")
	opts := &clickhouse.Options{
		Addr: []string{chURL},
		Auth: clickhouse.Auth{
			Database: envOr("CLICKHOUSE_DATABASE", "analytics"),
			Username: envOr("CLICKHOUSE_USER", "default"),
			Password: envOr("CLICKHOUSE_PASSWORD", ""),
		},
		Debug: envOr("ENV", "production") != "production",
	}

	var conn clickhouse.Conn
	ctx := context.Background()
	c, err := clickhouse.Open(opts)
	if err != nil {
		log.Error("failed to open clickhouse", "err", err)
		os.Exit(1)
	}
	if err := c.Ping(ctx); err != nil {
		log.Warn("clickhouse ping failed (may be unavailable in dev)", "err", err)
	}
	conn = c
	log.Info("connected to clickhouse", "addr", chURL)
	defer c.Close()

	// Repository + handlers
	repo := repository.New(conn)
	server := handler.New(repo)

	// Echo setup
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Health
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "analytics-service"})
	})

	// Tracking (ingestion)
	e.POST("/track", server.TrackEvent)
	e.POST("/track/batch", server.TrackBatch)

	// Analytics queries
	e.GET("/analytics/dashboard", server.GetDashboard)
	e.GET("/analytics/page-views", server.GetPageViews)
	e.GET("/analytics/top-sources", server.GetTopSources)
	e.GET("/analytics/top-pages", server.GetTopPages)
	e.GET("/analytics/trend", server.GetTrend)

	// Admin export
	admin := e.Group("/admin", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-Admin-API-Key")
			if key != os.Getenv("ADMIN_API_KEY") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(c)
		}
	})
	admin.POST("/export", server.ExportEvents)

	// Graceful shutdown
	port := envOr("PORT", "8099")
	go func() {
		log.Info("starting analytics-service", "port", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "err", err)
	}
	log.Info("server stopped")
}
