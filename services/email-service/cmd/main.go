// Package main — RINCO email-service entrypoint.
//
// Echo HTTP + Connect-RPC adapter, NATS subscriber (`email.send`) draining
// the queue into the configured driver, daily tier transition background
// worker, graceful shutdown, and embedded SQL migrations applied at boot.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

	"github.com/rinco/services/email-service/internal/driver"
	"github.com/rinco/services/email-service/internal/handler"
	mw "github.com/rinco/services/email-service/internal/middleware"
	"github.com/rinco/services/email-service/internal/platform"
)

const serviceName = "email-service"
const version = "1.0.0"

func main() {
	cfg := platform.LoadConfig()
	logger := platform.InitLogger(cfg.Env)
	logger.Info("email-service starting", slog.String("addr", cfg.HTTPAddr))

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

	drv, err := driver.New(cfg)
	if err != nil {
		logger.Warn("driver init failed, using console", slog.String("error", err.Error()))
		drv = driver.NewConsole()
	}
	srv := handler.NewServer(pool, rdb, nc, drv, cfg)

	store := newTieredStore(cfg)
	if store != nil {
		if err := store.EnsureBuckets(rootCtx); err != nil {
			logger.Warn("ensure buckets", slog.String("error", err.Error()))
		}
		go tieredWorker(rootCtx, logger, store, pool, time.Duration(cfg.MaxHotAgeDays)*24*time.Hour)
	}

	e := newEcho(cfg, pool, rdb, srv)
	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server", slog.String("error", err.Error()))
		}
	}()

	if nc != nil {
		go subscribeQueue(rootCtx, logger, nc, srv)
	}
	<-rootCtx.Done()
	logger.Info("shutdown requested")
	shut, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(shut); err != nil {
		logger.Error("graceful shutdown", slog.String("error", err.Error()))
	}
	logger.Info("email-service stopped")
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

	v1.POST("/email/send", srv.SendEmail)
	v1.POST("/email/batch", srv.BatchSend)
	v1.GET("/email/templates", srv.ListTemplates)
	v1.POST("/email/templates", srv.CreateTemplate)
	v1.GET("/email/templates/:id", srv.GetTemplate)
	v1.PUT("/email/templates/:id", srv.UpdateTemplate)
	v1.DELETE("/email/templates/:id", srv.DeleteTemplate)
	v1.POST("/email/templates/:id/render", srv.RenderTemplate)
	v1.GET("/email/templates/:id/preview", srv.PreviewTemplate)
	v1.GET("/email/logs", srv.ListLogs)
	v1.GET("/email/logs/:id", srv.GetLog)
	v1.GET("/email/stats", srv.Stats)
	v1.POST("/email/webhooks/:provider", srv.Webhook)

	// Public tracking endpoints (no tenant required)
	e.GET("/e/:msg_id.gif", srv.TrackingPixel)
	e.GET("/e/:msg_id", srv.TrackingPixel)
	e.GET("/c/:msg_id", srv.ClickRedirect)
	e.GET("/u/:msg_id", srv.Unsubscribe)

	// Connect-RPC adapter
	srv.ConnectRPC(e)
	return e
}

// =============================================================================
// Tiered S3 storage
// =============================================================================

type tieredStore struct {
	client *minio.Client
	hot    string
	cold   string
}

func newTieredStore(cfg *platform.Config) *tieredStore {
	if cfg.S3Endpoint == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		return nil
	}
	endpoint := strings.TrimPrefix(strings.TrimPrefix(cfg.S3Endpoint, "https://"), "http://")
	cl, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""), Secure: cfg.S3UseSSL})
	if err != nil {
		slog.Warn("tiered storage disabled", slog.String("error", err.Error()))
		return nil
	}
	return &tieredStore{client: cl, hot: cfg.S3BucketHot, cold: cfg.S3BucketCold}
}

func (t *tieredStore) EnsureBuckets(ctx context.Context) error {
	if t == nil || t.client == nil {
		return nil
	}
	for _, b := range []string{t.hot, t.cold} {
		if b == "" {
			continue
		}
		exists, err := t.client.BucketExists(ctx, b)
		if err != nil {
			return err
		}
		if !exists {
			if err := t.client.MakeBucket(ctx, b, minio.MakeBucketOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *tieredStore) MigrateOld(ctx context.Context, maxAge time.Duration) (int, error) {
	if t == nil || t.cold == "" {
		return 0, nil
	}
	objs := t.client.ListObjects(ctx, t.hot, minio.ListObjectsOptions{Recursive: true})
	migrated := 0
	for obj := range objs {
		if obj.Err != nil {
			return migrated, obj.Err
		}
		if time.Since(obj.LastModified) < maxAge {
			continue
		}
		if _, err := t.client.CopyObject(ctx, minio.CopyDestOptions{Bucket: t.cold, Object: obj.Key}, minio.CopySrcOptions{Bucket: t.hot, Object: obj.Key}); err != nil {
			return migrated, err
		}
		if err := t.client.RemoveObject(ctx, t.hot, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return migrated, err
		}
		migrated++
	}
	return migrated, nil
}

func tieredWorker(ctx context.Context, logger *slog.Logger, store *tieredStore, pool *pgxpool.Pool, maxAge time.Duration) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := store.MigrateOld(ctx, maxAge)
			if err != nil {
				logger.Warn("tiered migration", slog.String("error", err.Error()))
				continue
			}
			if n > 0 {
				logger.Info("tiered migration", slog.Int("migrated", n))
			}
			// Also push the migrated objects into the DB for cross-tier queries.
			if _, err := pool.Exec(ctx, `UPDATE email.email_attachments SET tier = 'cold' WHERE tier = 'hot' AND uploaded_at < NOW() - INTERVAL '1 second' * $1`, int(maxAge.Seconds())); err != nil {
				logger.Warn("tiered bookkeeping", slog.String("error", err.Error()))
			}
		}
	}
}

// =============================================================================
// NATS subscriber (background worker)
// =============================================================================

type queueItem struct {
	TenantID string         `json:"tenant_id"`
	To       []string      `json:"to"`
	Cc       []string      `json:"cc,omitempty"`
	Bcc      []string      `json:"bcc,omitempty"`
	Subject  string        `json:"subject"`
	Body     string        `json:"body"`
	BodyType string        `json:"body_type"`
	From     string        `json:"from,omitempty"`
	ReplyTo  string        `json:"reply_to,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Priority string        `json:"priority"`
	Retry    int           `json:"-"`
}

func subscribeQueue(ctx context.Context, logger *slog.Logger, nc *nats.Conn, srv *handler.Server) {
	if _, err := nc.QueueSubscribe("email.send", "email-workers", func(m *nats.Msg) {
		var item queueItem
		if err := json.Unmarshal(m.Data, &item); err != nil {
			logger.Warn("queue unmarshal", slog.String("error", err.Error()))
			_ = m.Nak()
			return
		}
		msgID := uuid.NewString()
		tenantCtx := platform.WithTenant(ctx, item.TenantID, "", false)

		// Use the driver directly from the server
		drv := srv.Driver()
		priority := item.Priority
		if priority == "" {
			priority = "normal"
		}
		bodyType := item.BodyType
		if bodyType == "" {
			bodyType = "text"
		}
		from := item.From
		if from == "" {
			from = srv.DefaultFrom()
		}
		msg := driver.Message{
			From: from, To: item.To, Cc: item.Cc, Bcc: item.Bcc,
			Subject: item.Subject, Body: item.Body, BodyType: bodyType,
			Headers: item.Headers, ReplyTo: item.ReplyTo,
		}
		res, err := drv.Send(tenantCtx, msg)

		status := "sent"
		errStr := ""
		if err != nil {
			status = "failed"
			errStr = err.Error()
			if item.Retry < 5 {
				scheduleRetry(nc, item, item.Retry+1)
			} else {
				publishDLQ(nc, item, "max-retries-exceeded")
			}
		}

		pool := srv.Pool()
		if pool != nil {
			providerID := ""
			if res.ProviderID != "" {
				providerID = res.ProviderID
			}
			driverName := drv.Name()
			if res.Driver != "" {
				driverName = res.Driver
			}
			_, _ = pool.Exec(tenantCtx, `
				INSERT INTO email.email_logs (tenant_id, msg_id, from_addr, to_addrs, cc, bcc, subject, body_rendered, status, driver, driver_msg_id, priority, sent_at, last_error)
				VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7, $8, $9, $10, $11, $12, $13, $14)`,
				item.TenantID, msgID, from, mustJSON(item.To), mustJSON(item.Cc), mustJSON(item.Bcc),
				item.Subject, item.Body, status, driverName, nullString(providerID), priority, res.SentAt, nullString(errStr))
		}
		_ = m.Ack()
	}); err != nil {
		logger.Warn("nats subscribe", slog.String("error", err.Error()))
	}
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scheduleRetry(nc *nats.Conn, item queueItem, attempt int) {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 12 * time.Hour}
	if attempt < 1 || attempt > len(delays) {
		return
	}
	d := delays[attempt-1]
	item.Retry = attempt
	go func() {
		time.Sleep(d)
		body, _ := json.Marshal(item)
		_ = nc.Publish("email.send", body)
	}()
}

func publishDLQ(nc *nats.Conn, item queueItem, reason string) {
	body, _ := json.Marshal(map[string]any{"item": item, "reason": reason, "ts": time.Now()})
	_ = nc.Publish("email.dlq", body)
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }