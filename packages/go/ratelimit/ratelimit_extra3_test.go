// Extra tests for ratelimit package (TokenBucket only - in-memory).
package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket_Allow_First(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	d, err := tb.Allow(context.Background(), "key1")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allowed {
		t.Error("first request should be allowed")
	}
	if d.Limit != 5 {
		t.Errorf("limit: %d", d.Limit)
	}
}

func TestTokenBucket_Allow_EmptyKey(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	_, err := tb.Allow(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestTokenBucket_AllowN_EmptyKey(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	_, err := tb.AllowN(context.Background(), "", 1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestTokenBucket_Burst(t *testing.T) {
	tb := NewTokenBucket(0.5, 3)
	// First 3 should be allowed (burst), 4th should be denied
	for i := 0; i < 3; i++ {
		d, err := tb.Allow(context.Background(), "burst")
		if err != nil {
			t.Fatal(err)
		}
		if !d.Allowed {
			t.Errorf("req %d should be allowed", i)
		}
	}
	d, err := tb.Allow(context.Background(), "burst")
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed {
		t.Error("4th should be denied")
	}
	if d.RetryAfter <= 0 {
		t.Error("retry-after should be set")
	}
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	tb := NewTokenBucket(1000, 1) // 1000 req/s, burst 1
	// First allowed
	d, _ := tb.Allow(context.Background(), "refill")
	if !d.Allowed {
		t.Fatal("first")
	}
	// Immediate second denied
	d, _ = tb.Allow(context.Background(), "refill")
	if d.Allowed {
		t.Fatal("second should be denied")
	}
	// Wait for refill
	time.Sleep(10 * time.Millisecond)
	d, _ = tb.Allow(context.Background(), "refill")
	if !d.Allowed {
		t.Error("after wait should be allowed")
	}
}

func TestTokenBucket_MultipleKeys(t *testing.T) {
	tb := NewTokenBucket(0.5, 1)
	d1, _ := tb.Allow(context.Background(), "user1")
	d2, _ := tb.Allow(context.Background(), "user2")
	if !d1.Allowed || !d2.Allowed {
		t.Error("both should be allowed - different keys")
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	d, err := tb.AllowN(context.Background(), "key", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allowed {
		t.Error("5 of 10 should be allowed")
	}
	d, _ = tb.AllowN(context.Background(), "key", 10)
	if d.Allowed {
		t.Error("10 more should be denied (only 5 left)")
	}
}

func TestTokenBucket_AllowN_Negative(t *testing.T) {
	tb := NewTokenBucket(100, 10)
	d, err := tb.AllowN(context.Background(), "key", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allowed {
		t.Error("n=0 should be allowed (defaults to 1)")
	}
}

func TestTokenBucket_StartGC(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	stop := tb.StartGC(50 * time.Millisecond)
	if stop == nil {
		t.Fatal("nil channel")
	}
	// Create a bucket
	tb.Allow(context.Background(), "ephemeral")
	time.Sleep(100 * time.Millisecond)
	close(stop)
}

func TestTokenBucket_RefillZero(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	tb.now = func() time.Time { return time.Time{} }
	b := &bucket{tokens: 0, lastRefill: time.Time{}}
	tb.refill(b)
	// With same time, elapsed is 0, no refill
	if b.tokens != 0 {
		t.Errorf("tokens: %f", b.tokens)
	}
}

func TestTokenBucket_Remaining(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	d, _ := tb.Allow(context.Background(), "rem")
	if d.Remaining < 0 {
		t.Error("remaining should be >= 0")
	}
}

func TestTokenBucket_GC_RemovesIdle(t *testing.T) {
	tb := NewTokenBucket(10, 1)
	tb.ttl = 50 * time.Millisecond
	tb.Allow(context.Background(), "stale")
	// bucket exists
	if _, ok := tb.buckets["stale"]; !ok {
		t.Fatal("should exist")
	}
	// simulate GC after long wait
	old := tb.now
	tb.now = func() time.Time { return time.Now().Add(1 * time.Hour) }
	defer func() { tb.now = old }()
	tb.gc()
	if _, ok := tb.buckets["stale"]; ok {
		t.Error("stale should be gc'd")
	}
}

func TestErrorsEx(t *testing.T) {
	if ErrLimitExceeded == nil {
		t.Error("nil error")
	}
	if ErrInvalidKey == nil {
		t.Error("nil error")
	}
}

func TestDecision_Struct(t *testing.T) {
	d := Decision{
		Allowed:    true,
		Remaining:  5,
		Limit:      10,
		RetryAfter: time.Second,
	}
	if !d.Allowed {
		t.Error("allowed")
	}
	if d.Remaining != 5 {
		t.Errorf("remaining: %d", d.Remaining)
	}
}
