// Package middleware provides Echo middleware (auth, tenant resolution,
// recovery, CORS, rate limiting, metrics, security headers) used by the
// email service.
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
	"github.com/redis/go-redis/v9"

	"github.com/rinco/services/email-service/internal/platform"
)

var (
	httpReqs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "email_service_http_requests_total",
		Help: "HTTP requests handled by email service",
	}, []string{"method", "route", "status"})
	httpDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "email_service_http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// RecoveryMW catches panics and returns a 500 with sanitised JSON.
func RecoveryMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic recovered",
						slog.Any("panic", r),
						slog.String("path", c.Request().URL.Path))
					_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
				}
			}()
			return next(c)
		}
	}
}

// LoggingMW logs every request and updates Prometheus counters.
func LoggingMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			route := c.Path()
			if route == "" {
				route = c.Request().URL.Path
			}
			httpReqs.WithLabelValues(c.Request().Method, route, strconv.Itoa(c.Response().Status)).Inc()
			httpDur.WithLabelValues(c.Request().Method, route).Observe(time.Since(start).Seconds())
			slog.Info("request",
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", c.Response().Status),
				slog.Duration("duration", time.Since(start)),
				slog.String("ip", c.RealIP()),
			)
			return err
		}
	}
}

// CORSMW applies a permissive CORS policy suitable for public APIs.
func CORSMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("Access-Control-Allow-Origin", "*")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-User-ID, X-Admin, X-Idempotency-Key")
			h.Set("Access-Control-Max-Age", "86400")
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}

// SecurityHeadersMW applies conservative default headers.
func SecurityHeadersMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			return next(c)
		}
	}
}

// TenantMW extracts X-Tenant-ID / X-User-ID headers and stores them on the
// Echo context. Returns 400 when tenant header is missing.
func TenantMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenant := strings.TrimSpace(c.Request().Header.Get("X-Tenant-ID"))
			if tenant == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID header required"})
			}
			c.Set("tenant_id", tenant)
			c.Set("user_id", c.Request().Header.Get("X-User-ID"))
			c.Set("is_admin", strings.EqualFold(c.Request().Header.Get("X-Admin"), "true"))
			c.SetRequest(c.Request().WithContext(platform.TenantToCtx(c.Request().Context(), tenant, c.Get("user_id").(string), c.Get("is_admin").(bool))))
			return next(c)
		}
	}
}

// AuthMW requires either a tenant context (set by TenantMW) or an explicit
// Authorization header.
func AuthMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Get("tenant_id") == nil || c.Get("tenant_id").(string) == "" {
				if c.Request().Header.Get("Authorization") == "" {
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				}
			}
			return next(c)
		}
	}
}

// RateLimitMW applies a simple fixed-window rate limit using a Redis-backed
// counter. The limit is per (tenant_id, route). The middleware is a no-op
// when redis is nil.
func RateLimitMW(rdb *redis.Client, limit int, window time.Duration) echo.MiddlewareFunc {
	if rdb == nil {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenant := c.Get("tenant_id")
			tenantID, _ := tenant.(string)
			if tenantID == "" {
				tenantID = c.RealIP()
			}
			route := c.Path()
			if route == "" {
				route = c.Request().URL.Path
			}
			key := "rl:email:" + tenantID + ":" + route
			ctx := c.Request().Context()
			count, err := rdb.Incr(ctx, key).Result()
			if err == nil && count == 1 {
				_ = rdb.Expire(ctx, key, window).Err()
			}
			if err == nil && count > int64(limit) {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "rate limit exceeded",
					"limit": strconv.Itoa(limit),
				})
			}
			return next(c)
		}
	}
}