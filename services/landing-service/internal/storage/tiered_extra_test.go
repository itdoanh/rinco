// Additional tests for landing-service storage.
package storage

import (
	"testing"
	"time"
)

func TestExtra_DefaultHotAge_Zero(t *testing.T) {
	if got := defaultHotAge(0); got != 30*24*time.Hour {
		t.Errorf("expected 30 days, got %v", got)
	}
}

func TestExtra_DefaultHotAge_Negative(t *testing.T) {
	if got := defaultHotAge(-1 * time.Hour); got != 30*24*time.Hour {
		t.Errorf("expected 30 days for negative, got %v", got)
	}
}

func TestExtra_DefaultHotAge_Custom(t *testing.T) {
	got := defaultHotAge(7 * 24 * time.Hour)
	if got != 7*24*time.Hour {
		t.Errorf("expected 7 days, got %v", got)
	}
}

func TestExtra_DefaultHotAge_OneHour(t *testing.T) {
	got := defaultHotAge(1 * time.Hour)
	if got != 1*time.Hour {
		t.Errorf("expected 1 hour, got %v", got)
	}
}

func TestExtra_Enabled_NilStore(t *testing.T) {
	var s *TieredStore
	if s.Enabled() {
		t.Error("nil store should not be enabled")
	}
}

func TestExtra_Enabled_NoClient(t *testing.T) {
	s := &TieredStore{hot: "hot-bucket"}
	if s.Enabled() {
		t.Error("store without client should not be enabled")
	}
}

func TestExtra_Enabled_NoHotBucket(t *testing.T) {
	// Even with client, no hot bucket = disabled
	// But we can't easily set client here without minio setup
	s := &TieredStore{hot: ""}
	if s.Enabled() {
		t.Error("store without hot bucket should not be enabled")
	}
}

func TestExtra_Bucket_ColdFallback(t *testing.T) {
	s := &TieredStore{hot: "hot", cold: ""}
	if got := s.bucket(true); got != "hot" {
		t.Errorf("cold with no cold bucket should fall back to hot: got %s", got)
	}
}

func TestExtra_Bucket_HotDefault(t *testing.T) {
	s := &TieredStore{hot: "hot", cold: "cold"}
	if got := s.bucket(false); got != "hot" {
		t.Errorf("hot: got %s", got)
	}
}

func TestExtra_Bucket_ColdExplicit(t *testing.T) {
	s := &TieredStore{hot: "hot", cold: "cold"}
	if got := s.bucket(true); got != "cold" {
		t.Errorf("cold: got %s", got)
	}
}

func TestExtra_New_EmptyEndpoint(t *testing.T) {
	// Empty endpoint returns a non-enabled store
	cfg := Config{
		Endpoint:  "",
		HotBucket: "hot-bucket",
		ColdBucket: "cold-bucket",
		MaxHotAge: 0,
	}
	s, err := New(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("nil store")
	}
	if s.Enabled() {
		t.Error("store with empty endpoint should not be enabled")
	}
}

func TestExtra_New_DefaultHotAge(t *testing.T) {
	cfg := Config{
		Endpoint:  "",
		HotBucket: "hot",
		ColdBucket: "cold",
	}
	s, err := New(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if s.maxHotAge != 30*24*time.Hour {
		t.Errorf("expected default 30 days, got %v", s.maxHotAge)
	}
}

func TestExtra_Upload_Disabled(t *testing.T) {
	s := &TieredStore{hot: "hot"} // no client
	_, err := s.Upload(nil, "obj", nil, 0, "text/plain", false)
	if err == nil {
		t.Error("expected error for disabled store")
	}
}

func TestExtra_PresignedGet_Disabled(t *testing.T) {
	s := &TieredStore{hot: "hot"}
	_, err := s.PresignedGet(nil, "obj", false, 0)
	if err == nil {
		t.Error("expected error for disabled store")
	}
}

func TestExtra_Download_Disabled(t *testing.T) {
	s := &TieredStore{hot: "hot"}
	_, err := s.Download(nil, "obj", false)
	if err == nil {
		t.Error("expected error for disabled store")
	}
}

func TestExtra_EnsureBuckets_Disabled(t *testing.T) {
	s := &TieredStore{hot: "hot"}
	if err := s.EnsureBuckets(nil); err != nil {
		t.Errorf("EnsureBuckets on disabled store should be no-op: %v", err)
	}
}

func TestExtra_MigrateOldObjects_NoColdBucket(t *testing.T) {
	s := &TieredStore{hot: "hot", cold: ""}
	if err := s.MigrateOldObjects(nil); err != nil {
		t.Errorf("no cold bucket should be no-op: %v", err)
	}
}

func TestExtra_Config_Fields(t *testing.T) {
	cfg := Config{
		Endpoint:   "localhost:9000",
		AccessKey:  "access",
		SecretKey:  "secret",
		HotBucket:  "hot",
		ColdBucket: "cold",
		UseSSL:     true,
		MaxHotAge:  24 * time.Hour,
	}
	if cfg.Endpoint != "localhost:9000" {
		t.Error("Endpoint mismatch")
	}
	if cfg.MaxHotAge != 24*time.Hour {
		t.Error("MaxHotAge mismatch")
	}
	if !cfg.UseSSL {
		t.Error("UseSSL should be true")
	}
}
