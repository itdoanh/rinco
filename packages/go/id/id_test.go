package id

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUIDv7Shape(t *testing.T) {
	for i := 0; i < 5; i++ {
		got := UUIDv7()
		assert.Regexp(t, regexp.MustCompile(`^[0-9a-f-]{36}$`), got)
		// v7 should be lexically sortable in time → check ordering.
	}
	a, b := UUIDv7(), UUIDv7()
	assert.True(t, a <= b, "UUIDv7 should be monotonic: %s ≤ %s", a, b)
}

func TestUUIDv7Bytes(t *testing.T) {
	b := UUIDv7Bytes()
	assert.Len(t, b, 16)
}

func TestUUIDv4Shape(t *testing.T) {
	u := UUIDv4()
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f-]{36}$`), u)
}

func TestParseUUIDAndIsUUID(t *testing.T) {
	u := UUIDv7()
	parsed, err := ParseUUID(u)
	require.NoError(t, err)
	assert.Equal(t, u, parsed.String())

	assert.True(t, IsUUID(u))
	assert.False(t, IsUUID("not-a-uuid"))
	assert.False(t, IsUUID(""))
}

func TestNewULIDShape(t *testing.T) {
	u := NewULID()
	assert.Len(t, u, 26)
}

func TestULIDAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	u := ULIDAt(now)
	info, err := ParseULID(u)
	require.NoError(t, err)
	assert.Equal(t, now.UnixMilli(), info.Time.UnixMilli())
}

func TestNanoID(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := NanoID()
		assert.Len(t, id, DefaultNanoIDSize)
		assert.False(t, seen[id], "duplicate nanoID: %s", id)
		seen[id] = true
	}
}

func TestNanoIDWithSize(t *testing.T) {
	sizes := []int{10, 21, 32, 64}
	for _, n := range sizes {
		assert.Len(t, NanoIDWithSize(n), n)
	}
}

func TestNanoIDWithPrefix(t *testing.T) {
	s := NanoIDWithPrefix("usr")
	assert.True(t, strings.HasPrefix(s, "usr_"))
	assert.Len(t, s, 4+DefaultNanoIDSize)
}

func TestSecretToken(t *testing.T) {
	tok := SecretToken(32)
	assert.GreaterOrEqual(t, len(tok), 42)
	// uniqueness
	t2 := SecretToken(32)
	assert.NotEqual(t, tok, t2)
}

func TestSnowflakeMonotonic(t *testing.T) {
	s := NewSnowflake()
	var prev int64 = -1
	for i := 0; i < 100; i++ {
		cur := s.Generate()
		if prev >= 0 {
			assert.GreaterOrEqual(t, cur, prev)
		}
		prev = cur
	}
}

func TestSnowflakeString(t *testing.T) {
	s := NewSnowflake()
	str := s.String()
	assert.NotEmpty(t, str)
}