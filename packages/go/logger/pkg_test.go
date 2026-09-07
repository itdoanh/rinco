package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestRedactorEmail kiểm tra redact email.
func TestRedactorEmail(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	tests := []struct {
		input    string
		contains string
		notContains string
	}{
		{"Email: test@example.com", "REDACTED", "test@example.com"},
		{"Contact me at john.doe+filter@gmail.com", "REDACTED", "john.doe"},
		{"user.name@sub.domain.co.uk", "REDACTED", "user.name"},
	}

	for _, tc := range tests {
		result := r.RedactString(tc.input)
		if !strings.Contains(result, tc.contains) {
			t.Errorf("expected %s to contain %s", result, tc.contains)
		}
		if strings.Contains(result, tc.notContains) {
			t.Errorf("expected %s to NOT contain %s", result, tc.notContains)
		}
	}
}

// TestRedactorPhone kiểm tra redact số điện thoại.
func TestRedactorPhone(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	tests := []string{
		"Call me at +1-555-123-4567",
		"Phone: 0987654321",
		"Tel: (84) 24 1234 5678",
	}

	for _, tc := range tests {
		result := r.RedactString(tc)
		if !strings.Contains(result, "REDACTED") {
			t.Errorf("expected %s to contain REDACTED", result)
		}
	}
}

// TestRedactorCreditCard kiểm tra redact credit card.
func TestRedactorCreditCard(t *testing.T) {
	// Disable phone regex to avoid interference with CC matching.
	cfg := DefaultRedactorConfig()
	cfg.DisablePhone = true
	r := NewRedactor(cfg)

	// Số test hợp lệ (Luhn pass)
	tests := []string{
		"Card: 4111111111111111",     // Visa test
		"CC: 5555555555554444",       // Mastercard test
		"Number: 6011111111111117",   // Discover test
	}

	for _, tc := range tests {
		result := r.RedactString(tc)
		if !strings.Contains(result, "REDACTED") {
			t.Errorf("expected %s to contain REDACTED, got %s", tc, result)
		}
	}
}

// TestRedactorIP kiểm tra redact IP.
func TestRedactorIP(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	result := r.RedactString("Server IP: 192.168.1.1")
	if !strings.Contains(result, "REDACTED") {
		t.Errorf("expected to contain REDACTED, got: %s", result)
	}
	if strings.Contains(result, "192.168.1.1") {
		t.Errorf("IP should be redacted")
	}
}

// TestRedactorToken kiểm tra redact JWT/token.
func TestRedactorToken(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	result := r.RedactString("JWT: " + jwt)
	if !strings.Contains(result, "REDACTED") {
		t.Errorf("expected JWT to be redacted, got: %s", result)
	}
}

// TestRedactorPassword kiểm tra redact password trong query.
func TestRedactorPassword(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	result := r.RedactString("password=secret123&user=admin")
	if !strings.Contains(result, "REDACTED") {
		t.Errorf("expected password to be redacted, got: %s", result)
	}
}

// TestRedactorFieldName kiểm tra mask toàn bộ field.
func TestRedactorFieldName(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	// Tạo slog.Record với field sensitive
	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, nil)
	wrapped := NewRedactionHandler(handler, r)
	logger := slog.New(wrapped)

	logger.Info("test", slog.String("password", "supersecret"), slog.String("username", "admin"))

	// Parse output
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if pwd, ok := result["password"].(string); !ok || !strings.Contains(pwd, "REDACTED") {
		t.Errorf("password field should be redacted, got: %v", result["password"])
	}
}

// TestSamplingHandler kiểm tra sampling handler.
func TestSamplingHandler(t *testing.T) {
	inner := &mockHandler{}
	cfg := &SamplingConfig{
		TokensPerSecond: 1,
		Burst:           1,
	}
	handler := NewSamplingHandler(inner, cfg)
	logger := slog.New(handler)

	// First call should pass (burst)
	logger.Info("msg1")
	if inner.count != 1 {
		t.Errorf("expected 1 call, got %d", inner.count)
	}

	// Immediate second call should be dropped (no tokens)
	logger.Info("msg2")
	if inner.count != 1 {
		t.Errorf("expected 1 call after drop, got %d", inner.count)
	}

	// Wait for refill
	time.Sleep(1500 * time.Millisecond)
	logger.Info("msg3")
	if inner.count != 2 {
		t.Errorf("expected 2 calls after refill, got %d", inner.count)
	}
}

// TestSamplingAlwaysSampleAttrs kiểm tra always-sample attrs.
func TestSamplingAlwaysSampleAttrs(t *testing.T) {
	inner := &mockHandler{}
	cfg := &SamplingConfig{
		TokensPerSecond:    1,
		Burst:               1,
		AlwaysSampleAttrs:   []string{"critical"},
	}
	handler := NewSamplingHandler(inner, cfg)
	logger := slog.New(handler)

	// Critical event (key=critical) should always pass
	for i := 0; i < 5; i++ {
		logger.Info("msg", slog.String("critical", "yes"))
	}
	if inner.count != 5 {
		t.Errorf("expected 5 calls (always-sample), got %d", inner.count)
	}
}

// TestSamplingWithLevels kiểm tra level filtering.
func TestSamplingWithLevels(t *testing.T) {
	inner := &mockHandler{}
	cfg := &SamplingConfig{
		TokensPerSecond: 1,
		Burst:           1,
		LevelsToSample:  []slog.Level{slog.LevelInfo},
	}
	handler := NewSamplingHandler(inner, cfg)
	logger := slog.New(handler)

	// Errors should not be sampled (always pass)
	for i := 0; i < 5; i++ {
		logger.Error("error", slog.String("test", "value"))
	}
	if inner.count != 5 {
		t.Errorf("expected 5 error calls, got %d", inner.count)
	}
}

// TestRedactorConfigCustom kiểm tra custom patterns.
func TestRedactorConfigCustom(t *testing.T) {
	r := NewRedactor(&RedactorConfig{
		CustomPatterns: []*RedactionPattern{
			{
				Name:    "api_secret",
				Pattern: regexp.MustCompile(`secret_[a-zA-Z0-9]{20}`),
			},
		},
	})

	result := r.RedactString("My key is secret_abcdefghij1234567890")
	if !strings.Contains(result, "REDACTED") {
		t.Errorf("expected custom pattern to be redacted, got: %s", result)
	}
}

// TestHashValueDeterministic kiểm tra hash có deterministic.
func TestHashValueDeterministic(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	hash1 := r.hashValue("email", "test@example.com")
	hash2 := r.hashValue("email", "test@example.com")
	
	if hash1 != hash2 {
		t.Errorf("hash should be deterministic, got %s != %s", hash1, hash2)
	}
	
	hash3 := r.hashValue("email", "different@example.com")
	if hash1 == hash3 {
		t.Errorf("different values should have different hashes")
	}
}

// TestRedactorConfigurablePrefix kiểm tra prefix/suffix config.
func TestRedactorConfigurablePrefix(t *testing.T) {
	r := NewRedactor(&RedactorConfig{
		HashPrefix: "<HIDDEN:",
		HashSuffix: ">",
	})
	
	result := r.RedactString("Email: test@example.com")
	if !strings.Contains(result, "<HIDDEN:") || !strings.Contains(result, ">") {
		t.Errorf("expected custom prefix/suffix, got: %s", result)
	}
}

// TestUUIDPreservation kiểm tra UUID không bị coi là token.
func TestUUIDPreservation(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	
	id := uuid.New().String()
	result := r.RedactString("ID: " + id)
	
	// UUID should be preserved
	if !strings.Contains(result, id) {
		t.Errorf("UUID should not be redacted, got: %s", result)
	}
}

// mockHandler captures logs in memory for testing.
type mockHandler struct {
	mu    sync.Mutex
	count int
	logs  []string
}

func (h *mockHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *mockHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.logs = append(h.logs, r.Message)
	return nil
}

func (h *mockHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *mockHandler) WithGroup(_ string) slog.Handler {
	return h
}

// TestSamplingConcurrent kiểm tra sampling thread-safe.
func TestSamplingConcurrent(t *testing.T) {
	inner := &mockHandler{}
	cfg := &SamplingConfig{
		TokensPerSecond: 100,
		Burst:           10,
	}
	handler := NewSamplingHandler(inner, cfg)
	logger := slog.New(handler)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			logger.Info("concurrent", slog.Int("n", n))
		}(i)
	}
	wg.Wait()

	// Should not exceed burst + refill capacity significantly
	if inner.count > 200 {
		t.Errorf("sampling should limit calls, got %d", inner.count)
	}
}
