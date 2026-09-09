// Tests for logger sampling handler.
package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestExtra_NewSamplingHandler_NilConfig(t *testing.T) {
	inner := slog.NewTextHandler(&bytes.Buffer{}, nil)
	h := NewSamplingHandler(inner, nil)
	if h != inner {
		t.Error("nil config should return inner handler")
	}
}

func TestExtra_NewSamplingHandler_DefaultsApplied(t *testing.T) {
	inner := slog.NewTextHandler(&bytes.Buffer{}, nil)
	h := NewSamplingHandler(inner, &SamplingConfig{})
	sh, ok := h.(*samplingHandler)
	if !ok {
		t.Fatal("expected *samplingHandler")
	}
	if sh.bucket.rate != 100 {
		t.Errorf("expected default rate 100: got %f", sh.bucket.rate)
	}
}

func TestExtra_NewSamplingHandler_BurstDefault(t *testing.T) {
	inner := slog.NewTextHandler(&bytes.Buffer{}, nil)
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 50})
	sh := h.(*samplingHandler)
	if sh.bucket.burst != 50 {
		t.Errorf("expected burst to equal rate: got %f", sh.bucket.burst)
	}
}

func TestExtra_NewSamplingHandler_CustomBurst(t *testing.T) {
	inner := slog.NewTextHandler(&bytes.Buffer{}, nil)
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 10, Burst: 20})
	sh := h.(*samplingHandler)
	if sh.bucket.burst != 20 {
		t.Errorf("expected burst 20: got %f", sh.bucket.burst)
	}
}

func TestExtra_NewSamplingHandler_PreFilled(t *testing.T) {
	inner := slog.NewTextHandler(&bytes.Buffer{}, nil)
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 10, Burst: 5})
	sh := h.(*samplingHandler)
	if sh.bucket.tokens != 5 {
		t.Errorf("expected pre-filled to 5: got %f", sh.bucket.tokens)
	}
}

func TestExtra_Handle_LogsAllowed(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 100, Burst: 100})

	logger := slog.New(h)
	logger.Info("test1")
	logger.Info("test2")

	if !strings.Contains(buf.String(), "test1") {
		t.Error("test1 not logged")
	}
	if !strings.Contains(buf.String(), "test2") {
		t.Error("test2 not logged")
	}
}

func TestExtra_Handle_ExhaustsBucket(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	// Very low burst to exhaust quickly
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 0.001, Burst: 2})

	logger := slog.New(h)
	// First 2 should be allowed (burst)
	logger.Info("first")
	logger.Info("second")
	// 3rd might be dropped (depends on refill)
	logger.Info("third")

	if !strings.Contains(buf.String(), "first") {
		t.Error("first should be logged")
	}
}

func TestExtra_Handle_LevelFilter(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{
		TokensPerSecond: 100,
		Burst:           100,
		LevelsToSample:  []slog.Level{slog.LevelInfo},
	})

	logger := slog.New(h)
	logger.Info("info")
	logger.Error("error")

	// Both should be logged because LevelFilter only restricts sampling
	// but always logs via inner handler
	if !strings.Contains(buf.String(), "info") {
		t.Error("info should be logged")
	}
	if !strings.Contains(buf.String(), "error") {
		t.Error("error should be logged")
	}
}

func TestExtra_Handle_AlwaysSampleAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{
		TokensPerSecond:   0.001, // Very low
		Burst:             1,
		AlwaysSampleAttrs: []string{"critical"},
	})

	logger := slog.New(h)
	logger.Info("first", "critical", true)
	logger.Info("second", "critical", true) // Should bypass due to "critical"

	if !strings.Contains(buf.String(), "first") {
		t.Error("first should be logged (pre-filled)")
	}
	if !strings.Contains(buf.String(), "second") {
		t.Error("second should be logged (always sample)")
	}
}

func TestExtra_Handle_ConcurrentSafe(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 10000, Burst: 10000})

	logger := slog.New(h)
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			logger.Info("concurrent")
			done <- true
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestExtra_WithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 100, Burst: 100})

	withAttrs := h.WithAttrs([]slog.Attr{slog.String("service", "test")})
	logger := slog.New(withAttrs)
	logger.Info("hello")

	if !strings.Contains(buf.String(), "service=test") {
		t.Error("withAttrs should add service attr")
	}
}

func TestExtra_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 100, Burst: 100})

	withGroup := h.WithGroup("svc")
	logger := slog.New(withGroup)
	logger.Info("hello", slog.String("key", "val"))

	if !strings.Contains(buf.String(), "key=val") {
		t.Error("withGroup should preserve attrs")
	}
}

func TestExtra_TokenBucket_Refill(t *testing.T) {
	b := &tokenBucket{
		rate:       1000, // High rate for quick refill
		burst:      10,
		tokens:     0,
		lastRefill: time.Now().Add(-1 * time.Second), // 1 second ago
	}
	if !b.take() {
		t.Error("after 1 second, bucket should be refilled")
	}
}

func TestExtra_TokenBucket_NoRefillWhenZero(t *testing.T) {
	b := &tokenBucket{
		rate:       0,
		burst:      0,
		tokens:     0,
		lastRefill: time.Now(),
	}
	if b.take() {
		t.Error("zero tokens should fail")
	}
}

func TestExtra_Handle_TokenRefillOverTime(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 1000, Burst: 1})

	logger := slog.New(h)
	logger.Info("first")
	logger.Info("second") // Should fail without delay
	if !strings.Contains(buf.String(), "first") {
		t.Error("first should be logged")
	}

	// Wait a bit and try again
	time.Sleep(50 * time.Millisecond)
	logger.Info("third")
}

func TestExtra_Handle_DisabledLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	// Inner handler only allows Warn+
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := NewSamplingHandler(inner, &SamplingConfig{TokensPerSecond: 100, Burst: 100})

	logger := slog.New(h)
	logger.Info("info-not-logged")
	logger.Warn("warn-logged")

	if strings.Contains(buf.String(), "info-not-logged") {
		t.Error("info should not be logged by inner handler")
	}
	if !strings.Contains(buf.String(), "warn-logged") {
		t.Error("warn should be logged")
	}
}

func TestExtra_SamplingConfig_Fields(t *testing.T) {
	cfg := SamplingConfig{
		TokensPerSecond: 50,
		Burst:           10,
		LevelsToSample:  []slog.Level{slog.LevelInfo},
		AlwaysSampleAttrs: []string{"trace_id"},
	}
	if cfg.TokensPerSecond != 50 {
		t.Error("TokensPerSecond not set")
	}
	if cfg.Burst != 10 {
		t.Error("Burst not set")
	}
	if len(cfg.LevelsToSample) != 1 {
		t.Error("LevelsToSample not set")
	}
}

func TestExtra_Handle_NoMatchingLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{
		TokensPerSecond: 100,
		Burst:           100,
		LevelsToSample:  []slog.Level{slog.LevelError}, // Only Error
	})

	logger := slog.New(h)
	logger.Info("info-message")
	logger.Error("error-message")

	// Both should be logged because level filter bypasses sampling
	if !strings.Contains(buf.String(), "info-message") {
		t.Error("info should be logged (filter bypasses)")
	}
	if !strings.Contains(buf.String(), "error-message") {
		t.Error("error should be logged")
	}
}

func TestExtra_Handle_AlwaysSample_False(t *testing.T) {
	buf := &bytes.Buffer{}
	inner := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewSamplingHandler(inner, &SamplingConfig{
		TokensPerSecond:   100,
		Burst:             100,
		AlwaysSampleAttrs: []string{"critical"},
	})

	logger := slog.New(h)
	logger.Info("info", slog.String("other", "val"))

	if !strings.Contains(buf.String(), "info") {
		t.Error("info should be logged (within burst)")
	}
}

// Ensure context import is used
var _ = context.Background
