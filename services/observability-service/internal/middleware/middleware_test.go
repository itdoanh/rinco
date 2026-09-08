// Tests for observability-service middleware.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestRateLimitMW_DefaultLimit(t *testing.T) {
	// Use limit > 0 to override default.
	mw := RateLimitMW(60)

	e := echo.New()
	calls := 0
	h := mw(func(c echo.Context) error {
		calls++
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRateLimitMW_ZeroLimit_DefaultsTo600(t *testing.T) {
	// limit <= 0 should use default 600.
	mw := RateLimitMW(0)

	e := echo.New()
	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRateLimitMW_ExhaustsTokens(t *testing.T) {
	// Set tiny limit so we can exhaust tokens quickly.
	mw := RateLimitMW(1) // 1 token per minute → 1 token total

	e := echo.New()
	calls := 0
	h := mw(func(c echo.Context) error {
		calls++
		return c.NoContent(http.StatusOK)
	})

	// First call should succeed.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("first call expected to succeed: got calls=%d", calls)
	}

	// Second call immediately should fail (rate limited).
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	if err := h(c2); err != nil {
		t.Fatal(err)
	}
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "rate limit exceeded") {
		t.Errorf("expected rate limit message, got: %s", rec2.Body.String())
	}
}

func TestRateLimitMW_DifferentKeys(t *testing.T) {
	// Different tenants or IPs should have separate buckets.
	mw := RateLimitMW(1)

	e := echo.New()
	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// First request from IP1.
	req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	if err := h(c1); err != nil {
		t.Fatal(err)
	}
	if rec1.Code != http.StatusOK {
		t.Errorf("IP1 first call: expected 200, got %d", rec1.Code)
	}

	// First request from IP2 should also succeed (different bucket).
	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.RemoteAddr = "5.6.7.8:1234"
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	if err := h(c2); err != nil {
		t.Fatal(err)
	}
	if rec2.Code != http.StatusOK {
		t.Errorf("IP2 first call: expected 200, got %d", rec2.Code)
	}
}

func TestRateLimitMW_TenantFromContext(t *testing.T) {
	// If tenant_id is set in context, use it as bucket key.
	mw := RateLimitMW(60)

	e := echo.New()
	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-abc")

	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRateLimitMW_Concurrent(t *testing.T) {
	// Run 50 concurrent requests against same bucket; ensure no race/panic.
	mw := RateLimitMW(100)

	e := echo.New()
	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	var wg sync.WaitGroup
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			_ = h(c)
		}()
	}
	wg.Wait()
}

func TestRateLimitMW_RefillOverTime(t *testing.T) {
	// Use a fast refill (limit high enough that one bucket gets tokens
	// refilled after a tiny wait).
	mw := RateLimitMW(60) // 60/min = 1 token/sec

	e := echo.New()
	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// Drain bucket.
	for i := 0; i < 60; i++ {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = h(c)
	}

	// Wait 2 seconds for refill (1 token/sec * 2 = 2 tokens).
	time.Sleep(2 * time.Second)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("after refill: expected 200, got %d", rec.Code)
	}
}
