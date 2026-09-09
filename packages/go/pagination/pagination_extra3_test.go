// Extra tests for pagination package.
package pagination

import (
	"net/url"
	"strings"
	"testing"
)

func TestCursor_Empty_True(t *testing.T) {
	c := Cursor("")
	if !c.Empty() {
		t.Error("empty cursor should be empty")
	}
}

func TestCursor_Empty_False(t *testing.T) {
	c := Cursor("abc")
	if c.Empty() {
		t.Error("non-empty cursor should not be empty")
	}
}

func TestEncodeCursor_Object(t *testing.T) {
	type pos struct {
		ID string `json:"id"`
		T  int64  `json:"t"`
	}
	c, err := EncodeCursor(pos{ID: "abc", T: 12345})
	if err != nil {
		t.Fatal(err)
	}
	if c.Empty() {
		t.Error("encoded cursor should not be empty")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	type pos struct {
		ID string `json:"id"`
		T  int64  `json:"t"`
	}
	original := pos{ID: "xyz", T: 999}
	c, err := EncodeCursor(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pos
	if err := DecodeCursor(c, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != original {
		t.Errorf("got %+v, want %+v", decoded, original)
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	var out struct{}
	if err := DecodeCursor(Cursor(""), &out); err == nil {
		t.Error("expected error for empty cursor")
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	var out struct{}
	if err := DecodeCursor(Cursor("!!!not-base64!!!"), &out); err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecodeCursor_TooShort(t *testing.T) {
	// base64url encode of "ab" = "YWI"
	var out struct{}
	if err := DecodeCursor(Cursor("YWI"), &out); err == nil {
		t.Error("expected error for short cursor")
	}
}

func TestDecodeCursor_Tampered(t *testing.T) {
	type pos struct {
		ID string `json:"id"`
	}
	c, _ := EncodeCursor(pos{ID: "abc"})
	// Try to tamper: pick a position guaranteed not to be a valid base64 -> checksum mismatch
	tampered := Cursor(string(c) + "X")
	var out pos
	if err := DecodeCursor(tampered, &out); err == nil {
		t.Error("expected error for tampered cursor")
	}
}

func TestNewPage_Defaults(t *testing.T) {
	p := NewPage(0, 0)
	if p.Offset != 0 {
		t.Errorf("offset: %d", p.Offset)
	}
	if p.Limit != DefaultLimit {
		t.Errorf("limit: %d", p.Limit)
	}
}

func TestNewPage_NegativeOffset(t *testing.T) {
	p := NewPage(-5, 10)
	if p.Offset != 0 {
		t.Errorf("negative offset should be 0: %d", p.Offset)
	}
}

func TestNewPage_ExceedsMax(t *testing.T) {
	p := NewPage(0, 1000)
	if p.Limit != MaxLimit {
		t.Errorf("limit should be capped: %d", p.Limit)
	}
}

func TestParsePage(t *testing.T) {
	v := url.Values{}
	v.Set("offset", "10")
	v.Set("limit", "20")
	p := ParsePage(v)
	if p.Offset != 10 || p.Limit != 20 {
		t.Errorf("got offset=%d limit=%d", p.Offset, p.Limit)
	}
}

func TestParsePage_NoValues(t *testing.T) {
	p := ParsePage(url.Values{})
	if p.Offset != 0 {
		t.Error("default offset")
	}
}

func TestParsePage_InvalidValues(t *testing.T) {
	v := url.Values{}
	v.Set("offset", "abc")
	v.Set("limit", "xyz")
	p := ParsePage(v)
	if p.Offset != 0 || p.Limit != DefaultLimit {
		t.Errorf("invalid values should default")
	}
}

func TestPage_OffsetClause(t *testing.T) {
	p := NewPage(0, 10)
	clause := p.OffsetClause(1)
	if clause != "OFFSET $1 LIMIT $2" {
		t.Errorf("got %s", clause)
	}
}

func TestPage_OffsetClause_StartIdx(t *testing.T) {
	p := NewPage(0, 10)
	clause := p.OffsetClause(5)
	if clause != "OFFSET $5 LIMIT $6" {
		t.Errorf("got %s", clause)
	}
}

func TestPage_Args(t *testing.T) {
	p := NewPage(10, 20)
	args := p.Args()
	if len(args) != 2 {
		t.Errorf("args len: %d", len(args))
	}
	if args[0] != 10 || args[1] != 20 {
		t.Error("args mismatch")
	}
}

func TestPage_HasMore_True(t *testing.T) {
	p := Page{Offset: 0, Limit: 10, Total: 50}
	if !p.HasMore() {
		t.Error("should have more")
	}
}

func TestPage_HasMore_False(t *testing.T) {
	p := Page{Offset: 0, Limit: 10, Total: 5}
	if p.HasMore() {
		t.Error("should not have more")
	}
}

func TestPage_HasMore_Exact(t *testing.T) {
	p := Page{Offset: 0, Limit: 10, Total: 10}
	if p.HasMore() {
		t.Error("exact match should not have more")
	}
}

func TestPage_HasMore_ZeroTotal(t *testing.T) {
	p := Page{Offset: 0, Limit: 10, Total: 0}
	if p.HasMore() {
		t.Error("zero total should not have more")
	}
}

func TestPage_NextOffset(t *testing.T) {
	p := Page{Offset: 10, Limit: 20}
	if p.NextOffset() != 30 {
		t.Errorf("got %d", p.NextOffset())
	}
}

func TestNewCursorPage_Defaults(t *testing.T) {
	cp := NewCursorPage("", 0, false)
	if cp.Limit != DefaultLimit {
		t.Errorf("limit: %d", cp.Limit)
	}
	if cp.Desc {
		t.Error("desc should be false")
	}
}

func TestNewCursorPage_MaxLimit(t *testing.T) {
	cp := NewCursorPage("", 1000, true)
	if cp.Limit != MaxLimit {
		t.Errorf("limit should be capped: %d", cp.Limit)
	}
}

func TestParseCursorPage(t *testing.T) {
	v := url.Values{}
	v.Set("cursor", "abc")
	v.Set("limit", "50")
	v.Set("order", "desc")
	cp := ParseCursorPage(v)
	if cp.Cursor != Cursor("abc") {
		t.Errorf("cursor: %s", cp.Cursor)
	}
	if cp.Limit != 50 {
		t.Errorf("limit: %d", cp.Limit)
	}
	if !cp.Desc {
		t.Error("desc should be true")
	}
}

func TestParseCursorPage_NoValues(t *testing.T) {
	cp := ParseCursorPage(url.Values{})
	if !cp.Cursor.Empty() {
		t.Error("cursor should be empty")
	}
	if cp.Desc {
		t.Error("desc should default to false")
	}
}

func TestParseCursorPage_Asc(t *testing.T) {
	v := url.Values{}
	v.Set("order", "asc")
	cp := ParseCursorPage(v)
	if cp.Desc {
		t.Error("asc should not be desc")
	}
}

func TestResponse_Basic(t *testing.T) {
	r := NewResponse([]string{"a", "b"}, 2, false)
	if len(r.Items) != 2 {
		t.Errorf("items: %d", len(r.Items))
	}
	if r.HasMore {
		t.Error("has_more should be false")
	}
	if r.Total != 2 {
		t.Errorf("total: %d", r.Total)
	}
}

func TestResponse_WithNextCursor(t *testing.T) {
	r := NewResponse([]int{1}, 10, true)
	r2 := r.WithNextCursor(Cursor("next"))
	if r2.NextCursor.Empty() {
		t.Error("cursor should be set")
	}
	if !r2.HasMore {
		t.Error("has_more should still be true")
	}
}

func TestResponse_GenericString(t *testing.T) {
	r := NewResponse([]string{"x"}, 1, false)
	if r.Items[0] != "x" {
		t.Error("item")
	}
}

func TestKeyset_Args(t *testing.T) {
	k := Keyset{CreatedAt: 12345, ID: "abc"}
	args := k.Args()
	if len(args) != 2 {
		t.Errorf("args len: %d", len(args))
	}
	if args[0] != int64(12345) {
		t.Error("created_at")
	}
	if args[1] != "abc" {
		t.Error("id")
	}
}

func TestKeyset_ClauseDesc(t *testing.T) {
	k := Keyset{}
	clause := k.ClauseDesc()
	if !strings.Contains(clause, "(created_at, id) <") {
		t.Errorf("got %s", clause)
	}
}

func TestConstants(t *testing.T) {
	if DefaultLimit <= 0 {
		t.Error("DefaultLimit should be positive")
	}
	if MaxLimit <= DefaultLimit {
		t.Error("MaxLimit should be greater than DefaultLimit")
	}
}
