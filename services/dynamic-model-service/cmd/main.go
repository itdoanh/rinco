// Package main — RINCO dynamic-model-service entrypoint.
//
// Meta-schema engine that allows tenants to define their own entity
// types (models) with field definitions, validation rules, JSONB-backed
// record storage, versioning, and CSV import/export.  Multi-tenant via
// X-Tenant-ID / X-User-ID headers and PostgreSQL RLS.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

	"github.com/rinco/services/dynamic-model-service/internal/handler"
	"github.com/rinco/services/dynamic-model-service/internal/middleware"
	"github.com/rinco/services/dynamic-model-service/internal/platform"
)

const serviceName = "dynamic-model-service"
const version = "1.0.0"

func main() {
	cfg := loadConfig()
	logger := platform.InitLogger(cfg.Env, version)
	logger.Info("dynamic-model-service starting", slog.String("addr", cfg.HTTPAddr))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, err := platform.InitTracer(rootCtx, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), cfg.Env)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		defer func() { _ = tp.Shutdown(context.Background()) }()
	}

	db, err := platform.OpenDB(rootCtx, cfg.DatabaseURL)
	if err != nil { logger.Error("db connect failed", slog.String("error", err.Error())); os.Exit(1) }
	defer db.Close()
	if err := platform.RunMigrations(rootCtx, db); err != nil { logger.Error("migrations failed", slog.String("error", err.Error())); os.Exit(1) }

	srv := handler.New(db)
	e := newEcho(db, srv)
	go func() { if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) { logger.Error("http server", slog.String("error", err.Error())) } }()
	<-rootCtx.Done()
	logger.Info("shutdown requested")
	shut, cancel := context.WithTimeout(context.Background(), 30*time.Second); defer cancel()
	if err := e.Shutdown(shut); err != nil { logger.Error("graceful shutdown", slog.String("error", err.Error())) }
	logger.Info("dynamic-model-service stopped")
}

type config struct { Env, HTTPAddr, DatabaseURL, ValkeyURL string }

func loadConfig() config {
	return config{
		Env:         platform.Getenv("ENV", "development"),
		HTTPAddr:    platform.Getenv("DYNAMIC_MODEL_HTTP_ADDR", ":8084"),
		DatabaseURL: os.Getenv("DYNAMIC_MODEL_DATABASE_URL"),
		ValkeyURL:   platform.Getenv("DYNAMIC_MODEL_VALKEY_URL", "localhost:6379"),
	}
}

func newEcho(db *sql.DB, srv *handler.Server) *echo.Echo {
	e := echo.New(); e.HideBanner = true; e.HidePort = true
	e.Use(middleware.Recovery()); e.Use(middleware.RequestLog()); e.Use(middleware.CORS()); e.Use(middleware.SecurityHeaders()); e.Use(otelecho.Middleware(serviceName))

	e.GET("/healthz", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName, "version": version}) })
	e.GET("/readyz", func(c echo.Context) error { ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second); defer cancel(); if err := db.PingContext(ctx); err != nil { return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_unreachable", "error": err.Error()}) }; return c.JSON(http.StatusOK, map[string]string{"status": "ready"}) })
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/version", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"service": serviceName, "version": version}) })

	v1 := e.Group("/v1/models")
	v1.POST("", srv.CreateModel)
	v1.GET("", srv.ListModels)
	v1.GET("/:id", srv.GetModel)
	v1.PUT("/:id", srv.UpdateModel)
	v1.DELETE("/:id", srv.DeleteModel)
	v1.POST("/:id/duplicate", srv.DuplicateModel)
	v1.POST("/:id/publish", srv.PublishModel)
	v1.GET("/:id/versions", srv.ListVersions)
	v1.POST("/:id/restore/:version", srv.RestoreVersion)
	v1.POST("/:id/migrate", srv.MigrateModel)
	v1.POST("/:id/fields", srv.AddField)
	v1.GET("/:id/fields", srv.ListFields)
	v1.DELETE("/:id/fields/:field_id", srv.DeleteField)
	v1.POST("/:id/records", srv.CreateRecord)
	v1.GET("/:id/records", srv.ListRecords)
	v1.GET("/:id/records/:record_id", srv.GetRecord)
	v1.PUT("/:id/records/:record_id", srv.UpdateRecord)
	v1.DELETE("/:id/records/:record_id", srv.DeleteRecord)
	v1.POST("/:id/validate", srv.ValidateData)
	v1.POST("/:id/import", srv.ImportCSV)
	v1.GET("/:id/export", srv.ExportCSV)
	v1.GET("/:id/ui-schema", srv.GenerateUISchema)
	v1.GET("/:id/json-schema", srv.GenerateJSONSchema)

	rpc := e.Group("/internal/dynamic_model.v1.DynamicModelService")
	rpc.POST("/GetModel", srv.GetModel)
	rpc.POST("/ListModels", srv.ListModels)
	rpc.POST("/CreateModel", srv.CreateModel)
	rpc.POST("/UpdateModel", srv.UpdateModel)
	rpc.POST("/DeleteModel", srv.DeleteModel)
	rpc.POST("/ValidateRecord", srv.ValidateData)
	rpc.POST("/ImportCSV", srv.ImportCSV)
	rpc.POST("/ExportCSV", srv.ExportCSV)
	return e
}
