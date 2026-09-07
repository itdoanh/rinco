package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var requests = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "landing_service_http_requests_total", Help: "HTTP requests handled by landing service"}, []string{"method", "route", "status"})
var latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "landing_service_http_request_duration_seconds", Help: "HTTP request latency"}, []string{"method", "route"})

func init() { prometheus.MustRegister(requests, latency) }

func Recovery() echo.MiddlewareFunc { return func(next echo.HandlerFunc) echo.HandlerFunc { return func(c echo.Context) (err error) { defer func() { if recovered := recover(); recovered != nil { slog.Error("panic recovered", slog.Any("panic", recovered)); err = c.JSON(http.StatusInternalServerError, map[string]string{"error":"internal server error"}) } }(); return next(c) } } }
func RequestLog() echo.MiddlewareFunc { return func(next echo.HandlerFunc) echo.HandlerFunc { return func(c echo.Context) error { started := time.Now(); err := next(c); route := c.Path(); if route == "" { route = c.Request().URL.Path }; status := c.Response().Status; requests.WithLabelValues(c.Request().Method, route, strconv.Itoa(status)).Inc(); latency.WithLabelValues(c.Request().Method, route).Observe(time.Since(started).Seconds()); slog.Info("http request", slog.String("method", c.Request().Method), slog.String("path", c.Request().URL.Path), slog.Int("status", status), slog.Duration("duration", time.Since(started))); return err } } }
func CORS() echo.MiddlewareFunc { return func(next echo.HandlerFunc) echo.HandlerFunc { return func(c echo.Context) error { h:=c.Response().Header(); h.Set("Access-Control-Allow-Origin", "*"); h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS"); h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Tenant-ID,X-User-ID,X-Admin,X-Idempotency-Key"); if c.Request().Method == http.MethodOptions { return c.NoContent(http.StatusNoContent) }; return next(c) } } }
func SecurityHeaders() echo.MiddlewareFunc { return func(next echo.HandlerFunc) echo.HandlerFunc { return func(c echo.Context) error { h:=c.Response().Header(); h.Set("X-Content-Type-Options","nosniff"); h.Set("X-Frame-Options","DENY"); h.Set("Referrer-Policy","strict-origin-when-cross-origin"); return next(c) } } }
func EditorAuth(next echo.HandlerFunc) echo.HandlerFunc { return func(c echo.Context) error { if c.Request().Header.Get("X-User-ID") == "" && c.Request().Header.Get("Authorization") == "" { return c.JSON(http.StatusUnauthorized, map[string]string{"error":"editor authentication required"}) }; return next(c) } }
func Tenant(c echo.Context) string { return c.Request().Header.Get("X-Tenant-ID") }
func User(c echo.Context) string { return c.Request().Header.Get("X-User-ID") }
