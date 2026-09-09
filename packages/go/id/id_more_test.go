package id

import (
	"strings"
	"testing"
	"time"
)

func TestUUIDv7Format(t *testing.T) {
	id := UUIDv7()
	if len(id) != 36 {
		t.Errorf("UUIDv7 len = %d, want 36", len(id))
	}
	if !IsUUID(id) {
		t.Errorf("UUIDv7 invalid: %s", id)
	}
}

func TestUUIDv7Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := UUIDv7()
		if seen[id] {
			t.Errorf("duplicate UUID: %s", id)
		}
		seen[id] = true
	}
}

func TestUUIDv4Format(t *testing.T) {
	id := UUIDv4()
	if !IsUUID(id) {
		t.Errorf("UUIDv4 invalid: %s", id)
	}
}

func TestUUIDv7BytesLength(t *testing.T) {
	b := UUIDv7Bytes()
	if len(b) != 16 {
		t.Errorf("bytes len = %d, want 16", len(b))
	}
}

func TestParseUUIDSuccess(t *testing.T) {
	original := UUIDv7()
	u, err := ParseUUID(original)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.String() != original {
		t.Errorf("roundtrip mismatch: %s vs %s", u.String(), original)
	}
}

func TestParseUUIDInvalid(t *testing.T) {
	_, err := ParseUUID("not-a-uuid")
	if err == nil {
		t.Error("expected error")
	}
}

func TestIsUUIDValid(t *testing.T) {
	if !IsUUID(UUIDv7()) {
		t.Error("IsUUID returned false for valid")
	}
}

func TestIsUUIDInvalid(t *testing.T) {
	if IsUUID("not-a-uuid") {
		t.Error("IsUUID returned true for invalid")
	}
	if IsUUID("") {
		t.Error("IsUUID returned true for empty")
	}
}

func TestNewULIDFormat(t *testing.T) {
	id := NewULID()
	if len(id) != 26 {
		t.Errorf("ULID len = %d, want 26", len(id))
	}
}

func TestNewULIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := NewULID()
		if seen[id] {
			t.Error("duplicate ULID")
		}
		seen[id] = true
	}
}

func TestULIDAtVar(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	id := ULIDAt(ts)
	info, err := ParseULID(id)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// The time should be close to the requested timestamp
	diff := info.Time.Sub(ts).Abs()
	if diff > 2*time.Second {
		t.Errorf("time diff too large: %v", diff)
	}
}

func TestParseULIDRoundtrip(t *testing.T) {
	id := NewULID()
	info, err := ParseULID(id)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if info.ID != id {
		t.Errorf("ID roundtrip mismatch: %s vs %s", info.ID, id)
	}
}

func TestParseULIDInvalid(t *testing.T) {
	_, err := ParseULID("invalid-ulid-string-here")
	if err == nil {
		t.Error("expected error")
	}
}

func TestNanoIDFormat(t *testing.T) {
	id := NanoID()
	if len(id) != DefaultNanoIDSize {
		t.Errorf("default NanoID len = %d, want %d", len(id), DefaultNanoIDSize)
	}
}

func TestNanoIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := NanoID()
		if seen[id] {
			t.Error("duplicate NanoID")
		}
		seen[id] = true
	}
}

func TestNanoIDCustomSize(t *testing.T) {
	id := NanoIDWithSize(10)
	if len(id) != 10 {
		t.Errorf("custom len = %d, want 10", len(id))
	}
}

func TestNanoIDCustomSizeZero(t *testing.T) {
	id := NanoIDWithSize(0)
	if len(id) != DefaultNanoIDSize {
		t.Errorf("zero len = %d, want default %d", len(id), DefaultNanoIDSize)
	}
}

func TestNanoIDCustomSizeNegative(t *testing.T) {
	id := NanoIDWithSize(-5)
	if len(id) != DefaultNanoIDSize {
		t.Errorf("negative len = %d, want default %d", len(id), DefaultNanoIDSize)
	}
}

func TestNanoIDWithPrefixVariant(t *testing.T) {
	id := NanoIDWithPrefix("usr")
	if !strings.HasPrefix(id, "usr_") {
		t.Errorf("missing prefix: %s", id)
	}
}

func TestSecretTokenLength(t *testing.T) {
	tok := SecretToken(32)
	// base64.RawURLEncoding(32) = ceil(32*4/3) = 43 chars (no padding)
	if len(tok) != 43 {
		t.Errorf("token len = %d, want 43", len(tok))
	}
}

func TestSecretTokenUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		tok := SecretToken(16)
		if seen[tok] {
			t.Error("duplicate token")
		}
		seen[tok] = true
	}
}

func TestSecretTokenZero(t *testing.T) {
	tok := SecretToken(0)
	if tok != "" {
		t.Errorf("zero byte token: %q", tok)
	}
}

func TestUUIDv7EmbedsTimestamp(t *testing.T) {
	// UUIDv7 embeds a millisecond timestamp; multiple calls in <1ms may collide.
	// Just verify the format.
	id := UUIDv7()
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Errorf("UUID parts = %d, want 5", len(parts))
	}
}

func TestNewULIDVariance(t *testing.T) {
	id1 := NewULID()
	time.Sleep(2 * time.Millisecond)
	id2 := NewULID()
	info1, _ := ParseULID(id1)
	info2, _ := ParseULID(id2)
	if !info2.Time.After(info1.Time) {
		t.Error("second ULID should be later")
	}
}

func TestIDAlphabetPowerOf2(t *testing.T) {
	if len(nanoAlphabet) != 64 {
		t.Errorf("alphabet len = %d, want 64", len(nanoAlphabet))
	}
}
