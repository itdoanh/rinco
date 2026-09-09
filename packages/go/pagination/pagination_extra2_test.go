// Tests for pagination package (pagination.go).
package pagination

import (
	"net/url"
	"testing"
)

func TestExtra2_Cursor_Empty(t *testing.T) {
	c := Cursor("")
	if !c.Empty() {
		t.Error("empty cursor should return true")
	}
	
	c2 := Cursor("abc")
	if c2.Empty() {
		t.Error("non-empty cursor should return false")
	}
}

func TestExtra2_EncodeCursor_DecodeCursor(t *testing.T) {
	input := map[string]any{"id": "123", "created_at": 1704067200}
	
	cursor, err := EncodeCursor(input)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if cursor == "" {
		t.Fatal("cursor should not be empty")
	}
	
	var output map[string]any
	err = DecodeCursor(cursor, &output)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	
	if output["id"] != "123" {
		t.Errorf("id: got %v", output["id"])
	}
}

func TestExtra2_EncodeCursor_Empty(t *testing.T) {
	_, err := EncodeCursor(nil)
	if err != nil {
		t.Errorf("encode nil: %v", err)
	}
}

func TestExtra2_DecodeCursor_Empty(t *testing.T) {
	err := DecodeCursor(Cursor(""), nil)
	if err == nil {
		t.Error("expected error for empty cursor")
	}
}

func TestExtra2_DecodeCursor_InvalidBase64(t *testing.T) {
	err := DecodeCursor(Cursor("!!!"), nil)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestExtra2_DecodeCursor_TooShort(t *testing.T) {
	// Need at least 4 bytes for checksum
	c := Cursor("aBcD") // 4 chars but only 3 bytes
	err := DecodeCursor(c, nil)
	if err == nil {
		t.Error("expected error for short cursor")
	}
}

func TestExtra2_DecodeCursor_Tampered(t *testing.T) {
	input := map[string]any{"id": "123"}
	cursor, _ := EncodeCursor(input)
	
	// Tamper with the cursor by modifying a character
	tampered := string(cursor) + "X"
	err := DecodeCursor(Cursor(tampered), nil)
	if err == nil {
		t.Error("expected error for tampered cursor")
	}
}

func TestExtra2_DecodeCursor_WrongType(t *testing.T) {
	input := map[string]any{"id": "123"}
	cursor, _ := EncodeCursor(input)
	
	var wrongType string
	err := DecodeCursor(cursor, &wrongType)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestExtra2_DefaultConstants(t *testing.T) {
	if DefaultLimit == 0 {
		t.Error("DefaultLimit should not be 0")
	}
	if MaxLimit == 0 {
		t.Error("MaxLimit should not be 0")
	}
	if MaxLimit < DefaultLimit {
		t.Error("MaxLimit should be >= DefaultLimit")
	}
}

func TestExtra2_NewPage_Defaults(t *testing.T) {
	p := NewPage(0, 0)
	if p.Limit != DefaultLimit {
		t.Errorf("Limit: got %d", p.Limit)
	}
}

func TestExtra2_NewPage_ClampMax(t *testing.T) {
	p := NewPage(0, MaxLimit+100)
	if p.Limit != MaxLimit {
		t.Errorf("Limit should be clamped to MaxLimit: got %d", p.Limit)
	}
}

func TestExtra2_NewPage_NegativeOffset(t *testing.T) {
	p := NewPage(-10, 20)
	if p.Offset != 0 {
		t.Errorf("Offset: got %d", p.Offset)
	}
}

func TestExtra2_NewPage_NegativeLimit(t *testing.T) {
	p := NewPage(0, -5)
	if p.Limit != DefaultLimit {
		t.Errorf("Limit: got %d", p.Limit)
	}
}

func TestExtra2_NewPage_Valid(t *testing.T) {
	p := NewPage(10, 50)
	if p.Offset != 10 {
		t.Errorf("Offset: got %d", p.Offset)
	}
	if p.Limit != 50 {
		t.Errorf("Limit: got %d", p.Limit)
	}
}

func TestExtra2_ParsePage(t *testing.T) {
	values := url.Values{
		"offset": []string{"100"},
		"limit":  []string{"50"},
	}
	p := ParsePage(values)
	if p.Offset != 100 {
		t.Errorf("Offset: got %d", p.Offset)
	}
	if p.Limit != 50 {
		t.Errorf("Limit: got %d", p.Limit)
	}
}

func TestExtra2_ParsePage_Empty(t *testing.T) {
	p := ParsePage(url.Values{})
	if p.Limit != DefaultLimit {
		t.Errorf("Limit should default: got %d", p.Limit)
	}
}

func TestExtra2_ParsePage_Invalid(t *testing.T) {
	values := url.Values{
		"offset": []string{"abc"},
		"limit":  []string{"xyz"},
	}
	p := ParsePage(values)
	if p.Offset != 0 {
		t.Errorf("Invalid offset should be 0: got %d", p.Offset)
	}
}

func TestExtra2_Page_OffsetClause(t *testing.T) {
	p := Page{Offset: 10, Limit: 20}
	clause := p.OffsetClause(1)
	if clause == "" {
		t.Error("OffsetClause should not be empty")
	}
}

func TestExtra2_Page_Args(t *testing.T) {
	p := Page{Offset: 10, Limit: 20}
	args := p.Args()
	if len(args) != 2 {
		t.Errorf("Args should return 2 values: got %d", len(args))
	}
	if args[0].(int) != 10 {
		t.Errorf("Offset: got %v", args[0])
	}
}

func TestExtra2_Page_HasMore_True(t *testing.T) {
	p := Page{Offset: 10, Limit: 20, Total: 50}
	if !p.HasMore() {
		t.Error("HasMore should be true")
	}
}

func TestExtra2_Page_HasMore_False(t *testing.T) {
	p := Page{Offset: 40, Limit: 20, Total: 50}
	if p.HasMore() {
		t.Error("HasMore should be false")
	}
}

func TestExtra2_Page_HasMore_Exact(t *testing.T) {
	p := Page{Offset: 30, Limit: 20, Total: 50}
	if p.HasMore() {
		t.Error("HasMore should be false when exact")
	}
}

func TestExtra2_Page_NextOffset(t *testing.T) {
	p := Page{Offset: 10, Limit: 20}
	if p.NextOffset() != 30 {
		t.Errorf("NextOffset: got %d", p.NextOffset())
	}
}

func TestExtra2_NewCursorPage_Defaults(t *testing.T) {
	cp := NewCursorPage(Cursor(""), 0, false)
	if cp.Limit != DefaultLimit {
		t.Errorf("Limit: got %d", cp.Limit)
	}
}

func TestExtra2_NewCursorPage_Valid(t *testing.T) {
	cp := NewCursorPage(Cursor("abc"), 50, true)
	if cp.Limit != 50 {
		t.Errorf("Limit: got %d", cp.Limit)
	}
	if !cp.Desc {
		t.Error("Desc should be true")
	}
}

func TestExtra2_NewCursorPage_ClampMax(t *testing.T) {
	cp := NewCursorPage(Cursor(""), MaxLimit+100, false)
	if cp.Limit != MaxLimit {
		t.Errorf("Limit: got %d", cp.Limit)
	}
}

func TestExtra2_ParseCursorPage(t *testing.T) {
	values := url.Values{
		"cursor": []string{"abc123"},
		"limit":  []string{"50"},
		"order":  []string{"desc"},
	}
	cp := ParseCursorPage(values)
	if string(cp.Cursor) != "abc123" {
		t.Errorf("Cursor: got %s", cp.Cursor)
	}
	if cp.Limit != 50 {
		t.Errorf("Limit: got %d", cp.Limit)
	}
	if !cp.Desc {
		t.Error("Desc should be true")
	}
}

func TestExtra2_NewResponse(t *testing.T) {
	items := []string{"a", "b", "c"}
	resp := NewResponse(items, 10, true)
	if len(resp.Items) != 3 {
		t.Errorf("Items: got %d", len(resp.Items))
	}
	if !resp.HasMore {
		t.Error("HasMore should be true")
	}
	if resp.Total != 10 {
		t.Errorf("Total: got %d", resp.Total)
	}
}

func TestExtra2_Response_WithNextCursor(t *testing.T) {
	items := []string{"a"}
	resp := NewResponse(items, 10, true)
	cursor := Cursor("next123")
	resp2 := resp.WithNextCursor(cursor)
	if resp2.NextCursor != cursor {
		t.Error("NextCursor not set")
	}
}

func TestExtra2_Keyset_ClauseDesc(t *testing.T) {
	k := Keyset{CreatedAt: 1704067200, ID: "abc123"}
	clause := k.ClauseDesc()
	if clause == "" {
		t.Error("ClauseDesc should not be empty")
	}
}

func TestExtra2_Keyset_Args(t *testing.T) {
	k := Keyset{CreatedAt: 1704067200, ID: "abc123"}
	args := k.Args()
	if len(args) != 2 {
		t.Errorf("Args: got %d", len(args))
	}
	if args[0].(int64) != 1704067200 {
		t.Errorf("CreatedAt: got %v", args[0])
	}
	if args[1].(string) != "abc123" {
		t.Errorf("ID: got %v", args[1])
	}
}

func TestExtra2_Keyset_Empty(t *testing.T) {
	k := Keyset{}
	args := k.Args()
	if args[0].(int64) != 0 {
		t.Error("Empty CreatedAt should be 0")
	}
}

func TestExtra2_Keyset_ClauseDesc_ContainsCreatedAt(t *testing.T) {
	k := Keyset{CreatedAt: 1704067200}
	clause := k.ClauseDesc()
	// Just verify it doesn't panic
	_ = clause
}
