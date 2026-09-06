// Package middleware cung cấp HTTP middleware cho Echo framework.
package middleware

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/logger"
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
				traceID = uuid.NewV7().String()
			}

			// 2. Gắn vào context
			ctx = logger.WithTraceID(ctx, traceID)
			ctx = logger.WithRequestID(ctx, uuid.NewV7().String())

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

			fields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.String("route", c.Path()),
				zap.Int("status", status),
				zap.Duration("duration", duration),
				zap.String("ip", c.RealIP()),
				zap.String("user_agent", req.UserAgent()),
			}

			if err != nil {
				fields = append(fields, zap.Error(err))
				logger.Error(ctx, "request failed", err, fields...)
			} else if status >= 500 {
				logger.Error(ctx, "server error", nil, fields...)
			} else if status >= 400 {
				logger.Warn(ctx, "client error", fields...)
			} else {
				logger.Info(ctx, "request", fields...)
			}

			return err
		}
	}
}

// Metrics middleware: đo Prometheus metrics.
func Metrics(serviceName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			inflight := httpInflight.WithLabelValues(serviceName)
			inflight.Inc()
			defer inflight.Dec()

			err := next(c)

			route := c.Path()
			status := ""
			if c.Response() != nil {
				status = c.Response().StatusString()
			}

			httpRequestsTotal.WithLabelValues(serviceName, c.Request().Method, route, status).Inc()
			httpRequestDuration.WithLabelValues(serviceName, c.Request().Method, route).Observe(time.Since(start).Seconds())

			return err
		}
	}
}

// Recovery middleware: handle panic gracefully.
func Recovery() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					ctx := c.Request().Context()
					logger.Error(ctx, "panic recovered", nil,
						zap.Any("panic", r),
						zap.String("path", c.Request().URL.Path),
					)
					err = echo.NewHTTPError(500, "Internal Server Error")
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
