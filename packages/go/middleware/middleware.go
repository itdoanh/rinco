// Package middleware cung cấp HTTP middleware cho Echo framework.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/itdoanh/rinco/packages/go/logger"
	"log/slog"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rinco_http_requests_total",
			Help: "Total HTTP requests processed",
		},
		[]string{"service", "method", "route", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rinco_http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "route"},
	)
	httpInflight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rinco_http_inflight_requests",
			Help: "Number of inflight HTTP requests",
		},
		[]string{"service"},
	)
)

// Config cho middleware.
type Config struct {
	ServiceName string
}

// Trace middleware: gắn trace_id vào context và response header.
func Trace() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// 1. Lấy trace_id từ header hoặc sinh mới
			traceID := c.Request().Header.Get("X-Trace-ID")
			if traceID == "" {
				id, err := uuid.NewV7()
				if err == nil {
					traceID = id.String()
				}
			}

			// 2. Gắn vào context
			ctx = logger.WithTraceID(ctx, traceID)
			reqID, err := uuid.NewV7()
			if err == nil {
				ctx = logger.WithRequestID(ctx, reqID.String())
			}

			// 3. Lưu trace vào response header
			c.Response().Header().Set("X-Trace-ID", traceID)

			// 4. Tiếp tục chain
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// Logger middleware: log mỗi request.
func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			ctx := c.Request().Context()

			err := next(c)

			req := c.Request()
			res := c.Response()

			duration := time.Since(start)
			status := res.Status

			log := slog.Default()
			if err != nil {
				log.ErrorContext(ctx, "request failed",
					slog.String("method", req.Method),
					slog.String("path", req.URL.Path),
					slog.String("route", c.Path()),
					slog.Int("status", status),
					slog.Duration("duration", duration),
					slog.String("ip", c.RealIP()),
					slog.String("user_agent", req.UserAgent()),
					slog.String("error", err.Error()),
				)
			} else if status >= 500 {
				log.ErrorContext(ctx, "server error",
					slog.String("method", req.Method),
					slog.String("path", req.URL.Path),
					slog.Int("status", status),
				)
			} else if status >= 400 {
				log.WarnContext(ctx, "client error",
					slog.String("method", req.Method),
					slog.String("path", req.URL.Path),
					slog.Int("status", status),
				)
			} else {
				log.InfoContext(ctx, "request",
					slog.String("method", req.Method),
					slog.String("path", req.URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", duration),
				)
			}

			return err
		}
	}
}

// RequestID middleware: gắn request_id vào context.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := c.Request().Header.Get("X-Request-ID")
			if reqID == "" {
				id, err := uuid.NewV7()
				if err == nil {
					reqID = id.String()
				}
			}
			c.Response().Header().Set("X-Request-ID", reqID)
			return next(c)
		}
	}
}

// Recover middleware: panic recovery.
func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					var err error
					if e, ok := r.(error); ok {
						err = e
					} else {
						err = fmt.Errorf("%v", r)
					}
					log := slog.Default()
					log.ErrorContext(c.Request().Context(), "panic recovered",
						slog.String("panic", fmt.Sprintf("%v", r)),
						slog.String("path", c.Path()),
					)
					_ = c.String(http.StatusInternalServerError, "Internal Server Error")
					_ = err // suppress unused variable
				}
			}()
			return next(c)
		}
	}
}

// CORS middleware.
func CORS(allowedOrigins []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				c.Response().Header().Set("Access-Control-Allow-Origin", origin)
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Trace-ID, X-Tenant-ID")
				c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
				c.Response().Header().Set("Access-Control-Max-Age", "86400")
			}

			if c.Request().Method == "OPTIONS" {
				return c.NoContent(204)
			}

			return next(c)
		}
	}
}

// SecurityHeaders middleware: thêm security headers.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			return next(c)
		}
	}
}

// RequestIDFromContext extracts request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	return logger.RequestIDFromContext(ctx)
}

// TraceIDFromContext extracts trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	return logger.TraceIDFromContext(ctx)
}
