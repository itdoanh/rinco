// Additional tests for id package.
package id

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestExtra_UUIDv7_Length(t *testing.T) {
	id := UUIDv7()
	if len(id) != 36 {
		t.Errorf("expected 36 chars, got %d", len(id))
	}
}

func TestExtra_UUIDv7_Uniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := UUIDv7()
		if seen[id] {
			t.Error("duplicate UUID")
		}
		seen[id] = true
	}
}

func TestExtra_UUIDv7Bytes_Length(t *testing.T) {
	b := UUIDv7Bytes()
	if len(b) != 16 {
		t.Errorf("expected 16 bytes, got %d", len(b))
	}
}

func TestExtra_UUIDv4_Length(t *testing.T) {
	id := UUIDv4()
	if len(id) != 36 {
		t.Errorf("expected 36 chars, got %d", len(id))
	}
}

func TestExtra_ParseUUID_Valid(t *testing.T) {
	id := UUIDv7()
	u, err := ParseUUID(id)
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if u.String() != id {
		t.Errorf("roundtrip mismatch")
	}
}

func TestExtra_ParseUUID_Invalid(t *testing.T) {
	_, err := ParseUUID("not-a-uuid")
	if err == nil {
		t.Error("invalid should fail")
	}
}

func TestExtra_IsUUID_Valid(t *testing.T) {
	if !IsUUID(UUIDv7()) {
		t.Error("valid UUID should return true")
	}
}

func TestExtra_IsUUID_Invalid(t *testing.T) {
	if IsUUID("not-a-uuid") {
		t.Error("invalid should return false")
	}
	if IsUUID("") {
		t.Error("empty should return false")
	}
}

func TestExtra_ULID_Length(t *testing.T) {
	u := NewULID()
	if len(u) != 26 {
		t.Errorf("expected 26 chars, got %d", len(u))
	}
}

func TestExtra_ULID_Uniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		u := NewULID()
		if seen[u] {
			t.Error("duplicate ULID")
		}
		seen[u] = true
	}
}

func TestExtra_ULIDAt_SpecificTime(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	u := ULIDAt(t1)
	info, err := ParseULID(u)
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if !info.Time.Equal(t1) {
		t.Errorf("time: got %v, want %v", info.Time, t1)
	}
}

func TestExtra_ParseULID_Invalid(t *testing.T) {
	_, err := ParseULID("not-a-ulid")
	if err == nil {
		t.Error("invalid should fail")
	}
}

func TestExtra_ParseULID_TooShort(t *testing.T) {
	_, err := ParseULID("ABC")
	if err == nil {
		t.Error("too short should fail")
	}
}

func TestExtra_NanoID_Length(t *testing.T) {
	id := NanoID()
	if len(id) != DefaultNanoIDSize {
		t.Errorf("expected %d chars, got %d", DefaultNanoIDSize, len(id))
	}
}

func TestExtra_NanoID_Uniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NanoID()
		if seen[id] {
			t.Error("duplicate NanoID")
		}
		seen[id] = true
	}
}

func TestExtra_NanoIDWithSize_Zero(t *testing.T) {
	// Should default to DefaultNanoIDSize
	id := NanoIDWithSize(0)
	if len(id) != DefaultNanoIDSize {
		t.Errorf("expected default size, got %d", len(id))
	}
}

func TestExtra_NanoIDWithSize_Negative(t *testing.T) {
	id := NanoIDWithSize(-5)
	if len(id) != DefaultNanoIDSize {
		t.Errorf("expected default size for negative, got %d", len(id))
	}
}

func TestExtra_NanoIDWithSize_Custom(t *testing.T) {
	id := NanoIDWithSize(10)
	if len(id) != 10 {
		t.Errorf("expected 10, got %d", len(id))
	}
}

func TestExtra_NanoIDWithPrefix(t *testing.T) {
	id := NanoIDWithPrefix("usr")
	if !strings.HasPrefix(id, "usr_") {
		t.Errorf("expected usr_ prefix: got %s", id)
	}
}

func TestExtra_SecretToken_Length(t *testing.T) {
	token := SecretToken(32)
	if len(token) == 0 {
		t.Error("empty token")
	}
}

func TestExtra_SecretToken_Uniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		t1 := SecretToken(32)
		if seen[t1] {
			t.Error("duplicate token")
		}
		seen[t1] = true
	}
}

func TestExtra_SecretToken_Sizes(t *testing.T) {
	for _, n := range []int{8, 16, 32, 64} {
		token := SecretToken(n)
		if len(token) == 0 {
			t.Errorf("empty token for n=%d", n)
		}
	}
}

func TestExtra_NewSnowflake(t *testing.T) {
	s := NewSnowflake()
	if s == nil {
		t.Fatal("nil snowflake")
	}
}

func TestExtra_Snowflake_Generate(t *testing.T) {
	s := NewSnowflake()
	id := s.Generate()
	if id == 0 {
		t.Error("snowflake should not be 0")
	}
}

func TestExtra_Snowflake_Uniqueness(t *testing.T) {
	s := NewSnowflake()
	seen := map[int64]bool{}
	for i := 0; i < 1000; i++ {
		id := s.Generate()
		if seen[id] {
			t.Error("duplicate snowflake")
		}
		seen[id] = true
	}
}

func TestExtra_Snowflake_String(t *testing.T) {
	s := NewSnowflake()
	str := s.String()
	if str == "" {
		t.Error("empty snowflake string")
	}
}

func TestExtra_Snowflake_Concurrent(t *testing.T) {
	s := NewSnowflake()
	var wg sync.WaitGroup
	ids := make(chan int64, 1000)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- s.Generate()
		}()
	}
	wg.Wait()
	close(ids)
}

func TestExtra_NanoID_Alphabet(t *testing.T) {
	// Verify all chars are from alphabet
	id := NanoID()
	for _, c := range id {
		found := false
		for _, a := range nanoAlphabet {
			if byte(c) == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("char %c not in alphabet", c)
			break
		}
	}
}
