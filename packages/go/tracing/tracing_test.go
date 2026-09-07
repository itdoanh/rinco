package tracing

import (
	"context"
	"testing"
	"time"
)

func TestInitNoEndpoint(t *testing.T) {
	// When OTLPEndpoint is empty, Init should return a no-op shutdown
	// without attempting network I/O.
	ctx := context.Background()
	shutdown, err := Init(ctx, Config{
		ServiceName: "test-service",
		Env:         "development",
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown")
	}
	if err := shutdown(ctx); err != nil {
		t.Errorf("shutdown err: %v", err)
	}
}

func TestInitDevelopmentEnv(t *testing.T) {
	// Env=development skips OTLP entirely
	ctx := context.Background()
	shutdown, err := Init(ctx, Config{
		ServiceName:    "auth",
		ServiceVersion: "1.0.0",
		Env:            "development",
		OTLPEndpoint:   "should-not-be-used:4317",
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown")
	}
}

func TestShutdownIdempotent(t *testing.T) {
	// Calling Shutdown twice should not panic
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := Shutdown(ctx); err != nil {
		t.Errorf("first shutdown: %v", err)
	}
	if err := Shutdown(ctx); err != nil {
		t.Errorf("second shutdown: %v", err)
	}
}

func TestHostname(t *testing.T) {
	h := hostname()
	// Hostname might be empty in some test environments
	if h == "" {
		t.Log("hostname empty (test env), skipping")
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{}
	if cfg.ServiceName != "" {
		t.Error("expected zero-value ServiceName")
	}
	if cfg.SampleRatio != 0 {
		t.Error("expected zero-value SampleRatio")
	}
}
