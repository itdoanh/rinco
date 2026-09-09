// Tests for ratelimit package (ratelimit.go).
package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestExtra2_Decision_Fields(t *testing.T) {
	d := Decision{
		Allowed:    true,
		Remaining:  50,
		Limit:      100,
		RetryAfter: 5 * time.Second,
	}
	if !d.Allowed {
		t.Error("Allowed")
	}
	if d.Remaining != 50 {
		t.Error("Remaining")
	}
	if d.Limit != 100 {
		t.Error("Limit")
	}
	if d.RetryAfter != 5*time.Second {
		t.Error("RetryAfter")
	}
}

func TestExtra2_NewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(100, 200)
	if tb == nil {
		t.Fatal("NewTokenBucket returned nil")
	}
	if tb.rate != 100 {
		t.Errorf("rate: got %f", tb.rate)
	}
	if tb.burst != 200 {
		t.Errorf("burst: got %f", tb.burst)
	}
	if tb.ttl != 10*time.Minute {
		t.Errorf("ttl: got %v", tb.ttl)
	}
}

func TestExtra2_TokenBucket_Allow_First(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	dec, err := tb.Allow(context.Background(), "key1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !dec.Allowed {
		t.Error("first request should be allowed")
	}
	if dec.Remaining != 9 { // burst - 1
		t.Errorf("Remaining: got %d", dec.Remaining)
	}
}

func TestExtra2_TokenBucket_Allow_EmptyKey(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	_, err := tb.Allow(context.Background(), "")
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestExtra2_TokenBucket_AllowN(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	dec, err := tb.AllowN(context.Background(), "key1", 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !dec.Allowed {
		t.Error("request should be allowed")
	}
	if dec.Remaining != 5 { // burst - 5
		t.Errorf("Remaining: got %d", dec.Remaining)
	}
}

func TestExtra2_TokenBucket_AllowN_ExceedsBurst(t *testing.T) {
	tb := NewTokenBucket(100, 5)
	// Exhaust tokens
	for i := 0; i < 5; i++ {
		tb.AllowN(context.Background(), "key1", 1)
	}
	// Should now be rate-limited
	dec, err := tb.AllowN(context.Background(), "key1", 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dec.Allowed {
		t.Error("should not be allowed after burst exhausted")
	}
	if dec.Remaining != 0 {
		t.Errorf("Remaining: got %d", dec.Remaining)
	}
	if dec.RetryAfter <= 0 {
		t.Error("RetryAfter should be positive")
	}
}

func TestExtra2_TokenBucket_AllowN_NegativeN(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	// Negative n should be treated as 1
	dec, err := tb.AllowN(context.Background(), "key1", -5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !dec.Allowed {
		t.Error("negative n should allow 1 request")
	}
}

func TestExtra2_TokenBucket_AllowN_ZeroN(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	dec, err := tb.AllowN(context.Background(), "key1", 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !dec.Allowed {
		t.Error("zero n should allow 1 request")
	}
}

func TestExtra2_TokenBucket_DifferentKeys(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	dec1, _ := tb.AllowN(context.Background(), "key1", 5)
	dec2, _ := tb.AllowN(context.Background(), "key2", 5)
	// Both should have 5 tokens used from their own buckets
	if dec1.Remaining != 5 {
		t.Errorf("key1 remaining: got %d", dec1.Remaining)
	}
	if dec2.Remaining != 5 {
		t.Errorf("key2 remaining: got %d", dec2.Remaining)
	}
}

func TestExtra2_TokenBucket_Refill(t *testing.T) {
	now := time.Now()
	tb := &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    100, // 100 tokens per second
		burst:   10,
		now:     func() time.Time { return now },
	}
	
	// Use all tokens
	tb.AllowN(context.Background(), "key1", 10)
	
	// Advance time by 1 second
	tb.now = func() time.Time { return now.Add(1 * time.Second) }
	
	// Should have refilled
	dec, _ := tb.AllowN(context.Background(), "key1", 1)
	if dec.Remaining != 9 { // 10 refilled - 1 used
		t.Errorf("Expected 9 remaining, got %d", dec.Remaining)
	}
}

func TestExtra2_TokenBucket_GC(t *testing.T) {
	now := time.Now()
	tb := &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    100,
		burst:   10,
		ttl:     1 * time.Second,
		now:     func() time.Time { return now },
	}
	
	tb.AllowN(context.Background(), "old-key", 1)
	
	// Advance time past TTL
	tb.now = func() time.Time { return now.Add(2 * time.Second) }
	
	initial := len(tb.buckets)
	tb.gc()
	
	if len(tb.buckets) >= initial {
		t.Error("gc should have removed old bucket")
	}
}

func TestExtra2_TokenBucket_StartGC(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	stop := tb.StartGC(100 * time.Millisecond)
	
	// Let it run a bit
	time.Sleep(250 * time.Millisecond)
	
	// Stop the GC
	close(stop)
}

func TestExtra2_bucket_Fields(t *testing.T) {
	now := time.Now()
	b := &bucket{
		tokens:     5.0,
		lastRefill: now,
		lastSeen:   now,
	}
	if b.tokens != 5.0 {
		t.Error("tokens")
	}
	if b.lastRefill != now {
		t.Error("lastRefill")
	}
}

func TestExtra2_TokenBucket_Limit(t *testing.T) {
	tb := NewTokenBucket(100, 50)
	dec, _ := tb.AllowN(context.Background(), "key1", 1)
	if dec.Limit != 50 {
		t.Errorf("Limit: got %d", dec.Limit)
	}
}

func TestExtra2_TokenBucket_AllowN_PartialTokens(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	// Use 8 tokens
	tb.AllowN(context.Background(), "key1", 8)
	
	// Try to use 5 more (only 2 left)
	dec, err := tb.AllowN(context.Background(), "key1", 5)
	if err != nil {
		t.Errorf("err: %v", err)
	}
	// Should use 2 tokens (exhaust), then wait for 3
	if dec.Remaining != 0 {
		t.Errorf("Remaining: got %d", dec.Remaining)
	}
	if dec.Allowed {
		t.Error("should not allow exceeding tokens")
	}
}

func TestExtra2_Limiter_Interface(t *testing.T) {
	// Verify TokenBucket implements Limiter
	var _ Limiter = NewTokenBucket(100, 10)
}

func TestExtra2_Errors(t *testing.T) {
	if ErrLimitExceeded.Error() == "" {
		t.Error("ErrLimitExceeded should have message")
	}
	if ErrInvalidKey.Error() == "" {
		t.Error("ErrInvalidKey should have message")
	}
}
