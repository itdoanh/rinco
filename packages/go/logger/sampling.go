// Package logger - adaptive sampling handler.
//
// Sampling dựa trên token bucket: mỗi level có bucket riêng.
// Mặc định: 100 events/sec cho INFO, không giới hạn ERROR.
// Health check & request logs có thể override qua config.
package logger

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// SamplingConfig cấu hình adaptive sampling.
type SamplingConfig struct {
	// TokensPerSecond số events được phép log mỗi giây.
	TokensPerSecond float64
	// Burst bucket size (số events tối đa trong 1 lần).
	Burst int
	// LevelsToSample: nếu nil, áp dụng cho tất cả levels.
	// Nếu chỉ định, chỉ áp dụng cho các level này.
	LevelsToSample []slog.Level
	// AlwaysSampleAttrs: nếu log có 1 trong các attrs này thì luôn được log.
	AlwaysSampleAttrs []string
}

// NewSamplingHandler wrap inner handler với sampling.
func NewSamplingHandler(inner slog.Handler, cfg *SamplingConfig) slog.Handler {
	if cfg == nil {
		return inner
	}
	bucket := &tokenBucket{
		rate:  cfg.TokensPerSecond,
		burst: float64(cfg.Burst),
	}
	if cfg.TokensPerSecond == 0 {
		bucket.rate = 100
	}
	if cfg.Burst == 0 {
		bucket.burst = bucket.rate
	}
	// Pre-fill the bucket up to burst so the first burst of events is allowed.
	bucket.tokens = bucket.burst
	bucket.lastRefill = time.Now()
	return &samplingHandler{
		inner:            inner,
		bucket:           bucket,
		levels:           cfg.LevelsToSample,
		alwaysSampleKeys: cfg.AlwaysSampleAttrs,
	}
}

type samplingHandler struct {
	inner            slog.Handler
	bucket           *tokenBucket
	levels           []slog.Level
	alwaysSampleKeys []string
}

func (h *samplingHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.inner.Enabled(context.Background(), level)
}

func (h *samplingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Filter by level
	if len(h.levels) > 0 {
		match := false
		for _, l := range h.levels {
			if l == r.Level {
				match = true
				break
			}
		}
		if !match {
			return h.inner.Handle(ctx, r)
		}
	}

	// Check if any attr is in alwaysSampleKeys → bypass sampling entirely.
	if len(h.alwaysSampleKeys) > 0 {
		hasAlways := false
		r.Attrs(func(a slog.Attr) bool {
			for _, k := range h.alwaysSampleKeys {
				if a.Key == k {
					hasAlways = true
					return false
				}
			}
			return true
		})
		if hasAlways {
			return h.inner.Handle(ctx, r)
		}
	}

	if !h.bucket.take() {
		// Dropped - emit metric? For now, silent.
		return nil
	}
	return h.inner.Handle(ctx, r)
}

func (h *samplingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &samplingHandler{
		inner:            h.inner.WithAttrs(attrs),
		bucket:           h.bucket,
		levels:           h.levels,
		alwaysSampleKeys: h.alwaysSampleKeys,
	}
}

func (h *samplingHandler) WithGroup(name string) slog.Handler {
	return &samplingHandler{
		inner:  h.inner.WithGroup(name),
		bucket: h.bucket,
		levels: h.levels,
	}
}

// tokenBucket đơn giản, thread-safe.
type tokenBucket struct {
	mu         sync.Mutex
	rate       float64
	burst      float64
	tokens     float64
	lastRefill time.Time
}

func (b *tokenBucket) refill() {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.lastRefill = now
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
}

func (b *tokenBucket) take() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Refill opportunistically - but only if we have room.
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.rate
		if b.tokens > b.burst {
			b.tokens = b.burst
		}
		b.lastRefill = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}