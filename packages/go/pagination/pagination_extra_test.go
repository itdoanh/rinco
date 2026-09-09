// Additional tests for pagination package.
package pagination

import (
	"net/url"
	"strings"
	"testing"
)

func TestExtra_Cursor_Empty(t *testing.T) {
	c := Cursor("")
	if !c.Empty() {
		t.Error("empty cursor should be empty")
	}
}

func TestExtra_Cursor_NonEmpty(t *testing.T) {
	c := Cursor("abc")
	if c.Empty() {
		t.Error("non-empty cursor should not be empty")
	}
}

func TestExtra_EncodeCursor_Struct(t *testing.T) {
	type payload struct {
		ID string `json:"id"`
	}
	c, err := EncodeCursor(payload{ID: "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if c == "" {
		t.Error("encoded cursor should not be empty")
	}
}

func TestExtra_DecodeCursor_Roundtrip(t *testing.T) {
	type payload struct {
		ID string `json:"id"`
	}
	in := payload{ID: "test"}
	c, err := EncodeCursor(in)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var out payload
	if err := DecodeCursor(c, &out); err != nil {
		t.Errorf("decode failed: %v", err)
	}
	if out.ID != "test" {
		t.Errorf("got %s", out.ID)
	}
}

func TestExtra_DecodeCursor_Empty(t *testing.T) {
	var out map[string]any
	if err := DecodeCursor("", &out); err == nil {
		t.Error("empty cursor should error")
	}
}

func TestExtra_DecodeCursor_Tampered(t *testing.T) {
	type p struct {
		ID string `json:"id"`
	}
	c, _ := EncodeCursor(p{ID: "original"})
	// Tamper: flip a character in the middle
	s := string(c)
	if len(s) < 5 {
		t.Skip("cursor too short to tamper")
	}
	// Flip a middle character
	mid := s[len(s)/2]
	var newChar byte
	if mid == 'a' {
		newChar = 'b'
	} else {
		newChar = 'a'
	}
	tampered := s[:len(s)/2] + string(newChar) + s[len(s)/2+1:]

	var out p
	if err := DecodeCursor(Cursor(tampered), &out); err == nil {
		t.Error("tampered cursor should error")
	}
}

func TestExtra_DecodeCursor_InvalidBase64(t *testing.T) {
	var out map[string]any
	if err := DecodeCursor("not-base64!", &out); err == nil {
		t.Error("invalid base64 should error")
	}
}

func TestExtra_DecodeCursor_TooShort(t *testing.T) {
	var out map[string]any
	// 1 char is shorter than 4 (crc32)
	if err := DecodeCursor("YQ", &out); err == nil {
		t.Error("too short cursor should error")
	}
}

func TestExtra_Page_Defaults(t *testing.T) {
	p := NewPage(0, 0)
	if p.Limit != DefaultLimit {
		t.Errorf("Limit: got %d", p.Limit)
	}
}

func TestExtra_Page_MaxLimit(t *testing.T) {
	p := NewPage(0, 1000)
	if p.Limit != MaxLimit {
		t.Errorf("Limit should be clamped: got %d", p.Limit)
	}
}

func TestExtra_Page_NegativeOffset(t *testing.T) {
	p := NewPage(-5, 10)
	if p.Offset != 0 {
		t.Errorf("negative offset should be 0: got %d", p.Offset)
	}
}

func TestExtra_Page_HasMore(t *testing.T) {
	p := Page{Offset: 0, Limit: 20, Total: 100}
	if !p.HasMore() {
		t.Error("should have more")
	}

	p2 := Page{Offset: 80, Limit: 20, Total: 100}
	if p2.HasMore() {
		t.Error("should not have more")
	}
}

func TestExtra_Page_NextOffset(t *testing.T) {
	p := Page{Offset: 20, Limit: 10}
	if got := p.NextOffset(); got != 30 {
		t.Errorf("got %d", got)
	}
}

func TestExtra_ParsePage(t *testing.T) {
	v := url.Values{}
	v.Set("offset", "10")
	v.Set("limit", "50")
	p := ParsePage(v)
	if p.Offset != 10 {
		t.Errorf("offset: got %d", p.Offset)
	}
	if p.Limit != 50 {
		t.Errorf("limit: got %d", p.Limit)
	}
}

func TestExtra_ParsePage_Empty(t *testing.T) {
	p := ParsePage(url.Values{})
	if p.Limit != DefaultLimit {
		t.Errorf("limit default: got %d", p.Limit)
	}
}

func TestExtra_ParsePage_Invalid(t *testing.T) {
	v := url.Values{}
	v.Set("offset", "abc")
	v.Set("limit", "xyz")
	// Should not panic
	p := ParsePage(v)
	if p.Offset != 0 || p.Limit != DefaultLimit {
		t.Errorf("invalid values: got %v", p)
	}
}

func TestExtra_OffsetClause(t *testing.T) {
	p := Page{Offset: 10, Limit: 20}
	c := p.OffsetClause(5)
	if !strings.Contains(c, "OFFSET") || !strings.Contains(c, "LIMIT") {
		t.Errorf("clause: got %s", c)
	}
}

func TestExtra_Args(t *testing.T) {
	p := Page{Offset: 10, Limit: 20}
	args := p.Args()
	if len(args) != 2 {
		t.Errorf("expected 2 args: got %d", len(args))
	}
}

func TestExtra_CursorPage_Defaults(t *testing.T) {
	cp := NewCursorPage("", 0, false)
	if cp.Limit != DefaultLimit {
		t.Errorf("Limit: got %d", cp.Limit)
	}
}

func TestExtra_CursorPage_MaxLimit(t *testing.T) {
	cp := NewCursorPage("", 1000, false)
	if cp.Limit != MaxLimit {
		t.Errorf("Limit: got %d", cp.Limit)
	}
}

func TestExtra_ParseCursorPage(t *testing.T) {
	v := url.Values{}
	v.Set("cursor", "abc")
	v.Set("limit", "50")
	v.Set("order", "desc")
	cp := ParseCursorPage(v)
	if cp.Cursor != "abc" {
		t.Errorf("cursor: got %s", cp.Cursor)
	}
	if cp.Limit != 50 {
		t.Errorf("limit: got %d", cp.Limit)
	}
	if !cp.Desc {
		t.Error("desc should be true")
	}
}

func TestExtra_ParseCursorPage_OrderAsc(t *testing.T) {
	v := url.Values{}
	v.Set("order", "asc")
	cp := ParseCursorPage(v)
	if cp.Desc {
		t.Error("asc should be false")
	}
}

func TestExtra_Response_Basic(t *testing.T) {
	r := NewResponse([]string{"a", "b"}, 100, true)
	if len(r.Items) != 2 {
		t.Errorf("items: got %d", len(r.Items))
	}
	if !r.HasMore {
		t.Error("hasMore should be true")
	}
	if r.Total != 100 {
		t.Errorf("total: got %d", r.Total)
	}
}

func TestExtra_Response_WithNextCursor(t *testing.T) {
	r := NewResponse([]int{1, 2}, 0, true)
	r2 := r.WithNextCursor("next123")
	if r2.NextCursor != "next123" {
		t.Errorf("next cursor: got %s", r2.NextCursor)
	}
	if r.NextCursor == "next123" {
		t.Error("original should not be mutated")
	}
}

func TestExtra_Keyset_Args(t *testing.T) {
	k := Keyset{CreatedAt: 12345, ID: "id-1"}
	args := k.Args()
	if len(args) != 2 {
		t.Errorf("expected 2 args: got %d", len(args))
	}
}

func TestExtra_Keyset_ClauseDesc(t *testing.T) {
	k := Keyset{}
	c := k.ClauseDesc()
	if !strings.Contains(c, "$1") || !strings.Contains(c, "$2") {
		t.Errorf("clause should reference $1 and $2: got %s", c)
	}
}
