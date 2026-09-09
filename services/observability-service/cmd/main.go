// Package main — RINCO observability-service entrypoint.
//
// Echo HTTP + Connect-RPC adapter wrapping four upstream observability
// backends (Loki, Prometheus, Jaeger/Tempo, ClickHouse).  Postgres is
// the system of record for alerts and audit; ClickHouse is a
// best-effort rollup.  Embedded SQL migrations are applied at boot.
//
// Endpoints:
//
//	GET  /v1/logs
//	GET  /v1/logs/aggregate
//	GET  /v1/traces/:trace_id
//	GET  /v1/traces
//	GET  /v1/metrics
//	GET  /v1/metrics/range
//	GET  /v1/services
//	GET  /v1/services/:service/health
//	GET  /v1/alerts/active
//	GET  /v1/alerts
//	POST /v1/alerts/:id/ack
//	GET  /v1/audit/logs
//	POST /v1/webhook/alertmanager
//	/healthz, /readyz, /metrics
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

	"github.com/rinco/services/observability-service/internal/clickhouse"
	"github.com/rinco/services/observability-service/internal/handler"
	mw "github.com/rinco/services/observability-service/internal/middleware"
	"github.com/rinco/services/observability-service/internal/platform"
)

const serviceName = "observability-service"
const version = "1.0.0"

// chWriter is the contract handler.Server expects for CH writes.  We
// declare it here so the handler package does not depend on the
// clickhouse-go driver package.
type chWriter interface {
	Insert(ctx context.Context, rec clickhouse.AuditRecord) error
}

func main() {
	cfg := platform.LoadConfig()
	logger := platform.InitLogger(cfg.Env)
	logger.Info("observability-service starting",
		slog.String("addr", cfg.HTTPAddr),
		slog.String("env", cfg.Env))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, err := platform.InitTracer(rootCtx, cfg.OTLP, cfg.Env)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		defer func() { _ = tp.Shutdown(context.Background()) }()
	}

	pool, err := platform.OpenPool(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Warn("db unavailable, continuing with degraded mode",
			slog.String("error", err.Error()))
	} else {
		defer pool.Close()
		if err := platform.RunMigrations(rootCtx, pool); err != nil {
			logger.Error("migrations failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	// ClickHouse: best-effort via HTTP gateway.  When unreachable the
	// service still serves traffic and writes audit rows only to
	// Postgres.
	var chW *clickhouse.Writer
	if cfg.ClickHouseURL != "" {
		ddlCtx, ddlCancel := context.WithTimeout(rootCtx, 5*time.Second)
		if err := platform.EnsureClickHouseDDL(ddlCtx, cfg.ClickHouseURL); err != nil {
			logger.Warn("clickhouse ddl failed", slog.String("error", err.Error()))
		}
		ddlCancel()
		chW = clickhouse.NewWriter(cfg.ClickHouseURL)
	}

	fanout := os.Getenv("OBSERVABILITY_FANOUT_WEBHOOK")
	srv := handler.NewServer(pool, cfg.LokiURL, cfg.PrometheusURL, cfg.JaegerURL, cfg.TempoURL,
		fanout, chW)

	e := newEcho(cfg, srv, pool)
	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server", slog.String("error", err.Error()))
		}
	}()
	<-rootCtx.Done()
	logger.Info("shutdown requested")
	shut, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(shut); err != nil {
		logger.Error("graceful shutdown", slog.String("error", err.Error()))
	}
	logger.Info("observability-service stopped")
}

func newEcho(cfg *platform.Config, srv *handler.Server, pool *pgxpool.Pool) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(mw.RecoveryMW())
	e.Use(mw.LoggingMW())
	e.Use(mw.CORSMW())
	e.Use(mw.SecurityHeadersMW())
	e.Use(otelecho.Middleware(serviceName))

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":   "ok",
			"service":  serviceName,
			"version":  version,
		})
	})
	e.GET("/readyz", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if pool != nil {
			if err := pool.Ping(ctx); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{
					"status": "db_unreachable",
					"error":  err.Error(),
				})
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
	v1.Use(mw.RateLimitMW(cfg.RateLimitPerMin))

	v1.GET("/logs", srv.QueryLogs)
	v1.GET("/logs/aggregate", srv.AggregateLogs)
	v1.GET("/traces/:trace_id", srv.GetTrace)
	v1.GET("/traces", srv.SearchTraces)
	v1.GET("/metrics", srv.QueryMetrics)
	v1.GET("/metrics/range", srv.RangeMetrics)
	v1.GET("/services", srv.ListServices)
	v1.GET("/services/:service/health", srv.ServiceHealth)
	v1.GET("/alerts/active", srv.ActiveAlerts)
	v1.GET("/alerts", srv.ListAlerts)
	v1.POST("/alerts/:id/ack", srv.AckAlert)
	v1.GET("/audit/logs", srv.QueryAuditLogs)
	v1.POST("/webhook/alertmanager", srv.WebhookAlertManager)

	srv.ConnectRPC(e)
	return e
}

// =============================================================================
// helpers
// =============================================================================