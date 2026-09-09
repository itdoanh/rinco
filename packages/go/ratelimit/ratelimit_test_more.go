package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestErrors(t *testing.T) {
	if ErrLimitExceeded == nil {
		t.Error("ErrLimitExceeded nil")
	}
	if ErrInvalidKey == nil {
		t.Error("ErrInvalidKey nil")
	}
}

func TestDecisionFields(t *testing.T) {
	d := Decision{
		Allowed:    true,
		Remaining:  10,
		Limit:      20,
		RetryAfter: 100 * time.Millisecond,
	}
	if !d.Allowed {
		t.Error("allowed should be true")
	}
	if d.Remaining != 10 {
		t.Errorf("remaining = %d", d.Remaining)
	}
	if d.Limit != 20 {
		t.Errorf("limit = %d", d.Limit)
	}
}

func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10.0, 20)
	if tb == nil {
		t.Fatal("nil")
	}
	if tb.rate != 10.0 {
		t.Errorf("rate = %f", tb.rate)
	}
	if tb.burst != 20.0 {
		t.Errorf("burst = %f", tb.burst)
	}
}

func TestTokenBucketAllowSuccess(t *testing.T) {
	tb := NewTokenBucket(10.0, 5)
	d, err := tb.Allow(context.Background(), "key1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !d.Allowed {
		t.Error("should be allowed")
	}
	if d.Limit != 5 {
		t.Errorf("limit = %d, want 5", d.Limit)
	}
}

func TestTokenBucketAllowBurstLimit(t *testing.T) {
	tb := NewTokenBucket(0.001, 3) // very slow refill, burst=3
	allowed := 0
	for i := 0; i < 10; i++ {
		d, _ := tb.Allow(context.Background(), "burst-test")
		if d.Allowed {
			allowed++
		}
	}
	if allowed != 3 {
		t.Errorf("expected 3 allowed, got %d", allowed)
	}
}

func TestTokenBucketInvalidKeyX(t *testing.T) {
	tb := NewTokenBucket(10, 10)
	_, err := tb.Allow(context.Background(), "")
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestTokenBucketAllowNX(t *testing.T) {
	tb := NewTokenBucket(10, 10)
	d, err := tb.AllowN(context.Background(), "key", 5)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !d.Allowed {
		t.Error("5 should be allowed within burst=10")
	}
}

func TestTokenBucketAllowNDeny(t *testing.T) {
	tb := NewTokenBucket(0.001, 5)
	d, err := tb.AllowN(context.Background(), "key", 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if d.Allowed {
		t.Error("10 > burst=5 should not be allowed")
	}
	if d.RetryAfter <= 0 {
		t.Error("retryAfter should be > 0")
	}
}

func TestTokenBucketAllowNZero(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	// n=0 should default to 1
	d, err := tb.AllowN(context.Background(), "key", 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !d.Allowed {
		t.Error("should be allowed (n defaults to 1)")
	}
}

func TestTokenBucketAllowNNegative(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	d, err := tb.AllowN(context.Background(), "key", -3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !d.Allowed {
		t.Error("should be allowed (negative defaults to 1)")
	}
}

func TestTokenBucketRefillX(t *testing.T) {
	// Use fake clock by manipulating now
	tb := NewTokenBucket(1000, 5) // 1000 tokens/sec refill
	// Drain the bucket
	for i := 0; i < 5; i++ {
		tb.Allow(context.Background(), "key")
	}
	// Next call should be denied
	d, _ := tb.Allow(context.Background(), "key")
	if d.Allowed {
		t.Error("should be denied after burst")
	}
	// Wait briefly for refill
	time.Sleep(50 * time.Millisecond)
	d, _ = tb.Allow(context.Background(), "key")
	// With 1000/sec refill and 50ms wait, should refill ~50 tokens
	if !d.Allowed {
		t.Error("should refill quickly")
	}
}

func TestTokenBucketStartGC(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	stop := tb.StartGC(100 * time.Millisecond)
	if stop == nil {
		t.Fatal("stop chan nil")
	}
	close(stop)
}

func TestTokenBucketMultipleKeys(t *testing.T) {
	tb := NewTokenBucket(10, 2)
	d1, _ := tb.Allow(context.Background(), "user1")
	d2, _ := tb.Allow(context.Background(), "user2")
	if !d1.Allowed || !d2.Allowed {
		t.Error("different keys should be independent")
	}
}

func TestTokenBucketRefillZero(t *testing.T) {
	// rate=0 means no refill
	tb := NewTokenBucket(0, 5)
	for i := 0; i < 5; i++ {
		d, _ := tb.Allow(context.Background(), "key")
		if !d.Allowed {
			t.Errorf("call %d should be allowed", i)
		}
	}
	d, _ := tb.Allow(context.Background(), "key")
	if d.Allowed {
		t.Error("6th call should be denied (rate=0)")
	}
}
