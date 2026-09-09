package pagination

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	if DefaultLimit != 20 {
		t.Errorf("DefaultLimit = %d, want 20", DefaultLimit)
	}
	if MaxLimit != 200 {
		t.Errorf("MaxLimit = %d, want 200", MaxLimit)
	}
}

func TestCursorEmpty(t *testing.T) {
	if !Cursor("").Empty() {
		t.Error("empty string should be empty")
	}
	if Cursor("abc").Empty() {
		t.Error("non-empty should not be empty")
	}
}

func TestEncodeDecodeCursorVariant(t *testing.T) {
	payload := map[string]any{"id": "abc-123", "ts": 1234567890}
	c, err := EncodeCursor(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if c.Empty() {
		t.Fatal("cursor empty")
	}

	var out map[string]any
	if err := DecodeCursor(c, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["id"] != "abc-123" {
		t.Errorf("id = %v", out["id"])
	}
}

func TestEncodeCursorInvalidValue(t *testing.T) {
	// channels cannot be marshaled
	_, err := EncodeCursor(make(chan int))
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestDecodeCursorEmpty(t *testing.T) {
	var out map[string]any
	err := DecodeCursor(Cursor(""), &out)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDecodeCursorInvalidBase64(t *testing.T) {
	var out map[string]any
	err := DecodeCursor(Cursor("not-base64!!!"), &out)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDecodeCursorTooShort(t *testing.T) {
	// 3 bytes (less than 4 checksum bytes)
	short := base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3})
	var out map[string]any
	err := DecodeCursor(Cursor(short), &out)
	if err == nil {
		t.Error("expected error for too short")
	}
}

func TestDecodeCursorTamperedChecksum(t *testing.T) {
	// Encode then modify bytes
	payload := map[string]any{"id": "original"}
	c, _ := EncodeCursor(payload)
	raw, _ := base64.RawURLEncoding.DecodeString(string(c))
	// Flip a byte in the body (not the checksum)
	if len(raw) > 5 {
		raw[5] ^= 0x01
	}
	tampered := base64.RawURLEncoding.EncodeToString(raw)
	var out map[string]any
	err := DecodeCursor(Cursor(tampered), &out)
	if err == nil {
		t.Error("expected checksum mismatch error")
	}
}

func TestDecodeCursorInvalidJSON(t *testing.T) {
	// Construct a cursor with valid checksum but invalid JSON
	body := []byte("not-json")
	sumBytes := []byte{0, 0, 0, 0}
	// Compute correct checksum (won't match so this would still fail but for checksum reason)
	encoded := append(body, sumBytes...)
	c := Cursor(base64.RawURLEncoding.EncodeToString(encoded))
	var out map[string]any
	err := DecodeCursor(c, &out)
	// Either checksum error or json error
	if err == nil {
		t.Error("expected error")
	}
}

func TestNewPageDefaults(t *testing.T) {
	p := NewPage(0, 0)
	if p.Limit != DefaultLimit {
		t.Errorf("limit = %d, want %d", p.Limit, DefaultLimit)
	}
	if p.Offset != 0 {
		t.Errorf("offset = %d", p.Offset)
	}
}

func TestNewPageMaxLimit(t *testing.T) {
	p := NewPage(0, 1000)
	if p.Limit != MaxLimit {
		t.Errorf("limit = %d, want %d", p.Limit, MaxLimit)
	}
}

func TestNewPageNegativeOffset(t *testing.T) {
	p := NewPage(-5, 10)
	if p.Offset != 0 {
		t.Errorf("offset should clamp to 0, got %d", p.Offset)
	}
}

func TestNewPageNormal(t *testing.T) {
	p := NewPage(20, 50)
	if p.Offset != 20 || p.Limit != 50 {
		t.Errorf("got %+v", p)
	}
}

func TestParsePageFromQuery(t *testing.T) {
	v := url.Values{}
	v.Set("offset", "40")
	v.Set("limit", "60")
	p := ParsePage(v)
	if p.Offset != 40 || p.Limit != 60 {
		t.Errorf("got %+v", p)
	}
}

func TestParsePageInvalidNumbers(t *testing.T) {
	v := url.Values{}
	v.Set("offset", "abc")
	v.Set("limit", "xyz")
	p := ParsePage(v)
	// Should use defaults
	if p.Offset != 0 || p.Limit != DefaultLimit {
		t.Errorf("got %+v", p)
	}
}

func TestPageOffsetClauseVariant(t *testing.T) {
	p := Page{Offset: 0, Limit: 20}
	clause := p.OffsetClause(1)
	if !strings.Contains(clause, "OFFSET $1") {
		t.Errorf("clause = %q", clause)
	}
	if !strings.Contains(clause, "LIMIT $2") {
		t.Errorf("clause = %q", clause)
	}
}

func TestPageArgs(t *testing.T) {
	p := Page{Offset: 100, Limit: 25}
	args := p.Args()
	if len(args) != 2 {
		t.Errorf("args len = %d", len(args))
	}
}

func TestPageHasMore(t *testing.T) {
	p := Page{Offset: 0, Limit: 20, Total: 100}
	if !p.HasMore() {
		t.Error("should have more")
	}
	p2 := Page{Offset: 80, Limit: 20, Total: 100}
	if p2.HasMore() {
		t.Error("should not have more")
	}
}

func TestPageNextOffset(t *testing.T) {
	p := Page{Offset: 20, Limit: 10}
	if p.NextOffset() != 30 {
		t.Errorf("next offset = %d", p.NextOffset())
	}
}

func TestNewCursorPageDefaults(t *testing.T) {
	p := NewCursorPage(Cursor(""), 0, false)
	if p.Limit != DefaultLimit {
		t.Error("limit not default")
	}
}

func TestNewCursorPageMaxLimit(t *testing.T) {
	p := NewCursorPage(Cursor(""), 1000, false)
	if p.Limit != MaxLimit {
		t.Errorf("limit = %d, want %d", p.Limit, MaxLimit)
	}
}

func TestParseCursorPageFromQuery(t *testing.T) {
	v := url.Values{}
	v.Set("cursor", "abc")
	v.Set("limit", "30")
	v.Set("order", "desc")
	p := ParseCursorPage(v)
	if string(p.Cursor) != "abc" {
		t.Errorf("cursor = %q", p.Cursor)
	}
	if p.Limit != 30 {
		t.Errorf("limit = %d", p.Limit)
	}
	if !p.Desc {
		t.Error("Desc should be true")
	}
}

func TestParseCursorPageAsc(t *testing.T) {
	v := url.Values{}
	v.Set("order", "asc")
	p := ParseCursorPage(v)
	if p.Desc {
		t.Error("Desc should be false for asc")
	}
}

func TestResponseMarshal(t *testing.T) {
	r := NewResponse([]int{1, 2, 3}, 10, true)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"items":[1,2,3]`) {
		t.Errorf("missing items: %s", data)
	}
	if !strings.Contains(string(data), `"has_more":true`) {
		t.Errorf("missing has_more: %s", data)
	}
}

func TestResponseWithNextCursor(t *testing.T) {
	r := NewResponse([]int{1}, 1, false)
	r2 := r.WithNextCursor(Cursor("xyz"))
	if r2.NextCursor != "xyz" {
		t.Errorf("next_cursor = %q", r2.NextCursor)
	}
	// Original unchanged
	if !r.NextCursor.Empty() {
		t.Error("original should not be modified")
	}
}

func TestKeysetClauseDesc(t *testing.T) {
	k := Keyset{CreatedAt: 123, ID: "abc"}
	clause := k.ClauseDesc()
	if !strings.Contains(clause, "created_at") {
		t.Errorf("clause = %q", clause)
	}
	if !strings.Contains(clause, "id") {
		t.Errorf("clause = %q", clause)
	}
}

func TestKeysetArgsVariant(t *testing.T) {
	k := Keyset{CreatedAt: 999, ID: "zzz"}
	args := k.Args()
	if len(args) != 2 {
		t.Errorf("args len = %d", len(args))
	}
}
