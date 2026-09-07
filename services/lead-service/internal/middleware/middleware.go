// Package middleware provides HTTP middleware for lead service.
package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpReqs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lead_service_http_requests_total", Help: "HTTP requests",
	}, []string{"method", "route", "status"})
	httpDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lead_service_http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// Recovery middleware.
func RecoveryMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic recovered", slog.Any("panic", r), slog.String("path", c.Request().URL.Path))
					_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
				}
			}()
			return next(c)
		}
	}
}

// Logging middleware.
func LoggingMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start).Seconds()
			status := c.Response().Status
			route := c.Path()
			if route == "" {
				route = c.Request().URL.Path
			}
			httpReqs.WithLabelValues(c.Request().Method, route, strconv.Itoa(status)).Inc()
			httpDur.WithLabelValues(c.Request().Method, route).Observe(latency)
			slog.Info("request",
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", status),
				slog.Float64("latency_s", latency),
				slog.String("ip", c.RealIP()),
			)
			return err
		}
	}
}

// CORS middleware.
func CORSMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Access-Control-Allow-Origin", "*")
			c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-User-ID, X-Admin")
			c.Response().Header().Set("Access-Control-Max-Age", "86400")
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}

// Security headers middleware.
func SecurityHeadersMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			return next(c)
		}
	}
}

// Metrics middleware.
func MetricsMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start).Seconds()
			route := c.Path()
			if route == "" {
				route = c.Request().URL.Path
			}
			httpReqs.WithLabelValues(c.Request().Method, route, strconv.Itoa(c.Response().Status)).Inc()
			httpDur.WithLabelValues(c.Request().Method, route).Observe(latency)
			return err
		}
	}
}

// Trace middleware.
func TraceMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			traceID := c.Request().Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = c.Response().Header().Get("X-Trace-ID")
			}
			c.Set("trace_id", traceID)
			return next(c)
		}
	}
}

// Tenant middleware - extracts tenant info from headers.
func TenantMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Request().Header.Get("X-Tenant-ID")
			userID := c.Request().Header.Get("X-User-ID")
			isAdminStr := c.Request().Header.Get("X-Admin")
			isAdmin := strings.ToLower(isAdminStr) == "true"

			if tenantID == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID header required"})
			}
			c.Set("tenant_id", tenantID)
			c.Set("user_id", userID)
			c.Set("is_admin", isAdmin)
			return next(c)
		}
	}
}

// Auth middleware (simplified - just checks headers exist).
func AuthMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Get("tenant_id")
			userID := c.Get("user_id")
			if tenantID == nil || tenantID.(string) == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}
			_ = userID
			return next(c)
		}
	}
}
