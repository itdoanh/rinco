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

	"github.com/itdoanh/rinco/services/billing-service/internal/handler"
	"github.com/itdoanh/rinco/services/billing-service/internal/payment"
	"github.com/itdoanh/rinco/services/billing-service/internal/repository"
	"github.com/itdoanh/rinco/services/billing-service/internal/webhook"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// Logger
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(log)

	// Database pool
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

	// Payment driver
	var driver payment.Driver
	if os.Getenv("STRIPE_SECRET_KEY") != "" {
		driver = payment.NewStripeDriverFromEnv()
		log.Info("stripe payment driver initialized")
	} else {
		log.Warn("STRIPE_SECRET_KEY not set, running without payment provider")
	}

	// Repository
	repo := repository.New(pool)

	// Handlers
	server := handler.New(repo, driver)

	// Echo setup
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-Tenant-ID"},
	}))

	// Health
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "billing-service"})
	})

	// Tenant-facing routes (require X-Tenant-ID header)
	tenant := e.Group("")
	tenant.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get("X-Tenant-ID") == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "X-Tenant-ID required"})
			}
			return next(c)
		}
	})

	// Subscription routes
	tenant.POST("/subscriptions", server.CreateSubscription)
	tenant.GET("/subscriptions", server.GetSubscription)
	tenant.PUT("/subscriptions", server.UpdateSubscription)
	tenant.DELETE("/subscriptions", server.CancelSubscription)
	tenant.GET("/billing-portal", server.GetBillingPortal)

	// Invoice routes
	tenant.GET("/invoices", server.ListInvoices)
	tenant.GET("/invoices/:id", server.GetInvoice)

	// Usage routes
	tenant.GET("/usage", server.GetUsage)

	// Admin routes (protected by API key)
	admin := e.Group("/admin", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-Admin-API-Key")
			if key != os.Getenv("ADMIN_API_KEY") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(c)
		}
	})
	admin.POST("/subscriptions", server.AdminCreateSubscription)

	// Webhook route
	e.POST("/webhooks/stripe", func(c echo.Context) error {
		webhookSecret := envOr("STRIPE_WEBHOOK_SECRET", "")
		webhookHandler := webhook.NewHandler(repo, driver, webhookSecret)
		return webhookHandler.StripeWebhook(c)
	})

	// Graceful shutdown
	port := envOr("PORT", "8097")
	go func() {
		log.Info("starting billing-service", "port", port)
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
