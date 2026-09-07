// Package main — RINCO notification-service entrypoint.
//
// Echo HTTP + Connect-RPC adapter, NATS subscriber for broadcast events,
// graceful shutdown, and embedded SQL migrations applied at boot.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

	"github.com/rinco/services/notification-service/internal/handler"
	mw "github.com/rinco/services/notification-service/internal/middleware"
	"github.com/rinco/services/notification-service/internal/platform"
)

const serviceName = "notification-service"
const version = "1.0.0"

func main() {
	cfg := platform.LoadConfig()
	logger := platform.InitLogger(cfg.Env)
	logger.Info("notification-service starting", slog.String("addr", cfg.HTTPAddr))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, err := platform.InitTracer(rootCtx, cfg.OTLP, cfg.Env)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		defer func() { _ = tp.Shutdown(context.Background()) }()
	}

	pool, err := platform.OpenPool(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	if err := platform.RunMigrations(rootCtx, pool); err != nil {
		logger.Error("migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	rdb := platform.OpenRedis(cfg.ValkeyURL)
	defer func() { _ = rdb.Close() }()
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		logger.Warn("valkey unavailable", slog.String("error", err.Error()))
	}

	nc, err := platform.OpenNATS(cfg.NatsURL)
	if err != nil {
		logger.Warn("nats unavailable", slog.String("error", err.Error()))
	}
	if nc != nil {
		defer nc.Close()
	}

	srv := handler.NewServer(pool, rdb, nc, cfg)

	e := newEcho(cfg, pool, rdb, srv)
	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server", slog.String("error", err.Error()))
		}
	}()

	if nc != nil {
		go subscribeNATS(rootCtx, logger, nc, srv)
	}

	<-rootCtx.Done()
	logger.Info("shutdown requested")
	shut, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(shut); err != nil {
		logger.Error("graceful shutdown", slog.String("error", err.Error()))
	}
	logger.Info("notification-service stopped")
}

func newEcho(cfg *platform.Config, pool *pgxpool.Pool, rdb *redis.Client, srv *handler.Server) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(mw.RecoveryMW())
	e.Use(mw.LoggingMW())
	e.Use(mw.CORSMW())
	e.Use(mw.SecurityHeadersMW())
	e.Use(otelecho.Middleware(serviceName))

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName, "version": version})
	})
	e.GET("/readyz", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_unreachable", "error": err.Error()})
		}
		if rdb != nil {
			if err := rdb.Ping(ctx).Err(); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "valkey_unreachable", "error": err.Error()})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"service": serviceName, "version": version})
	})

	v1 := e.Group("/v1")
	v1.Use(mw.TenantMW())
	v1.Use(mw.AuthMW())
	v1.Use(mw.RateLimitMW(rdb, 200, time.Minute))

	v1.POST("/notifications/send", srv.Send)
	v1.POST("/notifications/broadcast", srv.Broadcast)
	v1.GET("/notifications", srv.List)
	v1.GET("/notifications/:id", srv.Get)
	v1.POST("/notifications/:id/read", srv.MarkRead)
	v1.POST("/preferences/:user_id", srv.SetPreferences)
	v1.GET("/preferences/:user_id", srv.GetPreferences)
	v1.POST("/subscriptions/webpush", srv.RegisterWebPush)
	v1.DELETE("/subscriptions/webpush/:id", srv.DeleteWebPush)
	v1.POST("/subscriptions/fcm", srv.RegisterFCM)
	v1.GET("/stats", srv.Stats)

	// Connect-RPC adapter
	srv.ConnectRPC(e)
	return e
}

// =============================================================================
// NATS subscriber
// =============================================================================

type natsNotifPayload struct {
	TenantID  string            `json:"tenant_id"`
	UserID    string            `json:"user_id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Icon      string            `json:"icon,omitempty"`
	Category  string            `json:"category,omitempty"`
	Priority  string            `json:"priority,omitempty"`
	Channels  []string          `json:"channels,omitempty"`
	Data      map[string]any   `json:"data,omitempty"`
	Email     string            `json:"email,omitempty"`
	Phone     string            `json:"phone,omitempty"`
	ExpiresAt *time.Time       `json:"expires_at,omitempty"`
}

func subscribeNATS(ctx context.Context, logger *slog.Logger, nc *nats.Conn, srv *handler.Server) {
	// Subscribe to system events that should trigger notifications
	subjects := []string{
		"notification.broadcast",
		"user.notification",
		"lead.scored",
		"lead.assigned",
		"task.created",
	}
	for _, subj := range subjects {
		if _, err := nc.QueueSubscribe(subj, "notification-workers", func(m *nats.Msg) {
			var payload natsNotifPayload
			if err := json.Unmarshal(m.Data, &payload); err != nil {
				logger.Warn("nats unmarshal", slog.String("error", err.Error()))
				return
			}
			// Forward to the handler's Send method using a synthetic request
			// We bypass the Echo layer and call the DB/logic directly.
			deliverNATSNotification(ctx, srv, payload)
		}); err != nil {
			logger.Warn("nats subscribe", slog.String("subject", subj), slog.String("error", err.Error()))
		}
	}
}

func deliverNATSNotification(ctx context.Context, srv *handler.Server, payload natsNotifPayload) {
	if payload.UserID == "" || payload.Title == "" {
		return
	}
	priority := payload.Priority
	if priority == "" {
		priority = "normal"
	}
	channels := payload.Channels
	if len(channels) == 0 {
		channels = []string{"in_app"}
	}
	notifID := uuid.NewString()
	data := payload.Data
	if data == nil {
		data = map[string]any{}
	}
	data["notif_id"] = notifID

	pool := srv.Pool()
	if pool == nil {
		return
	}

	_, _ = pool.Exec(ctx, `
		INSERT INTO notification.notifications (id, tenant_id, user_id, type, title, body, icon, category, priority, channels_resolved, data, status, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, 'sent', NOW())`,
		notifID, payload.TenantID, payload.UserID, payload.Type, payload.Title, payload.Body,
		payload.Icon, payload.Category, priority, mustJSON(channels), mustJSON(data))
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }