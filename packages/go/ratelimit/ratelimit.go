// Package ratelimit provides in-memory and Redis-backed rate limiters:
// token bucket, fixed window, and sliding window. Suitable for HTTP middleware,
// per-tenant quota enforcement, and per-API-key throttling.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ===== Errors =====

var (
	ErrLimitExceeded = errors.New("ratelimit: limit exceeded")
	ErrInvalidKey    = errors.New("ratelimit: invalid key")
)

// Decision is the outcome of an Allow check.
type Decision struct {
	Allowed    bool
	Remaining  int           // tokens remaining in the current window
	Limit      int           // bucket capacity
	RetryAfter time.Duration // suggested wait before retry when not allowed
}

// ===== Limiter interface =====

// Limiter is the contract implemented by every strategy.
type Limiter interface {
	Allow(ctx context.Context, key string) (Decision, error)
	AllowN(ctx context.Context, key string, n int) (Decision, error)
}

// ===== Token Bucket (in-memory) =====

// TokenBucket implements an in-memory token bucket per key.
type TokenBucket struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
	ttl     time.Duration
	now     func() time.Time
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

// NewTokenBucket creates a token bucket limiter.
// rate = tokens per second (e.g. 100 means 100 req/s).
// burst = max bucket size (e.g. 200 = max 200 req instantaneous).
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   float64(burst),
		ttl:     10 * time.Minute,
		now:     time.Now,
	}
}

func (t *TokenBucket) refill(b *bucket) {
	now := t.now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	b.tokens += elapsed * t.rate
	if b.tokens > t.burst {
		b.tokens = t.burst
	}
	b.lastRefill = now
}

// Allow consumes one token for the key.
func (t *TokenBucket) Allow(_ context.Context, key string) (Decision, error) {
	return t.AllowN(context.Background(), key, 1)
}

// AllowN consumes N tokens for the key.
func (t *TokenBucket) AllowN(_ context.Context, key string, n int) (Decision, error) {
	if key == "" {
		return Decision{}, ErrInvalidKey
	}
	if n <= 0 {
		n = 1
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	b, ok := t.buckets[key]
	if !ok {
		b = &bucket{tokens: t.burst, lastRefill: t.now()}
		t.buckets[key] = b
	}
	t.refill(b)
	b.lastSeen = t.now()

	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return Decision{
			Allowed:   true,
			Remaining: int(b.tokens),
			Limit:     int(t.burst),
		}, nil
	}
	// Compute retry-after: time to accrue n tokens
	deficit := float64(n) - b.tokens
	wait := time.Duration(deficit / t.rate * float64(time.Second))
	return Decision{
		Allowed:    false,
		Remaining:  0,
		Limit:      int(t.burst),
		RetryAfter: wait,
	}, nil
}

// gc removes idle buckets.
func (t *TokenBucket) gc() {
	t.mu.Lock()
	defer t.mu.Unlock()
	cutoff := t.now().Add(-t.ttl)
	for k, b := range t.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(t.buckets, k)
		}
	}
}

// StartGC starts a goroutine that periodically cleans idle buckets.
func (t *TokenBucket) StartGC(interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				t.gc()
			}
		}
	}()
	return stop
}

// ===== Fixed Window (Redis) =====

// FixedWindow implements a per-key fixed window counter in Redis.
type FixedWindow struct {
	rdb     *redis.Client
	prefix  string
	window  time.Duration
	maxN    int
	nowFunc func() time.Time
}

// NewFixedWindow creates a fixed-window limiter.
// window = bucket size (e.g. 1 minute). max = max requests per window.
func NewFixedWindow(rdb *redis.Client, window time.Duration, max int) *FixedWindow {
	return &FixedWindow{
		rdb:     rdb,
		prefix:  "ratelimit:fw:",
		window:  window,
		maxN:    max,
		nowFunc: time.Now,
	}
}

// Allow consumes 1 request from the key's window.
func (f *FixedWindow) Allow(ctx context.Context, key string) (Decision, error) {
	return f.AllowN(ctx, key, 1)
}

// AllowN consumes N requests.
func (f *FixedWindow) AllowN(ctx context.Context, key string, n int) (Decision, error) {
	if key == "" {
		return Decision{}, ErrInvalidKey
	}
	if n <= 0 {
		n = 1
	}
	now := f.nowFunc()
	bucket := now.Truncate(f.window).Unix()
	rkey := fmt.Sprintf("%s%s:%d", f.prefix, key, bucket)

	count, err := f.rdb.IncrBy(ctx, rkey, int64(n)).Result()
	if err != nil {
		return Decision{}, fmt.Errorf("ratelimit: redis incr: %w", err)
	}
	// Set expiry on first hit
	if count == int64(n) {
		f.rdb.Expire(ctx, rkey, f.window+time.Second)
	}

	remaining := f.maxN - int(count)
	if remaining < 0 {
		remaining = 0
	}
	allowed := int(count) <= f.maxN
	retryAfter := time.Duration(0)
	if !allowed {
		// Compute retry-after as Duration
		windowSec := int64(f.window / time.Second)
		nextBucket := bucket + windowSec
		retryAfterSec := nextBucket - now.Unix()
		if retryAfterSec < 0 {
			retryAfterSec = 0
		}
		retryAfter = time.Duration(retryAfterSec) * time.Second
	}
	return Decision{
		Allowed:    allowed,
		Remaining:  remaining,
		Limit:      f.maxN,
		RetryAfter: retryAfter,
	}, nil
}

// ===== Sliding Window (Redis) =====

// SlidingWindow implements a sliding window counter using Redis sorted sets.
type SlidingWindow struct {
	rdb    *redis.Client
	prefix string
	window time.Duration
	maxN   int
}

// NewSlidingWindow creates a sliding window limiter.
func NewSlidingWindow(rdb *redis.Client, window time.Duration, max int) *SlidingWindow {
	return &SlidingWindow{
		rdb:    rdb,
		prefix: "ratelimit:sw:",
		window: window,
		maxN:   max,
	}
}

// Allow consumes 1 request.
func (s *SlidingWindow) Allow(ctx context.Context, key string) (Decision, error) {
	return s.AllowN(ctx, key, 1)
}

// AllowN consumes N requests using sliding-window-log algorithm.
func (s *SlidingWindow) AllowN(ctx context.Context, key string, n int) (Decision, error) {
	if key == "" {
		return Decision{}, ErrInvalidKey
	}
	if n <= 0 {
		n = 1
	}
	rkey := fmt.Sprintf("%s%s", s.prefix, key)
	now := time.Now().UnixNano()
	cutoff := now - s.window.Nanoseconds()

	pipe := s.rdb.TxPipeline()
	pipe.ZRemRangeByScore(ctx, rkey, "-inf", fmt.Sprintf("%d", cutoff))
	countCmd := pipe.ZCard(ctx, rkey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return Decision{}, fmt.Errorf("ratelimit: zcard: %w", err)
	}
	current := int(countCmd.Val())
	if current+n > s.maxN {
		// Find oldest entry to compute retry-after
		oldest, err := s.rdb.ZRangeWithScores(ctx, rkey, 0, 0).Result()
		retry := time.Duration(0)
		if err == nil && len(oldest) > 0 {
			oldestNs := int64(oldest[0].Score)
			retry = time.Duration(oldestNs+s.window.Nanoseconds()-now) * time.Nanosecond
			if retry < 0 {
				retry = 0
			}
		}
		return Decision{
			Allowed:    false,
			Remaining:  0,
			Limit:      s.maxN,
			RetryAfter: retry,
		}, nil
	}
	// Add N entries
	pipe = s.rdb.TxPipeline()
	for i := 0; i < n; i++ {
		pipe.ZAdd(ctx, rkey, redis.Z{Score: float64(now + int64(i)), Member: fmt.Sprintf("%d-%d", now, i)})
	}
	pipe.Expire(ctx, rkey, s.window+time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return Decision{}, fmt.Errorf("ratelimit: zadd: %w", err)
	}
	return Decision{
		Allowed:   true,
		Remaining: s.maxN - current - n,
		Limit:     s.maxN,
	}, nil
}