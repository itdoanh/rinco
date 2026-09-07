package platform

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const serviceName = "dynamic-model-service"

func Getenv(key, fallback string) string { if value := os.Getenv(key); value != "" { return value }; return fallback }
func GetenvInt(key string, fallback int) int { value, err := strconv.Atoi(os.Getenv(key)); if err != nil { return fallback }; return value }

func InitLogger(env, version string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) { case "debug": level = slog.LevelDebug; case "warn": level = slog.LevelWarn; case "error": level = slog.LevelError }
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level, AddSource: env == "development"}).WithAttrs([]slog.Attr{slog.String("service", serviceName), slog.String("env", env), slog.String("version", version)}))
	slog.SetDefault(logger)
	return logger
}

func OpenDB(ctx context.Context, dsn string) (*sql.DB, error) {
	if dsn == "" { dsn = "postgres://rinco:rinco_dev_password@localhost:5432/rinco?sslmode=disable" }
	db, err := sql.Open("pgx", dsn); if err != nil { return nil, fmt.Errorf("open: %w", err) }
	db.SetMaxOpenConns(25); db.SetMaxIdleConns(2); db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil { db.Close(); return nil, fmt.Errorf("ping: %w", err) }
	return db, nil
}

func InitTracer(ctx context.Context, endpoint, env string) (*sdktrace.TracerProvider, error) {
	if endpoint == "" || env == "development" { return nil, nil }
	u, err := url.Parse(endpoint); if err != nil { return nil, err }
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(u.Host), otlptracegrpc.WithInsecure()); if err != nil { return nil, err }
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName), semconv.ServiceVersion("1.0.0"), semconv.DeploymentEnvironment(env))); if err != nil { return nil, err }
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return provider, nil
}

func RunMigrations(ctx context.Context, db *sql.DB) error { return db.PingContext(ctx) }
