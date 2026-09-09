package tracing

import (
	"context"
	"testing"
	"time"
)

func TestInitNoEndpointDev(t *testing.T) {
	cfg := Config{
		ServiceName: "test-svc",
		Env:         "development",
	}
	shutdown, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown err: %v", err)
	}
}

func TestInitNoEndpointEmpty(t *testing.T) {
	cfg := Config{
		ServiceName: "test-svc",
		Env:         "production",
		// OTLPEndpoint empty -> noop
	}
	shutdown, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown nil")
	}
}

func TestShutdownWithoutInit(t *testing.T) {
	// Should not panic even if Init was never called.
	globalShutdown = nil
	if err := Shutdown(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShutdownMultipleTimes(t *testing.T) {
	cfg := Config{
		ServiceName: "test-svc",
		Env:         "development",
	}
	shutdown, _ := Init(context.Background(), cfg)
	// Call shutdown multiple times — should be idempotent
	for i := 0; i < 3; i++ {
		if err := shutdown(context.Background()); err != nil {
			t.Errorf("call %d: %v", i, err)
		}
	}
}

func TestHostnameGet(t *testing.T) {
	h := hostname()
	if h == "" {
		t.Error("hostname empty")
	}
}

func TestInitValidContext(t *testing.T) {
	cfg := Config{
		ServiceName: "svc",
		Env:         "development",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	shutdown, err := Init(ctx, cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if shutdown == nil {
		t.Fatal("nil shutdown")
	}
}

func TestShutdownFuncsAdd(t *testing.T) {
	s := &shutdownFuncs{}
	called := 0
	s.addShutdown(func(ctx context.Context) error {
		called++
		return nil
	})
	s.addShutdown(func(ctx context.Context) error {
		called++
		return nil
	})
	if len(s.funcs) != 2 {
		t.Errorf("funcs len = %d, want 2", len(s.funcs))
	}
	// shutdown should not be called yet
	_ = s.shutdown(context.Background())
	if called != 2 {
		t.Errorf("shutdown called: %d", called)
	}
}

func TestShutdownFuncsOnce(t *testing.T) {
	s := &shutdownFuncs{}
	called := 0
	s.addShutdown(func(ctx context.Context) error {
		called++
		return nil
	})
	s.shutdown(context.Background())
	s.shutdown(context.Background())
	if called != 1 {
		t.Errorf("called = %d, want 1", called)
	}
}

func TestConfigDefaultsX(t *testing.T) {
	cfg := Config{}
	// Just check zero values are valid
	if cfg.ServiceName != "" {
		t.Errorf("expected empty, got %q", cfg.ServiceName)
	}
	if cfg.SampleRatio != 0.0 {
		t.Errorf("ratio = %f", cfg.SampleRatio)
	}
}
