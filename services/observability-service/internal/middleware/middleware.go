// Package middleware provides Echo middleware (auth, tenant resolution,
// recovery, CORS, rate limiting, security headers) used by the
// observability service.  Compared to other services we keep middleware
// small and dedicated; the rate limit uses an in-memory token bucket
// (no Redis dependency) and the auth middleware trusts X-Tenant-ID /
// X-User-ID headers + a static admin token for the AlertManager webhook.
package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/rinco/services/observability-service/internal/platform"
)

var (
	httpReqs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "observability_service_http_requests_total",
		Help: "HTTP requests handled by observability service",
	}, []string{"method", "route", "status"})
	httpDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "observability_service_http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// RecoveryMW catches panics and returns a 500.
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

// CORSMW applies a permissive CORS policy.
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

// TenantMW extracts X-Tenant-ID / X-User-ID headers and stores them on
// the Echo context.
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
			c.SetRequest(c.Request().WithContext(platform.WithTenant(c.Request().Context(), tenant, c.Get("user_id").(string), c.Get("is_admin").(bool))))
			return next(c)
		}
	}
}

// AuthMW requires either a tenant context (set by TenantMW) or an
// explicit Authorization header.
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

// RateLimitMW applies an in-memory token-bucket rate limit per
// (tenant_id, route).  No Redis dependency.
func RateLimitMW(limitPerMin int) echo.MiddlewareFunc {
	if limitPerMin <= 0 {
		limitPerMin = 600
	}
	type bucket struct {
		tokens   float64
		lastSeen time.Time
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	limit := float64(limitPerMin) / 60.0

	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			mu.Lock()
			cutoff := time.Now().Add(-15 * time.Minute)
			for k, b := range buckets {
				if b.lastSeen.Before(cutoff) {
					delete(buckets, k)
				}
			}
			mu.Unlock()
		}
	}()

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
			key := tenantID + ":" + route
			now := time.Now()

			mu.Lock()
			b, ok := buckets[key]
			if !ok {
				b = &bucket{tokens: float64(limitPerMin), lastSeen: now}
				buckets[key] = b
			}
			elapsed := now.Sub(b.lastSeen).Seconds()
			b.tokens += elapsed * limit
			if b.tokens > float64(limitPerMin) {
				b.tokens = float64(limitPerMin)
			}
			b.lastSeen = now
			if b.tokens < 1 {
				mu.Unlock()
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "rate limit exceeded",
				})
			}
			b.tokens -= 1
			mu.Unlock()
			return next(c)
		}
	}
}