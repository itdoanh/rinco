package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketAllow(t *testing.T) {
	tb := NewTokenBucket(100, 5) // 100 req/s, burst 5

	// First 5 should pass
	for i := 0; i < 5; i++ {
		dec, err := tb.Allow(context.Background(), "k1")
		if err != nil {
			t.Fatalf("Allow err: %v", err)
		}
		if !dec.Allowed {
			t.Errorf("iter %d: expected allowed", i)
		}
	}

	// 6th should be denied (burst exhausted)
	dec, err := tb.Allow(context.Background(), "k1")
	if err != nil {
		t.Fatalf("Allow err: %v", err)
	}
	if dec.Allowed {
		t.Errorf("expected denied after burst exhausted")
	}
	if dec.RetryAfter <= 0 {
		t.Errorf("retry-after should be > 0, got %v", dec.RetryAfter)
	}
}

func TestTokenBucketRefill(t *testing.T) {
	// Use a synthetic clock to test refill deterministically
	now := time.Now()
	tb := NewTokenBucket(10, 1) // 10 req/s, burst 1
	tb.now = func() time.Time { return now }

	dec, _ := tb.Allow(context.Background(), "k")
	if !dec.Allowed {
		t.Fatal("first should be allowed")
	}

	// Advance clock by 100ms — bucket should refill 1 token
	now = now.Add(100 * time.Millisecond)
	dec, _ = tb.Allow(context.Background(), "k")
	if !dec.Allowed {
		t.Errorf("after 100ms, expected allowed (refilled)")
	}
}

func TestTokenBucketInvalidKey(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	_, err := tb.Allow(context.Background(), "")
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestTokenBucketAllowN(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	dec, err := tb.AllowN(context.Background(), "k", 3)
	if err != nil {
		t.Fatalf("AllowN err: %v", err)
	}
	if !dec.Allowed {
		t.Error("3 tokens from burst=5 should be allowed")
	}
	if dec.Remaining != 2 {
		t.Errorf("expected 2 remaining, got %d", dec.Remaining)
	}

	// Ask for 3 more — would exceed 5
	dec, _ = tb.AllowN(context.Background(), "k", 3)
	if dec.Allowed {
		t.Error("6 total should be denied")
	}
}

func TestTokenBucketGC(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	tb.ttl = 10 * time.Millisecond

	tb.Allow(context.Background(), "k")
	if len(tb.buckets) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(tb.buckets))
	}

	// Wait past TTL then GC
	time.Sleep(20 * time.Millisecond)
	tb.gc()
	if len(tb.buckets) != 0 {
		t.Errorf("expected 0 buckets after GC, got %d", len(tb.buckets))
	}
}

func TestDecisionStructure(t *testing.T) {
	dec := Decision{Allowed: true, Remaining: 5, Limit: 10, RetryAfter: 0}
	if !dec.Allowed || dec.Remaining != 5 || dec.Limit != 10 {
		t.Errorf("Decision struct mismatch: %+v", dec)
	}
}
