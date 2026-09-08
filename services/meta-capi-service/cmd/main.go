package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/itdoanh/rinco/services/meta-capi-service/internal/handler"
	"github.com/itdoanh/rinco/services/meta-capi-service/internal/repository"
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

	// PostgreSQL pool
	dbURL := envOr("DATABASE_URL", "postgres://rinco:rinco_dev_password@localhost:5432/rinco")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Warn("database ping failed (may be unavailable in dev)", "err", err)
	}
	log.Info("connected to postgres")

	// Repository + handlers
	repo := repository.New(pool)
	server := handler.New(repo)

	// Echo setup
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-Tenant-ID"},
	}))

	// Health
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "meta-capi-service"})
	})

	// Tenant routes (require X-Tenant-ID)
	tenant := e.Group("")
	tenant.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get("X-Tenant-ID") == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "X-Tenant-ID required"})
			}
			return next(c)
		}
	})

	// CAPI endpoints
	tenant.GET("/capi/status", server.GetCAPIStatus)
	tenant.POST("/capi/events", server.SendEvent)

	// CRM bridge (receives events from CRM service)
	e.POST("/bridge/crm", server.CRMBridge)

	// Admin routes
	admin := e.Group("/admin", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-Admin-API-Key")
			if key != os.Getenv("ADMIN_API_KEY") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(c)
		}
	})
	admin.POST("/capi/setup", server.SetupCAPI)

	// Graceful shutdown
	port := envOr("PORT", "8100")
	go func() {
		log.Info("starting meta-capi-service", "port", port)
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
