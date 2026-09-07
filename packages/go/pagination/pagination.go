// Package pagination provides cursor-based and offset-based pagination
// helpers that integrate with PostgreSQL (LIMIT/OFFSET, keyset) and any
// transport (HTTP query strings, gRPC).
package pagination

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/url"
	"strconv"
	"strings"
)

// ===== Defaults =====

const (
	DefaultLimit = 20
	MaxLimit     = 200
)

// ===== Cursor =====

// Cursor is an opaque base64url-encoded JSON object pointing at a row's
// position. The shape is decided by the consumer (e.g. {id, created_at}).
type Cursor string

// Empty returns true if the cursor has no value.
func (c Cursor) Empty() bool { return string(c) == "" }

// EncodeCursor marshals an arbitrary value as a JSON payload and returns a
// base64url-encoded string with a 4-byte crc32 checksum suffix. Use this to
// encode "where to start next page" markers.
func EncodeCursor(v any) (Cursor, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("pagination: encode cursor: %w", err)
	}
	// Compute crc32 checksum to detect tampering.
	sum := crc32.ChecksumIEEE(b)
	sumBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sumBytes, sum)
	payload := append(b, sumBytes...)
	return Cursor(base64.RawURLEncoding.EncodeToString(payload)), nil
}

// DecodeCursor parses a cursor produced by EncodeCursor into out.
// Returns an error if the cursor is empty, malformed, or tampered with.
func DecodeCursor(c Cursor, out any) error {
	if c.Empty() {
		return errors.New("pagination: cursor is empty")
	}
	raw, err := base64.RawURLEncoding.DecodeString(string(c))
	if err != nil {
		return fmt.Errorf("pagination: invalid cursor encoding: %w", err)
	}
	if len(raw) < 4 {
		return errors.New("pagination: cursor too short")
	}
	body := raw[:len(raw)-4]
	gotSum := binary.BigEndian.Uint32(raw[len(raw)-4:])
	wantSum := crc32.ChecksumIEEE(body)
	if gotSum != wantSum {
		return errors.New("pagination: cursor checksum mismatch (tampered)")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("pagination: invalid cursor json: %w", err)
	}
	return nil
}

// ===== Offset =====

// Page describes an offset/limit pagination request and (optionally) the
// total row count returned by the caller.
type Page struct {
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total,omitempty"`
}

// NewPage returns a page with defaults applied and offset/limit clamped to
// safe bounds.
func NewPage(offset, limit int) Page {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return Page{Offset: offset, Limit: limit}
}

// ParsePage parses standard ?offset=&limit= query parameters.
func ParsePage(values url.Values) Page {
	offset, _ := strconv.Atoi(values.Get("offset"))
	limit, _ := strconv.Atoi(values.Get("limit"))
	return NewPage(offset, limit)
}

// OffsetClause returns a SQL "OFFSET $1 LIMIT $2" fragment (with appropriate
// placeholders starting at startIdx).
func (p Page) OffsetClause(startIdx int) string {
	return fmt.Sprintf("OFFSET $%d LIMIT $%d", startIdx, startIdx+1)
}

// Args returns the values to bind to placeholders from OffsetClause.
func (p Page) Args() []any {
	return []any{p.Offset, p.Limit}
}

// HasMore returns true if there are likely more rows after this page.
func (p Page) HasMore() bool {
	return int64(p.Offset+p.Limit) < p.Total
}

// NextOffset returns the offset to request the next page.
func (p Page) NextOffset() int { return p.Offset + p.Limit }

// ===== Cursor-based params =====

// CursorPage describes a cursor/limit pagination request.
type CursorPage struct {
	Cursor Cursor
	Limit  int
	Desc   bool
}

// NewCursorPage applies defaults and clamp.
func NewCursorPage(c Cursor, limit int, desc bool) CursorPage {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return CursorPage{Cursor: c, Limit: limit, Desc: desc}
}

// ParseCursorPage parses ?cursor=&limit=&order=.
func ParseCursorPage(values url.Values) CursorPage {
	limit, _ := strconv.Atoi(values.Get("limit"))
	cursor := Cursor(values.Get("cursor"))
	desc := strings.EqualFold(values.Get("order"), "desc")
	return NewCursorPage(cursor, limit, desc)
}

// ===== Standardized response =====

// Response is a generic envelope for paginated lists.
type Response[T any] struct {
	Items      []T    `json:"items"`
	NextCursor Cursor `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
	Total      int64  `json:"total,omitempty"`
}

// NewResponse builds a Response from items + an optional total.
// The next cursor is left empty unless the caller knows there are more rows.
func NewResponse[T any](items []T, total int64, hasMore bool) Response[T] {
	return Response[T]{
		Items:   items,
		HasMore: hasMore,
		Total:   total,
	}
}

// WithNextCursor returns a copy of the response with a cursor attached.
func (r Response[T]) WithNextCursor(c Cursor) Response[T] {
	r.NextCursor = c
	return r
}

// ===== Keyset SQL helpers =====

// Keyset encodes the typical (created_at, id) cursor pair.
type Keyset struct {
	CreatedAt int64  `json:"t,omitempty"`
	ID        string `json:"id,omitempty"`
}

// KeysetClause returns a SQL WHERE clause fragment for keyset pagination
// assuming rows are ordered by (created_at, id) DESC. The first placeholder
// is cursorCreatedAt and the second is cursorID.
func (k Keyset) ClauseDesc() string {
	return "(created_at, id) < ($1, $2)"
}

// Args returns the values to bind to placeholders from ClauseDesc.
func (k Keyset) Args() []any {
	return []any{k.CreatedAt, k.ID}
}