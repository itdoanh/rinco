// Additional tests for packages/go/capi dedup.go (GenerateEventIDFromFields).
package capi

import (
	"strings"
	"testing"
	"time"
)

// =============================================================================
// GenerateEventIDFromFields — UUID format validation
// =============================================================================

func TestExtraDedup_GenerateEventIDFromFields_UUIDFormat(t *testing.T) {
	id := GenerateEventIDFromFields("tenant-1", "contact", "a@b.com", "+1234")
	// The format produces: 8-4-5-4-12 = 33 hex chars + 4 dashes = 37 total.
	// (The 3rd group has the "4" v4-marker prefix prepended.)
	if len(id) != 37 {
		t.Errorf("expected 37 chars, got %d (%s)", len(id), id)
	}
	// Dashes at expected positions (4 of them).
	dashCount := strings.Count(id, "-")
	if dashCount != 4 {
		t.Errorf("expected 4 dashes, got %d in %s", dashCount, id)
	}
}

func TestExtraDedup_GenerateEventIDFromFields_Version4(t *testing.T) {
	// First nibble of 3rd group should be '4' (UUID v4).
	id := GenerateEventIDFromFields("t", "f", "a@b.com", "")
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d in %s", len(parts), id)
	}
	// 3rd group: "4" + hash[12:16] = "4xxxx"
	if !strings.HasPrefix(parts[2], "4") {
		t.Errorf("expected v4 marker, got %s", parts[2])
	}
}

func TestExtraDedup_GenerateEventIDFromFields_HexChars(t *testing.T) {
	id := GenerateEventIDFromFields("t", "f", "e", "p")
	for _, c := range id {
		if c == '-' {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex char %q in %s", c, id)
		}
	}
}

func TestExtraDedup_GenerateEventIDFromFields_SameInputSameOutput(t *testing.T) {
	a := GenerateEventIDFromFields("t1", "f1", "e1", "p1")
	b := GenerateEventIDFromFields("t1", "f1", "e1", "p1")
	if a != b {
		t.Errorf("deterministic: %s vs %s", a, b)
	}
}

func TestExtraDedup_GenerateEventIDFromFields_DifferentTenant(t *testing.T) {
	a := GenerateEventIDFromFields("t1", "f", "e", "p")
	b := GenerateEventIDFromFields("t2", "f", "e", "p")
	if a == b {
		t.Error("different tenant should produce different ID")
	}
}

func TestExtraDedup_GenerateEventIDFromFields_DifferentFormSlug(t *testing.T) {
	a := GenerateEventIDFromFields("t", "f1", "e", "p")
	b := GenerateEventIDFromFields("t", "f2", "e", "p")
	if a == b {
		t.Error("different form slug should produce different ID")
	}
}

func TestExtraDedup_GenerateEventIDFromFields_DifferentEmail(t *testing.T) {
	a := GenerateEventIDFromFields("t", "f", "e1", "p")
	b := GenerateEventIDFromFields("t", "f", "e2", "p")
	if a == b {
		t.Error("different email should produce different ID")
	}
}

func TestExtraDedup_GenerateEventIDFromFields_DifferentPhone(t *testing.T) {
	a := GenerateEventIDFromFields("t", "f", "e", "p1")
	b := GenerateEventIDFromFields("t", "f", "e", "p2")
	if a == b {
		t.Error("different phone should produce different ID")
	}
}

func TestExtraDedup_GenerateEventIDFromFields_EmptyFields(t *testing.T) {
	id := GenerateEventIDFromFields("", "", "", "")
	if len(id) != 37 {
		t.Errorf("expected 37 chars, got %d", len(id))
	}
}

func TestExtraDedup_GenerateEventIDFromFields_UnicodeFields(t *testing.T) {
	id := GenerateEventIDFromFields("vn-tenant", "đăng-ký", "test@example.com", "✉️")
	if len(id) != 37 {
		t.Errorf("expected 37 chars, got %d", len(id))
	}
}

func TestExtraDedup_GenerateEventIDFromFields_LongInputs(t *testing.T) {
	long := strings.Repeat("x", 1000)
	id := GenerateEventIDFromFields(long, long, long, long)
	if len(id) != 37 {
		t.Errorf("expected 37 chars, got %d", len(id))
	}
}

func TestExtraDedup_GenerateEventIDFromFields_PipeInField_StableFormat(t *testing.T) {
	// Field containing "|" should not crash and should still produce valid IDs.
	a := GenerateEventIDFromFields("a|b", "c", "d", "e")
	b := GenerateEventIDFromFields("a", "b|c", "d", "e")
	if len(a) != 37 || len(b) != 37 {
		t.Errorf("expected 37 chars: %s, %s", a, b)
	}
}

// =============================================================================
// NewEventID
// =============================================================================

func TestExtraDedup_NewEventID_AutoTimestamp(t *testing.T) {
	cfg := EventIDConfig{TenantID: "t", FormSlug: "f", Email: "e", Phone: "p"}
	id := NewEventID(cfg)
	if len(id) != 37 {
		t.Errorf("expected 37 chars, got %d", len(id))
	}
}

func TestExtraDedup_NewEventID_DeterministicWithTimestamp(t *testing.T) {
	cfg := EventIDConfig{TenantID: "t", FormSlug: "f", Email: "e", Phone: "p", Timestamp: 1700000000}
	a := NewEventID(cfg)
	b := NewEventID(cfg)
	// Note: NewEventID currently doesn't use Timestamp in the hash, but the
	// behavior should still be deterministic for the same input.
	if a != b {
		t.Errorf("deterministic: %s vs %s", a, b)
	}
}

func TestExtraDedup_NewEventID_ZeroTimestamp(t *testing.T) {
	cfg := EventIDConfig{TenantID: "t", FormSlug: "f", Email: "e", Phone: "p", Timestamp: 0}
	id := NewEventID(cfg)
	if len(id) != 37 {
		t.Errorf("expected 37 chars, got %d", len(id))
	}
}

// =============================================================================
// DedupResult
// =============================================================================

func TestExtraDedup_DedupResult_Fields(t *testing.T) {
	r := DedupResult{IsDuplicate: true, EventID: "e1"}
	if !r.IsDuplicate {
		t.Error("IsDuplicate should be true")
	}
	if r.EventID != "e1" {
		t.Errorf("EventID: %s", r.EventID)
	}
}

// =============================================================================
// DedupConfig
// =============================================================================

func TestExtraDedup_DefaultDedupConfig_Fields(t *testing.T) {
	cfg := DefaultDedupConfig(nil)
	if cfg.TTL != 72*time.Hour {
		t.Errorf("TTL: got %v", cfg.TTL)
	}
	if cfg.KeyPrefix != "capi:dedup:" {
		t.Errorf("KeyPrefix: got %s", cfg.KeyPrefix)
	}
	if cfg.Redis != nil {
		t.Error("Redis should be nil (passed in)")
	}
}

// =============================================================================
// NewDeduper — defaults
// =============================================================================

func TestExtraDedup_NewDeduper_DefaultsTTL(t *testing.T) {
	d := NewDeduper(DedupConfig{}) // TTL=0 → default
	if d.config.TTL != 72*time.Hour {
		t.Errorf("expected 72h, got %v", d.config.TTL)
	}
}

func TestExtraDedup_NewDeduper_DefaultsKeyPrefix(t *testing.T) {
	d := NewDeduper(DedupConfig{}) // KeyPrefix="" → default
	if d.config.KeyPrefix != "capi:dedup:" {
		t.Errorf("expected capi:dedup:, got %s", d.config.KeyPrefix)
	}
}

func TestExtraDedup_NewDeduper_PreservesCustom(t *testing.T) {
	d := NewDeduper(DedupConfig{
		TTL:       1 * time.Hour,
		KeyPrefix: "custom:",
	})
	if d.config.TTL != 1*time.Hour {
		t.Errorf("TTL: got %v", d.config.TTL)
	}
	if d.config.KeyPrefix != "custom:" {
		t.Errorf("KeyPrefix: got %s", d.config.KeyPrefix)
	}
}

// =============================================================================
// FilterDuplicates — nil/empty input
// =============================================================================

// =============================================================================
// Key building smoke test (no Redis required)
// =============================================================================

func TestExtraDedup_KeyPrefix_Custom(t *testing.T) {
	d := NewDeduper(DedupConfig{KeyPrefix: "myapp:", TTL: time.Minute})
	if !strings.HasSuffix(d.config.KeyPrefix, ":") {
		t.Errorf("expected colon-suffix prefix, got %s", d.config.KeyPrefix)
	}
}
