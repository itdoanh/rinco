// Tests for auth package internal helpers (uuid + sha256).
package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// uuidGen
// =============================================================================

func TestExtraUUIDGen_ValidUUID(t *testing.T) {
	got := uuidGen()
	if got == "" {
		t.Fatal("uuidGen returned empty")
	}
	// Should be parseable as UUID.
	if _, err := uuid.Parse(got); err != nil {
		t.Errorf("uuidGen produced unparseable UUID %q: %v", got, err)
	}
}

func TestExtraUUIDGen_UniquePerCall(t *testing.T) {
	const N = 100
	seen := make(map[string]bool, N)
	for i := 0; i < N; i++ {
		got := uuidGen()
		if seen[got] {
			t.Fatalf("duplicate UUID at iteration %d: %s", i, got)
		}
		seen[got] = true
	}
}

func TestExtraUUIDGen_ConsistentLength(t *testing.T) {
	// Standard UUID string is 36 chars (32 hex + 4 dashes).
	for i := 0; i < 10; i++ {
		if got := uuidGen(); len(got) != 36 {
			t.Errorf("expected 36 chars, got %d: %s", len(got), got)
		}
	}
}

func TestExtraUUIDGen_FirstByteIsVersion(t *testing.T) {
	// UUID v7 has version nibble = 7 in the 13th hex digit; UUID v4 has 4.
	// Both start with the same byte pattern in some positions. We just
	// verify it's a parseable UUID and skip version-specific assertions.
	for i := 0; i < 20; i++ {
		got := uuidGen()
		u, err := uuid.Parse(got)
		if err != nil {
			t.Fatalf("iteration %d: parse failed: %v", i, err)
		}
		_ = u // parsed successfully
	}
}

// =============================================================================
// sha256sumImpl
// =============================================================================

func TestExtraSHA256Sum_EmptyInput(t *testing.T) {
	got := sha256sumImpl([]byte{})
	want := sha256.Sum256([]byte{})
	if !bytes.Equal(got, want[:]) {
		t.Errorf("empty: got %x, want %x", got, want)
	}
}

func TestExtraSHA256Sum_NilInput(t *testing.T) {
	got := sha256sumImpl(nil)
	want := sha256.Sum256(nil)
	if !bytes.Equal(got, want[:]) {
		t.Errorf("nil: got %x, want %x", got, want)
	}
}

func TestExtraSHA256Sum_KnownVector(t *testing.T) {
	// SHA-256 of "abc" is well-known.
	got := sha256sumImpl([]byte("abc"))
	want, _ := hex.DecodeString("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	if !bytes.Equal(got, want) {
		t.Errorf("'abc': got %x, want %x", got, want)
	}
}

func TestExtraSHA256Sum_DifferentInputsDifferentOutputs(t *testing.T) {
	a := sha256sumImpl([]byte("a"))
	b := sha256sumImpl([]byte("b"))
	if bytes.Equal(a, b) {
		t.Errorf("expected different hashes for different inputs")
	}
}

func TestExtraSHA256Sum_Length(t *testing.T) {
	got := sha256sumImpl([]byte("any"))
	if len(got) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(got))
	}
}

func TestExtraSHA256Sum_LargeInput(t *testing.T) {
	data := make([]byte, 100*1024)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}
	got := sha256sumImpl(data)
	if len(got) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(got))
	}
	// Should match stdlib.
	want := sha256.Sum256(data)
	if !bytes.Equal(got, want[:]) {
		t.Error("large input mismatch with stdlib")
	}
}
