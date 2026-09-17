// Package middleware — RINCO WS-E security middleware.
//
// Composed for the WS-E "security hardening" milestone (DEV-PLAN §10):
//
//   - Rate limit middleware backed by the packages/go/ratelimit token
//     bucket (in-memory) or the Valkey-backed sliding window when a
//     Redis client is supplied.
//   - CORS, CSP, HSTS, Referrer-Policy and Permissions-Policy headers.
//   - Input sanitization helpers (HTML escape, JSON escape, header
//     allowlist).
//
// Every middleware is "best-effort": if a downstream dependency
// (Valkey, etc.) is down it falls back to in-memory, never blocks
// the request.
package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/itdoanh/rinco/packages/go/ratelimit"
)

// =============================================================================
// Security headers (CORS + CSP + HSTS)
// =============================================================================

// SecurityHeadersConfig configures the security-headers middleware.
type SecurityHeadersConfig struct {
	// AllowedOrigins is the list of origins permitted by CORS.  "*"
	// allows any origin (NOT recommended for production).
	AllowedOrigins []string
	// AllowedMethods is the HTTP methods allowed for CORS pre-flight.
	AllowedMethods []string
	// AllowedHeaders is the HTTP headers allowed for CORS pre-flight.
	AllowedHeaders []string
	// ExposedHeaders are headers the browser may surface to JS.
	ExposedHeaders []string
	// AllowCredentials toggles Access-Control-Allow-Credentials.
	AllowCredentials bool
	// MaxAge is the preflight cache lifetime.
	MaxAge time.Duration

	// EnableHSTS toggles Strict-Transport-Security.
	EnableHSTS bool
	// HSTSMaxAge is the max-age value.
	HSTSMaxAge time.Duration
	// HSTSPreload toggles "preload" directive.
	HSTSPreload bool
	// HSTSSubdomains toggles "includeSubDomains".
	HSTSSubdomains bool

	// CSP is the Content-Security-Policy value.  When empty a
	// strict default is used.
	CSP string
	// PermissionsPolicy is the Permissions-Policy value.
	PermissionsPolicy string
	// FrameOptions sets X-Frame-Options.  Default DENY.
	FrameOptions string
	// ReferrerPolicy sets Referrer-Policy.  Default
	// strict-origin-when-cross-origin.
	ReferrerPolicy string

	// Logger for diagnostics.
	Logger *slog.Logger
}

// DefaultSecurityHeaders returns a sane default for browser-facing
// services.
func DefaultSecurityHeaders(allowedOrigins []string) SecurityHeadersConfig {
	return SecurityHeadersConfig{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Tenant-ID", "X-User-ID", "X-Request-ID", "X-Idempotency-Key", "X-Admin"},
		ExposedHeaders:   []string{"X-Request-ID", "X-Trace-ID", "X-RateLimit-Remaining"},
		AllowCredentials: false,
		MaxAge:           24 * time.Hour,
		EnableHSTS:       true,
		HSTSMaxAge:       365 * 24 * time.Hour,
		HSTSPreload:      true,
		HSTSSubdomains:   true,
		CSP:              "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' wss: https:; frame-ancestors 'none'; base-uri 'self'; object-src 'none'",
		PermissionsPolicy: "camera=(), microphone=(), geolocation=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
		FrameOptions:     "DENY",
		ReferrerPolicy:   "strict-origin-when-cross-origin",
		Logger:           slog.Default(),
	}
}

// SecurityHeaders is the Echo middleware that applies CORS, CSP, HSTS
// and other hardening headers.  OPTIONS requests are short-circuited
// with 204.
func SecurityHeadersWithConfig(cfg SecurityHeadersConfig) echo.MiddlewareFunc {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{"Content-Type", "Authorization", "X-Tenant-ID", "X-User-ID"}
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 24 * time.Hour
	}
	if cfg.FrameOptions == "" {
		cfg.FrameOptions = "DENY"
	}
	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = "strict-origin-when-cross-origin"
	}
	if cfg.CSP == "" {
		cfg.CSP = "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; object-src 'none'"
	}

	allowedOrigins := map[string]bool{}
	wildcard := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			wildcard = true
		}
		allowedOrigins[strings.ToLower(strings.TrimSpace(o))] = true
	}

	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := int(cfg.MaxAge.Seconds())

	hstsParts := []string{}
	if cfg.EnableHSTS {
		hstsParts = append(hstsParts, fmt.Sprintf("max-age=%d", int(cfg.HSTSMaxAge.Seconds())))
		if cfg.HSTSSubdomains {
			hstsParts = append(hstsParts, "includeSubDomains")
		}
		if cfg.HSTSPreload {
			hstsParts = append(hstsParts, "preload")
		}
	}
	hstsValue := strings.Join(hstsParts, "; ")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			req := c.Request()

			origin := strings.ToLower(strings.TrimSpace(req.Header.Get("Origin")))
			if wildcard && origin != "" {
				h.Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && allowedOrigins[origin] {
				h.Set("Access-Control-Allow-Origin", req.Header.Get("Origin"))
				h.Add("Vary", "Origin")
			}

			h.Set("Access-Control-Allow-Methods", allowedMethods)
			h.Set("Access-Control-Allow-Headers", allowedHeaders)
			if exposedHeaders != "" {
				h.Set("Access-Control-Expose-Headers", exposedHeaders)
			}
			h.Set("Access-Control-Max-Age", fmt.Sprintf("%d", maxAge))
			h.Set("Access-Control-Allow-Credentials", boolHeader(cfg.AllowCredentials))

			if hstsValue != "" {
				h.Set("Strict-Transport-Security", hstsValue)
			}
			h.Set("Content-Security-Policy", cfg.CSP)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", cfg.FrameOptions)
			h.Set("Referrer-Policy", cfg.ReferrerPolicy)
			h.Set("Permissions-Policy", cfg.PermissionsPolicy)
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Embedder-Policy", "require-corp")
			h.Set("Cross-Origin-Resource-Policy", "same-site")

			if req.Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}

func boolHeader(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// =============================================================================
// Rate limit (token bucket via packages/go/ratelimit + optional Valkey)
// =============================================================================

// RateLimiterConfig configures the rate-limit middleware.  When
// Redis is nil the middleware falls back to an in-memory token
// bucket.
type RateLimiterConfig struct {
	// Rate is requests-per-second per key.
	Rate float64
	// Burst is the bucket size.
	Burst int
	// WindowSize controls the fixed-window / sliding-window size when
	// Redis is used.
	WindowSize time.Duration
	// WindowMax is the max requests per window when Redis is used.
	WindowMax int

	// Redis is the optional Valkey/Redis client.  When set the
	// middleware uses a sliding-window log; otherwise in-memory.
	Redis *redis.Client

	// KeyFunc returns the rate-limit key.  Defaults to the tenant
	// header or the remote IP.
	KeyFunc func(echo.Context) string
	// SkipPaths bypasses rate limiting entirely.
	SkipPaths []string
	// OnLimit is called when a request is rate-limited (default
	// returns 429 JSON).
	OnLimit func(echo.Context, ratelimit.Decision) error
	// Logger for diagnostics.
	Logger *slog.Logger
}

// DefaultRateLimiter returns a conservative default (100 req/s, burst
// 200) for non-Redis deployments.
func DefaultRateLimiter() RateLimiterConfig {
	return RateLimiterConfig{
		Rate:       100,
		Burst:      200,
		WindowSize: time.Minute,
		WindowMax:  600,
		Logger:     slog.Default(),
	}
}

// RateLimiter is the Echo middleware.  It applies a per-key token
// bucket (in-memory) when no Redis is configured, or a sliding-window
// log via Redis when one is provided.  Best-effort: when Redis is
// unreachable the middleware falls back to in-memory.
func RateLimiter(cfg RateLimiterConfig) echo.MiddlewareFunc {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Rate <= 0 {
		cfg.Rate = 100
	}
	if cfg.Burst <= 0 {
		cfg.Burst = int(cfg.Rate * 2)
	}
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = time.Minute
	}
	if cfg.WindowMax <= 0 {
		cfg.WindowMax = int(cfg.Rate * 60)
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c echo.Context) string {
			if t := c.Request().Header.Get("X-Tenant-ID"); t != "" {
				return "t:" + t + ":" + c.Path()
			}
			return "ip:" + c.RealIP() + ":" + c.Path()
		}
	}

	skip := map[string]bool{}
	for _, p := range cfg.SkipPaths {
		skip[p] = true
	}

	memLimiter := ratelimit.NewTokenBucket(cfg.Rate, cfg.Burst)
	var redisLimiter *ratelimit.SlidingWindow
	if cfg.Redis != nil {
		redisLimiter = ratelimit.NewSlidingWindow(cfg.Redis, cfg.WindowSize, cfg.WindowMax)
	}
	var blocked atomic.Uint64

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if skip[path] {
				return next(c)
			}

			key := cfg.KeyFunc(c)
			ctx := c.Request().Context()

			var (
				decision ratelimit.Decision
				err      error
				source   string
			)
			if redisLimiter != nil {
				decision, err = redisLimiter.Allow(ctx, key)
				source = "redis"
				if err != nil {
					cfg.Logger.Warn("rate limit redis failed; falling back to in-memory",
						slog.String("err", err.Error()))
					decision, err = memLimiter.Allow(ctx, key)
					source = "memory"
				}
			} else {
				decision, err = memLimiter.Allow(ctx, key)
				source = "memory"
			}
			if err != nil && !errors.Is(err, ratelimit.ErrLimitExceeded) {
				// Total failure -> best-effort: allow.
				cfg.Logger.Warn("rate limit allow failed; allowing through",
					slog.String("err", err.Error()), slog.String("source", source))
				return next(c)
			}

			// Always surface rate-limit headers (helpful for clients
			// and observability).
			h := c.Response().Header()
			h.Set("X-RateLimit-Limit", fmt.Sprintf("%d", decision.Limit))
			h.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", maxInt(decision.Remaining, 0)))
			h.Set("X-RateLimit-Source", source)
			if decision.RetryAfter > 0 {
				h.Set("Retry-After", fmt.Sprintf("%d", int(decision.RetryAfter.Seconds())))
			}

			if !decision.Allowed {
				blocked.Add(1)
				cfg.Logger.Info("rate limit hit",
					slog.String("key", key),
					slog.String("path", path),
					slog.String("source", source),
					slog.Uint64("blocked_total", blocked.Load()),
				)
				if cfg.OnLimit != nil {
					return cfg.OnLimit(c, decision)
				}
				return c.JSON(http.StatusTooManyRequests, map[string]any{
					"error":       "rate limit exceeded",
					"retry_after": int(decision.RetryAfter.Seconds()),
					"limit":       decision.Limit,
					"source":      source,
				})
			}

			return next(c)
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// =============================================================================
// Input sanitization helpers
// =============================================================================

// SanitizeString is a fast-path strip of control characters and a
// length cap.  Use it for free-form text fields (names, subjects,
// descriptions).
func SanitizeString(s string, maxLen int) string {
	if s == "" {
		return s
	}
	// Strip NULL bytes, keep tabs/newlines.
	s = strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		if r < 0x20 && r != '\n' && r != '\t' && r != '\r' {
			return -1
		}
		return r
	}, s)
	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen]
	}
	return strings.TrimSpace(s)
}

// SanitizeHTML escapes HTML metacharacters.  Use for any string that
// is rendered in a browser.
func SanitizeHTML(s string) string {
	return html.EscapeString(s)
}

// SanitizeEmail trims, lowercases and validates an email-like string.
func SanitizeEmail(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) > 254 {
		return ""
	}
	if strings.ContainsAny(s, " \t\r\n<>\"'") {
		return ""
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return ""
	}
	return s
}

// SanitizeJSON sanitizes a JSON object (or array) recursively: every
// string field is run through SanitizeString with the provided
// limits.  Sensitive keys (per DefaultSensitiveFields) are
// redacted to "[REDACTED]".
func SanitizeJSON(payload []byte, maxStringLen int) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	v = sanitizeJSONValue(v, maxStringLen)
	return json.Marshal(v)
}

func sanitizeJSONValue(v any, maxLen int) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if isSensitive(k, DefaultSensitiveFields) {
				t[k] = "[REDACTED]"
				continue
			}
			t[k] = sanitizeJSONValue(val, maxLen)
		}
		return t
	case []any:
		for i, item := range t {
			t[i] = sanitizeJSONValue(item, maxLen)
		}
		return t
	case string:
		return SanitizeString(t, maxLen)
	default:
		return v
	}
}

// SanitizeHeaderValue removes CR/LF from a header value (defends
// against header-injection / response splitting).
func SanitizeHeaderValue(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, v)
}

// ValidateUUID returns the canonical lower-case UUID or an error.
func ValidateUUID(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 36 {
		return "", errors.New("invalid uuid length")
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return "", errors.New("invalid uuid format")
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return "", errors.New("invalid uuid char")
			}
		}
	}
	return s, nil
}

// SanitizeMiddleware is an Echo middleware that wraps any handler
// whose body is JSON and sanitizes every string field before
// delegating to the handler.  It is best-effort: malformed bodies
// are passed through unchanged.
func SanitizeMiddleware(maxStringLen int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			if req.Body != nil && req.ContentLength > 0 {
				ct := req.Header.Get("Content-Type")
				if strings.Contains(ct, "application/json") {
					body, err := readAndRestoreBody(req)
					if err == nil && len(body) > 0 {
						sanitized, err := SanitizeJSON(body, maxStringLen)
						if err == nil {
							req.Body = readCloser{Reader: bytes.NewReader(sanitized)}
							req.ContentLength = int64(len(sanitized))
						}
					}
				}
			}
			return next(c)
		}
	}
}

func readAndRestoreBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	defer req.Body.Close()
	buf, err := readAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = readCloser{Reader: bytes.NewReader(buf)}
	return buf, nil
}

type readCloser struct{ *bytes.Reader }

func (readCloser) Close() error { return nil }

func readAll(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return buf.Bytes(), nil
			}
			return buf.Bytes(), err
		}
	}
}