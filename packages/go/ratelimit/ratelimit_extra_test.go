// Additional tests for ratelimit package.
package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestExtra_Decision_Fields(t *testing.T) {
	d := Decision{
		Allowed:    true,
		Remaining:  5,
		Limit:      10,
		RetryAfter: 1 * time.Second,
	}
	if !d.Allowed {
		t.Error("Allowed mismatch")
	}
	if d.Remaining != 5 {
		t.Errorf("Remaining: got %d", d.Remaining)
	}
}

func TestExtra_TokenBucket_NewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 20)
	if tb == nil {
		t.Fatal("nil")
	}
	if tb.rate != 10 {
		t.Errorf("rate: got %v", tb.rate)
	}
	if tb.burst != 20 {
		t.Errorf("burst: got %v", tb.burst)
	}
}

func TestExtra_TokenBucket_AllowEmptyKey(t *testing.T) {
	tb := NewTokenBucket(10, 20)
	_, err := tb.Allow(context.Background(), "")
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey: got %v", err)
	}
}

func TestExtra_TokenBucket_AllowOK(t *testing.T) {
	tb := NewTokenBucket(100, 100)
	d, err := tb.Allow(context.Background(), "key1")
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if !d.Allowed {
		t.Error("first call should be allowed")
	}
	if d.Limit != 100 {
		t.Errorf("limit: got %d", d.Limit)
	}
}

func TestExtra_TokenBucket_AllowNMultiple(t *testing.T) {
	tb := NewTokenBucket(1, 10)
	d, err := tb.AllowN(context.Background(), "key1", 5)
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if !d.Allowed {
		t.Error("should be allowed")
	}
	if d.Remaining != 5 {
		t.Errorf("Remaining: got %d", d.Remaining)
	}
}

func TestExtra_TokenBucket_BurstExhaustion(t *testing.T) {
	tb := NewTokenBucket(0.001, 5) // very low rate, burst=5
	// Use up the burst
	for i := 0; i < 5; i++ {
		tb.Allow(context.Background(), "key1")
	}
	// Next should be denied
	d, _ := tb.Allow(context.Background(), "key1")
	if d.Allowed {
		t.Error("should be denied after burst")
	}
	if d.RetryAfter == 0 {
		t.Error("RetryAfter should be set")
	}
}

func TestExtra_TokenBucket_DifferentKeys(t *testing.T) {
	tb := NewTokenBucket(1, 2)
	// Use up key1
	tb.AllowN(context.Background(), "key1", 2)
	d, _ := tb.Allow(context.Background(), "key1")
	if d.Allowed {
		t.Error("key1 should be exhausted")
	}
	// key2 should still be allowed
	d, _ = tb.Allow(context.Background(), "key2")
	if !d.Allowed {
		t.Error("key2 should be allowed (separate bucket)")
	}
}

func TestExtra_TokenBucket_RefillOverTime(t *testing.T) {
	// Create with mock clock
	now := time.Now()
	mockNow := func() time.Time { return now }
	tb := &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    10,
		burst:   10,
		ttl:     10 * time.Minute,
		now:     mockNow,
	}
	// Use all tokens
	tb.AllowN(context.Background(), "key1", 10)
	// Move time forward
	now = now.Add(1 * time.Second) // should add 10 tokens
	d, _ := tb.Allow(context.Background(), "key1")
	if !d.Allowed {
		t.Error("should be allowed after 1s with rate=10")
	}
}

func TestExtra_TokenBucket_StartGC(t *testing.T) {
	tb := NewTokenBucket(10, 10)
	stop := tb.StartGC(10 * time.Millisecond)
	if stop == nil {
		t.Fatal("nil stop channel")
	}
	close(stop)
}

func TestExtra_TokenBucket_GC(t *testing.T) {
	tb := &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    10,
		burst:   10,
		ttl:     1 * time.Millisecond,
		now:     time.Now,
	}
	tb.Allow(context.Background(), "key1")
	time.Sleep(10 * time.Millisecond)
	tb.gc()
	if len(tb.buckets) != 0 {
		t.Errorf("expected buckets cleared, got %d", len(tb.buckets))
	}
}

func TestExtra_TokenBucket_AllowN_Negative(t *testing.T) {
	tb := NewTokenBucket(10, 10)
	// n <= 0 should default to 1
	d, _ := tb.AllowN(context.Background(), "key1", 0)
	if !d.Allowed {
		t.Error("should be allowed")
	}
	d, _ = tb.AllowN(context.Background(), "key1", -5)
	if !d.Allowed {
		t.Error("should be allowed with negative n")
	}
}

func TestExtra_ErrLimitExceeded(t *testing.T) {
	if ErrLimitExceeded == nil {
		t.Error("ErrLimitExceeded should not be nil")
	}
}

func TestExtra_ErrInvalidKey(t *testing.T) {
	if ErrInvalidKey == nil {
		t.Error("ErrInvalidKey should not be nil")
	}
}

func TestExtra_FixedWindow_NewFixedWindow(t *testing.T) {
	fw := NewFixedWindow(nil, time.Minute, 100)
	if fw == nil {
		t.Fatal("nil")
	}
	if fw.maxN != 100 {
		t.Errorf("maxN: got %d", fw.maxN)
	}
}

func TestExtra_SlidingWindow_NewSlidingWindow(t *testing.T) {
	sw := NewSlidingWindow(nil, time.Minute, 100)
	if sw == nil {
		t.Fatal("nil")
	}
	if sw.maxN != 100 {
		t.Errorf("maxN: got %d", sw.maxN)
	}
}

func TestExtra_Limiter_InterfaceImplemented(t *testing.T) {
	// TokenBucket implements Limiter
	tb := NewTokenBucket(10, 10)
	var l Limiter = tb
	_, _ = l.Allow(context.Background(), "key1")

	// FixedWindow implements Limiter
	fw := NewFixedWindow(nil, time.Minute, 10)
	var l2 Limiter = fw
	_ = l2

	// SlidingWindow implements Limiter
	sw := NewSlidingWindow(nil, time.Minute, 10)
	var l3 Limiter = sw
	_ = l3
}

func TestExtra_TokenBucket_Concurrent(t *testing.T) {
	tb := NewTokenBucket(1000, 100)
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				tb.Allow(context.Background(), "key1")
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
