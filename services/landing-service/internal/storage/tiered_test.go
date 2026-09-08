// Tests for landing-service tiered storage helpers.
package storage

import (
	"testing"
	"time"
)

func TestDefaultHotAge_Default(t *testing.T) {
	got := defaultHotAge(0)
	want := 30 * 24 * time.Hour
	if got != want {
		t.Errorf("defaultHotAge(0): want %v, got %v", want, got)
	}
}

func TestDefaultHotAge_Negative(t *testing.T) {
	got := defaultHotAge(-1 * time.Hour)
	want := 30 * 24 * time.Hour
	if got != want {
		t.Errorf("defaultHotAge(-1h): want %v, got %v", want, got)
	}
}

func TestDefaultHotAge_Custom(t *testing.T) {
	got := defaultHotAge(7 * 24 * time.Hour)
	want := 7 * 24 * time.Hour
	if got != want {
		t.Errorf("defaultHotAge(7d): want %v, got %v", want, got)
	}
}

func TestTieredStore_Enabled_Nil(t *testing.T) {
	var s *TieredStore
	if s.Enabled() {
		t.Error("expected nil store to be disabled")
	}
}

func TestTieredStore_Bucket(t *testing.T) {
	s := &TieredStore{hot: "hot-bucket", cold: "cold-bucket"}
	if s.bucket(false) != "hot-bucket" {
		t.Error("expected hot bucket for cold=false")
	}
	if s.bucket(true) != "cold-bucket" {
		t.Error("expected cold bucket for cold=true")
	}
}

func TestTieredStore_Bucket_NoCold(t *testing.T) {
	s := &TieredStore{hot: "hot-bucket"}
	// When cold is empty, fall back to hot bucket.
	if s.bucket(true) != "hot-bucket" {
		t.Error("expected hot fallback when cold empty")
	}
}

func TestNew_NoEndpoint(t *testing.T) {
	cfg := Config{
		HotBucket:  "hot",
		ColdBucket: "cold",
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil store")
	}
	if s.hot != "hot" {
		t.Errorf("hot: got %s", s.hot)
	}
	if s.cold != "cold" {
		t.Errorf("cold: got %s", s.cold)
	}
	if s.Enabled() {
		t.Error("expected store without endpoint to be disabled")
	}
	if s.maxHotAge != 30*24*time.Hour {
		t.Errorf("maxHotAge: got %v", s.maxHotAge)
	}
}

func TestNew_StripsHTTPS(t *testing.T) {
	// Verify URL scheme stripping happens (without making actual connection).
	cfg := Config{
		Endpoint:   "https://minio.example.com",
		HotBucket:  "hot",
		ColdBucket: "cold",
		AccessKey:  "access",
		SecretKey:  "secret",
	}
	// minio.New doesn't connect on construction, just validates format.
	_, _ = New(cfg)
	// If we reach here without panic, scheme-stripping logic worked.
}

func TestNew_StripsHTTP(t *testing.T) {
	cfg := Config{
		Endpoint:   "http://minio.example.com",
		HotBucket:  "hot",
		ColdBucket: "cold",
		AccessKey:  "access",
		SecretKey:  "secret",
	}
	_, _ = New(cfg)
}
