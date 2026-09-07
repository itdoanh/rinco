// Package id provides UUID v7, ULID, and NanoID generators for sortable,
// URL-safe identifiers used across RINCO services.
package id

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// UUIDv7 returns a UUID v7 string (time-ordered, lexically sortable).
// Falls back to UUID v4 if v7 is unsupported by the underlying library.
func UUIDv7() string {
	if u, err := uuid.NewV7(); err == nil {
		return u.String()
	}
	return uuid.New().String()
}

// UUIDv7Bytes returns the 16 raw bytes of a UUID v7.
func UUIDv7Bytes() []byte {
	if u, err := uuid.NewV7(); err == nil {
		return u[:]
	}
	u := uuid.New()
	return u[:]
}

// UUIDv4 returns a UUID v4 string.
func UUIDv4() string {
	return uuid.New().String()
}

// ParseUUID parses a UUID string. Returns an error on invalid input.
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// IsUUID returns true if the string is a valid UUID.
func IsUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// ULID generates a 26-char Crockford base32 ULID (time-ordered, sortable).
func NewULID() string {
	return ulid.Make().String()
}

// ULIDAt returns the ULID string for a specific time (useful for tests).
func ULIDAt(t time.Time) string {
	return ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
}

// ParseULID parses a ULID string into its components.
func ParseULID(s string) (ULIDInfo, error) {
	id, err := ulid.Parse(s)
	if err != nil {
		return ULIDInfo{}, err
	}
	return ULIDInfo{
		ID:    id.String(),
		Time:  ulid.Time(id.Time()),
		Bytes: id[:],
	}, nil
}

// ULIDInfo is the result of parsing a ULID.
type ULIDInfo struct {
	ID    string
	Time  time.Time
	Bytes []byte
}

// NanoID generates a 21-character URL-safe unique identifier.
// Uses crypto/rand under the hood; thread-safe.
func NanoID() string {
	return nanoIDWithSize(DefaultNanoIDSize)
}

// DefaultNanoIDSize is the default length for NanoID.
const DefaultNanoIDSize = 21

var nanoAlphabet = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-")

var nanoMu sync.Mutex
var nanoBuf [16]byte

// nanoIDWithSize returns a NanoID of the requested length. Length must be > 0.
func nanoIDWithSize(size int) string {
	if size <= 0 {
		size = DefaultNanoIDSize
	}
	out := make([]byte, size)
	nanoMu.Lock()
	defer nanoMu.Unlock()
	// Each iteration we mask off bytes we don't need and fill one output char.
	for i := 0; i < size; i++ {
		// Read 8 bytes at a time until we find a byte less than the alphabet boundary.
		for {
			if _, err := rand.Read(nanoBuf[:]); err != nil {
				panic(fmt.Errorf("nanoID: %w", err))
			}
			// Bound must be a multiple of the alphabet length for unbiased selection.
			mask := byte(len(nanoAlphabet) - 1)
			if n := len(nanoAlphabet); n&int(mask) != 0 {
				panic("alphabet length must be a power of 2")
			}
			out[i] = nanoAlphabet[nanoBuf[0]&mask]
			if int(out[i]) < len(nanoAlphabet) {
				break
			}
		}
	}
	return string(out)
}

// NanoIDWithSize generates a NanoID of the requested length.
func NanoIDWithSize(size int) string { return nanoIDWithSize(size) }

// NanoIDWithPrefix returns a NanoID prefixed by a short string for visual grouping
// (e.g. "usr_a8s9d7f6...").
func NanoIDWithPrefix(prefix string) string {
	return prefix + "_" + NanoID()
}

// SecretToken returns a base64url-encoded random token of N bytes.
// N=32 → 43-char token (256-bit entropy).
func SecretToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("SecretToken: %w", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Snowflake-like 64-bit ID combining timestamp + random bits.
// Format: 41-bit ms timestamp | 10-bit random | 12-bit sequence-like random.
// Not actually distributed-safe (no per-instance worker ID), but useful within
// a single process where you need compact, roughly-ordered IDs.
type Snowflake struct {
	mu   sync.Mutex
	last int64
	seq  uint16
}

// NewSnowflake creates a generator with epoch = Jan 1 2024 UTC.
func NewSnowflake() *Snowflake {
	return &Snowflake{}
}

const (
	snowflakeEpoch int64 = 1704067200000 // 2024-01-01T00:00:00Z in ms
)

// Generate returns a 64-bit snowflake ID.
func (s *Snowflake) Generate() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli() - snowflakeEpoch
	if now == s.last {
		s.seq++
	} else {
		s.seq = 0
		s.last = now
	}
	id := (now << 22) | int64(s.seq&0x3FF)<<12 | int64(uint16Fast(randUint16())&0xFFF)
	return id
}

// String returns the base32 representation of a generated ID (no padding).
func (s *Snowflake) String() string {
	id := s.Generate()
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(uint64ToBytes(uint64(id))))
}

func uint16Fast(v uint16) uint16 { return v }

func randUint16() uint16 {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint16(time.Now().UnixNano() & 0xFFFF)
	}
	return binary.BigEndian.Uint16(b[:])
}

func uint64ToBytes(v uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	return b[:]
}