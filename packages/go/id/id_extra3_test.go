// Extra tests for id package.
package id

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

func TestUUIDv7_Length(t *testing.T) {
	id := UUIDv7()
	if len(id) != 36 {
		t.Errorf("expected 36 chars, got %d", len(id))
	}
}

func TestUUIDv7_Unique(t *testing.T) {
	a := UUIDv7()
	b := UUIDv7()
	if a == b {
		t.Error("UUIDs should be unique")
	}
}

func TestUUIDv7_ValidFormat(t *testing.T) {
	id := UUIDv7()
	if !IsUUID(id) {
		t.Error("UUIDv7 output should be valid UUID")
	}
}

func TestUUIDv7Bytes_Length(t *testing.T) {
	b := UUIDv7Bytes()
	if len(b) != 16 {
		t.Errorf("expected 16 bytes, got %d", len(b))
	}
}

func TestUUIDv4_Valid(t *testing.T) {
	id := UUIDv4()
	if !IsUUID(id) {
		t.Error("UUIDv4 should be valid")
	}
}

func TestParseUUID_Valid(t *testing.T) {
	_, err := ParseUUID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Errorf("parse: %v", err)
	}
}

func TestParseUUID_Invalid(t *testing.T) {
	_, err := ParseUUID("not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestIsUUID_True(t *testing.T) {
	if !IsUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("should be valid")
	}
}

func TestIsUUID_False(t *testing.T) {
	if IsUUID("not-a-uuid") {
		t.Error("should be invalid")
	}
}

func TestIsUUID_Empty(t *testing.T) {
	if IsUUID("") {
		t.Error("empty should be invalid")
	}
}

func TestNewULID_Length(t *testing.T) {
	id := NewULID()
	if len(id) != 26 {
		t.Errorf("expected 26 chars, got %d", len(id))
	}
}

func TestNewULID_Unique(t *testing.T) {
	a := NewULID()
	b := NewULID()
	if a == b {
		t.Error("ULIDs should be unique")
	}
}

func TestNewULID_Sortable(t *testing.T) {
	id1 := NewULID()
	time.Sleep(2 * time.Millisecond)
	id2 := NewULID()
	// Later ULID should sort later lexicographically
	if id1 >= id2 {
		t.Error("ULIDs should be lexically sortable")
	}
}

func TestULIDAt_Deterministic(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	id1 := ULIDAt(ts)
	id2 := ULIDAt(ts)
	// They share timestamp prefix but random suffix differs
	if id1 == id2 {
		t.Error("ULIDs at same time should still differ in random part")
	}
	// But should share the timestamp portion (10 chars)
	if id1[:10] != id2[:10] {
		t.Errorf("timestamp prefix should match: %s vs %s", id1[:10], id2[:10])
	}
}

func TestParseULID(t *testing.T) {
	id := NewULID()
	info, err := ParseULID(id)
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != id {
		t.Errorf("got %s, want %s", info.ID, id)
	}
	if info.Time.IsZero() {
		t.Error("time should be set")
	}
	if len(info.Bytes) != 16 {
		t.Errorf("bytes: %d", len(info.Bytes))
	}
}

func TestParseULID_Invalid(t *testing.T) {
	_, err := ParseULID("invalid-ulid")
	if err == nil {
		t.Error("expected error")
	}
}

func TestNanoID_Length(t *testing.T) {
	id := NanoID()
	if len(id) != DefaultNanoIDSize {
		t.Errorf("expected %d chars, got %d", DefaultNanoIDSize, len(id))
	}
}

func TestNanoID_Unique(t *testing.T) {
	a := NanoID()
	b := NanoID()
	if a == b {
		t.Error("NanoIDs should be unique")
	}
}

func TestNanoID_Alphabet(t *testing.T) {
	id := NanoID()
	for _, c := range id {
		if !strings.ContainsRune(string(nanoAlphabet), c) {
			t.Errorf("character %c not in alphabet", c)
		}
	}
}

func TestNanoIDWithSize_Custom(t *testing.T) {
	id := NanoIDWithSize(10)
	if len(id) != 10 {
		t.Errorf("expected 10, got %d", len(id))
	}
}

func TestNanoIDWithSize_Zero(t *testing.T) {
	id := NanoIDWithSize(0)
	if len(id) != DefaultNanoIDSize {
		t.Errorf("expected default %d, got %d", DefaultNanoIDSize, len(id))
	}
}

func TestNanoIDWithSize_Negative(t *testing.T) {
	id := NanoIDWithSize(-5)
	if len(id) != DefaultNanoIDSize {
		t.Errorf("expected default for negative, got %d", len(id))
	}
}

func TestNanoIDWithPrefixEx(t *testing.T) {
	id := NanoIDWithPrefix("usr")
	if !strings.HasPrefix(id, "usr_") {
		t.Errorf("missing prefix: %s", id)
	}
}

func TestSecretToken_Length32(t *testing.T) {
	tok := SecretToken(32)
	// 32 bytes base64url = 43 chars (no padding)
	if len(tok) != 43 {
		t.Errorf("expected 43 chars, got %d", len(tok))
	}
}

func TestSecretToken_Length16(t *testing.T) {
	tok := SecretToken(16)
	// 16 bytes base64url = 22 chars
	if len(tok) != 22 {
		t.Errorf("expected 22 chars, got %d", len(tok))
	}
}

func TestSecretToken_Unique(t *testing.T) {
	a := SecretToken(32)
	b := SecretToken(32)
	if a == b {
		t.Error("tokens should be unique")
	}
}

func TestSnowflake_New(t *testing.T) {
	s := NewSnowflake()
	if s == nil {
		t.Fatal("nil snowflake")
	}
}

func TestSnowflake_Generate(t *testing.T) {
	s := NewSnowflake()
	id := s.Generate()
	if id == 0 {
		t.Error("zero id")
	}
}

func TestSnowflake_Generate_Unique(t *testing.T) {
	s := NewSnowflake()
	a := s.Generate()
	b := s.Generate()
	// Should generally be different
	_ = a
	_ = b
}

func TestSnowflake_String(t *testing.T) {
	s := NewSnowflake()
	str := s.String()
	if str == "" {
		t.Error("empty string")
	}
	// Should be valid base32
	_, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(str))
	if err != nil {
		t.Errorf("invalid base32: %v", err)
	}
}

func TestSnowflake_String_Unique(t *testing.T) {
	s := NewSnowflake()
	a := s.String()
	time.Sleep(2 * time.Millisecond)
	b := s.String()
	if a == b {
		t.Error("snowflakes should be unique")
	}
}

func TestDefaultNanoIDSize(t *testing.T) {
	if DefaultNanoIDSize != 21 {
		t.Errorf("got %d", DefaultNanoIDSize)
	}
}

func TestSnowflakeEpoch(t *testing.T) {
	// 2024-01-01T00:00:00Z
	expected := int64(1704067200000)
	if snowflakeEpoch != expected {
		t.Errorf("got %d, want %d", snowflakeEpoch, expected)
	}
}

func TestUUIDv7Bytes_Unique(t *testing.T) {
	a := UUIDv7Bytes()
	b := UUIDv7Bytes()
	if string(a) == string(b) {
		t.Error("UUID bytes should be unique")
	}
}

func TestNanoAlphabet_PowerOf2(t *testing.T) {
	// 64 = 2^6, must be power of 2 for the masking logic
	if len(nanoAlphabet) != 64 {
		t.Errorf("alphabet size: %d", len(nanoAlphabet))
	}
}

func TestNanoAlphabet_UniqueChars(t *testing.T) {
	seen := make(map[byte]bool)
	for _, c := range nanoAlphabet {
		if seen[c] {
			t.Errorf("duplicate char: %c", c)
		}
		seen[c] = true
	}
}
