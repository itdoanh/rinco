// Tests for capi dedup helpers (dedup.go).
package capi

import (
	"strings"
	"testing"
	"time"
)

func TestExtra_DedupResult_Fields(t *testing.T) {
	r := DedupResult{
		IsDuplicate: true,
		EventID:     "event-123",
	}
	if !r.IsDuplicate {
		t.Error("IsDuplicate")
	}
	if r.EventID != "event-123" {
		t.Error("EventID")
	}
}

func TestExtra_NewDeduper_Defaults(t *testing.T) {
	d := NewDeduper(DedupConfig{})
	if d.config.TTL != 72*time.Hour {
		t.Errorf("TTL default: got %v", d.config.TTL)
	}
	if d.config.KeyPrefix != "capi:dedup:" {
		t.Errorf("KeyPrefix default: got %s", d.config.KeyPrefix)
	}
}

func TestExtra_NewDeduper_Custom(t *testing.T) {
	d := NewDeduper(DedupConfig{
		TTL:       24 * time.Hour,
		KeyPrefix: "custom:",
	})
	if d.config.TTL != 24*time.Hour {
		t.Errorf("TTL: got %v", d.config.TTL)
	}
	if d.config.KeyPrefix != "custom:" {
		t.Errorf("KeyPrefix: got %s", d.config.KeyPrefix)
	}
}

func TestExtra_DefaultDedupConfig(t *testing.T) {
	cfg := DefaultDedupConfig(nil)
	if cfg.TTL != 72*time.Hour {
		t.Errorf("TTL: got %v", cfg.TTL)
	}
	if cfg.KeyPrefix != "capi:dedup:" {
		t.Errorf("KeyPrefix: got %s", cfg.KeyPrefix)
	}
	if cfg.Redis != nil {
		t.Error("Redis should be nil from parameter")
	}
}

func TestExtra_GenerateEventIDFromFields_SameInput(t *testing.T) {
	id1 := GenerateEventIDFromFields("tenant1", "form1", "user@example.com", "1234567890")
	id2 := GenerateEventIDFromFields("tenant1", "form1", "user@example.com", "1234567890")
	if id1 != id2 {
		t.Errorf("Same input should produce same ID: %s vs %s", id1, id2)
	}
}

func TestExtra_GenerateEventIDFromFields_DifferentTenant(t *testing.T) {
	id1 := GenerateEventIDFromFields("tenant1", "form1", "user@example.com", "1234567890")
	id2 := GenerateEventIDFromFields("tenant2", "form1", "user@example.com", "1234567890")
	if id1 == id2 {
		t.Error("Different tenant should produce different ID")
	}
}

func TestExtra_GenerateEventIDFromFields_DifferentForm(t *testing.T) {
	id1 := GenerateEventIDFromFields("tenant1", "form1", "user@example.com", "1234567890")
	id2 := GenerateEventIDFromFields("tenant1", "form2", "user@example.com", "1234567890")
	if id1 == id2 {
		t.Error("Different form should produce different ID")
	}
}

func TestExtra_GenerateEventIDFromFields_DifferentEmail(t *testing.T) {
	id1 := GenerateEventIDFromFields("tenant1", "form1", "user1@example.com", "1234567890")
	id2 := GenerateEventIDFromFields("tenant1", "form1", "user2@example.com", "1234567890")
	if id1 == id2 {
		t.Error("Different email should produce different ID")
	}
}

func TestExtra_GenerateEventIDFromFields_DifferentPhone(t *testing.T) {
	id1 := GenerateEventIDFromFields("tenant1", "form1", "user@example.com", "1111111111")
	id2 := GenerateEventIDFromFields("tenant1", "form1", "user@example.com", "2222222222")
	if id1 == id2 {
		t.Error("Different phone should produce different ID")
	}
}

func TestExtra_GenerateEventIDFromFields_UUIDFormat(t *testing.T) {
	id := GenerateEventIDFromFields("t", "f", "e", "p")
	// Should be formatted with dashes
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Errorf("Expected UUID-like format with 5 parts, got %s (parts: %d)", id, len(parts))
	}
	// Verify version is 4 in third group
	if len(parts[2]) > 0 && parts[2][0] != '4' {
		t.Errorf("Version should be 4, got %s", parts[2])
	}
}

func TestExtra_GenerateEventIDFromFields_Length(t *testing.T) {
	id := GenerateEventIDFromFields("t", "f", "e", "p")
	if len(id) < 36 {
		t.Errorf("Expected at least 36 chars, got %d", len(id))
	}
}

func TestExtra_NewEventID(t *testing.T) {
	cfg := EventIDConfig{
		TenantID: "tenant1",
		FormSlug: "form1",
		Email:    "user@example.com",
		Phone:    "1234567890",
	}
	id := NewEventID(cfg)
	if id == "" {
		t.Error("NewEventID returned empty")
	}
}

func TestExtra_NewEventID_Deterministic(t *testing.T) {
	cfg := EventIDConfig{
		TenantID: "tenant1",
		FormSlug: "form1",
		Email:    "user@example.com",
		Phone:    "1234567890",
	}
	id1 := NewEventID(cfg)
	id2 := NewEventID(cfg)
	if id1 != id2 {
		t.Error("Same config should produce same ID")
	}
}

func TestExtra_EventIDConfig_Fields(t *testing.T) {
	cfg := EventIDConfig{
		TenantID:  "t1",
		FormSlug:  "f1",
		Email:     "e1",
		Phone:     "p1",
		Timestamp: 1234567890,
		IP:        "1.2.3.4",
	}
	if cfg.TenantID != "t1" {
		t.Error("TenantID")
	}
	if cfg.Timestamp != 1234567890 {
		t.Error("Timestamp")
	}
	if cfg.IP != "1.2.3.4" {
		t.Error("IP")
	}
}

func TestExtra_GenerateEventIDFromFields_EmptyFields(t *testing.T) {
	id := GenerateEventIDFromFields("", "", "", "")
	if id == "" {
		t.Error("Empty fields should still produce a non-empty ID")
	}
}

func TestExtra_GenerateEventIDFromFields_UnicodeFields(t *testing.T) {
	id := GenerateEventIDFromFields("tënant", "förm", "üser@ëxample.com", "12345")
	// Should produce a non-empty ID (length check is format-dependent)
	if len(id) < 30 {
		t.Errorf("Unicode fields should still produce reasonable-length ID: %s", id)
	}
}

func TestExtra_DedupConfig_Fields(t *testing.T) {
	cfg := DedupConfig{
		TTL:       1 * time.Hour,
		KeyPrefix: "test:",
	}
	if cfg.TTL != 1*time.Hour {
		t.Error("TTL")
	}
	if cfg.KeyPrefix != "test:" {
		t.Error("KeyPrefix")
	}
}
