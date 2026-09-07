// Package main — RINCO landing-service entrypoint.
//
// Self-contained landing service handling dynamic page rendering, form
// submission, server-side tracking pixels, click redirectors, and
// Facebook Conversions API (CAPI) delivery.  Tiered S3 storage with
// auto-transition between hot SSD bucket and cold HDD bucket based on
// object age.  Multi-tenant via X-Tenant-ID / X-User-ID headers; row
// level security expected at the database level.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOpts "go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

	"github.com/rinco/services/landing-service/internal/handler"
	"github.com/rinco/services/landing-service/internal/middleware"
	"github.com/rinco/services/landing-service/internal/platform"
	"github.com/rinco/services/landing-service/internal/storage"
)

const serviceName = "landing-service"
const version = "1.0.0"

type config struct {
	Env, HTTPAddr, DatabaseURL, MongoURL, S3Endpoint, S3AccessKey, S3SecretKey string
	HotBucket, ColdBucket, ValkeyURL, NatsURL string
	FBPixelID, FBAppSecret, TrackingSalt, S3UseSSL string
}

func loadConfig() config {
	return config{
		Env:          platform.Getenv("ENV", "development"),
		HTTPAddr:     platform.Getenv("LANDING_HTTP_ADDR", ":8086"),
		DatabaseURL:  os.Getenv("LANDING_DATABASE_URL"),
		MongoURL:     platform.Getenv("LANDING_MONGO_URL", ""),
		S3Endpoint:   os.Getenv("LANDING_S3_ENDPOINT"),
		S3AccessKey:  os.Getenv("LANDING_S3_ACCESS_KEY"),
		S3SecretKey:  os.Getenv("LANDING_S3_SECRET_KEY"),
		HotBucket:    platform.Getenv("LANDING_S3_BUCKET_HOT", "rinco-hot-ssd"),
		ColdBucket:   platform.Getenv("LANDING_S3_BUCKET_COLD", "rinco-cold-hdd"),
		S3UseSSL:     platform.Getenv("LANDING_S3_SSL", "false"),
		ValkeyURL:    platform.Getenv("LANDING_VALKEY_URL", "localhost:6379"),
		NatsURL:      platform.Getenv("LANDING_NATS_URL", ""),
		FBPixelID:    os.Getenv("LANDING_FB_PIXEL_ID"),
		FBAppSecret:  os.Getenv("LANDING_FB_APP_SECRET"),
		TrackingSalt: platform.Getenv("LANDING_TRACKING_SALT", "rinco-tracking-salt"),
	}
}

func main() {
	cfg := loadConfig()
	logger := platform.InitLogger(cfg.Env, version)
	logger.Info("landing-service starting", slog.String("addr", cfg.HTTPAddr))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, err := platform.InitTracer(rootCtx, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), cfg.Env)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		defer func() { _ = tp.Shutdown(context.Background()) }()
	}

	pool, err := platform.OpenPool(rootCtx, cfg.DatabaseURL)
	if err != nil { logger.Error("db connect failed", slog.String("error", err.Error())); os.Exit(1) }
	defer pool.Close()
	if err := platform.RunMigrations(rootCtx, pool); err != nil { logger.Error("migrations failed", slog.String("error", err.Error())); os.Exit(1) }

	var mongoClient *mongo.Client
	if cfg.MongoURL != "" {
		mongoClient, err = mongo.Connect(rootCtx, mongoOpts.Client().ApplyURI(cfg.MongoURL))
		if err != nil { logger.Warn("mongo connect failed", slog.String("error", err.Error())) } else { defer func() { _ = mongoClient.Disconnect(context.Background()) }() }
	}

	valkey := redis.NewClient(&redis.Options{Addr: cfg.ValkeyURL})
	defer valkey.Close()
	if err := valkey.Ping(rootCtx).Err(); err != nil { logger.Warn("valkey unavailable", slog.String("error", err.Error())) }

	store, err := storage.New(storage.Config{Endpoint: cfg.S3Endpoint, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, HotBucket: cfg.HotBucket, ColdBucket: cfg.ColdBucket, UseSSL: strings.EqualFold(cfg.S3UseSSL, "true"), MaxHotAge: 30 * 24 * time.Hour})
	if err != nil { logger.Warn("tiered storage disabled", slog.String("error", err.Error())) }
	if store != nil { _ = store.EnsureBuckets(rootCtx) }

	srv := handler.New(pool, mongoClient, store, cfg.FBPixelID, cfg.FBAppSecret, cfg.TrackingSalt)
	_ = srv.MongoInit(rootCtx)

	e := newEcho(cfg, srv, pool, valkey)
	go func() { if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) { logger.Error("http server", slog.String("error", err.Error())) } }()
	<-rootCtx.Done()
	logger.Info("shutdown requested")
	shut, cancel := context.WithTimeout(context.Background(), 30*time.Second); defer cancel()
	if err := e.Shutdown(shut); err != nil { logger.Error("graceful shutdown", slog.String("error", err.Error())) }
	logger.Info("landing-service stopped")
}

func newEcho(cfg config, srv *handler.Server, pool *pgxpool.Pool, valkey *redis.Client) *echo.Echo {
	e := echo.New(); e.HideBanner = true; e.HidePort = true
	e.Use(middleware.Recovery()); e.Use(middleware.RequestLog()); e.Use(middleware.CORS()); e.Use(middleware.SecurityHeaders()); e.Use(otelecho.Middleware(serviceName))

	e.GET("/healthz", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName, "version": version}) })
	e.GET("/readyz", func(c echo.Context) error { ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second); defer cancel(); if err := pool.Ping(ctx); err != nil { return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_unreachable", "error": err.Error()}) }; if err := valkey.Ping(ctx).Err(); err != nil { return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "valkey_unreachable", "error": err.Error()}) }; return c.JSON(http.StatusOK, map[string]string{"status": "ready"}) })
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/version", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"service": serviceName, "version": version}) })

	v1 := e.Group("/v1")
	v1.GET("/pages/:tenant_slug/:page_slug", srv.RenderPage)
	v1.GET("/pages/:tenant_slug/index", srv.RenderPage)
	v1.GET("/preview/:page_id", srv.PreviewPage, middleware.EditorAuth)
	v1.POST("/pages", srv.CreatePage, middleware.EditorAuth)
	v1.GET("/pages/:id", srv.GetPage, middleware.EditorAuth)
	v1.POST("/forms/:form_slug/submit", srv.SubmitForm)
	v1.POST("/forms/:form_slug/submit/batch", srv.SubmitFormBatch)
	v1.GET("/forms/:form_slug/schema", srv.FormSchema)
	v1.POST("/track/pageview", srv.TrackPageview)
	v1.POST("/track/event", srv.TrackEvent)
	v1.POST("/track/conversion", srv.TrackConversion)
	v1.GET("/track/pixel/:tenant_slug/p.gif", srv.TrackingPixel)
	v1.GET("/track/redirect/:tenant_slug/:click_id", srv.TrackingRedirect)
	v1.POST("/capi/send", srv.CAPISend)
	v1.GET("/capi/status", srv.CAPIStatus)
	v1.POST("/capi/test", srv.CAPITest, middleware.EditorAuth)

	rpc := e.Group("/internal/landing.v1.LandingService")
	rpc.POST("/GetPageConfig", srv.RenderPage)
	rpc.POST("/SubmitLead", srv.SubmitForm)
	rpc.POST("/TrackEvent", srv.TrackEvent)
	rpc.POST("/SendCAPIEvent", srv.CAPISend)
	rpc.POST("/GetFBPixelConfig", srv.CAPIStatus)
	return e
}
