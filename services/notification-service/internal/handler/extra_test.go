// Tests for notification-service handler helpers.
package handler

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// mustJSON
// =============================================================================

func TestExtraMustJSON_Struct(t *testing.T) {
	type S struct {
		Name string `json:"name"`
	}
	got := mustJSON(S{Name: "alice"})
	if !strings.Contains(string(got), `"name":"alice"`) {
		t.Errorf("got %s", string(got))
	}
}

func TestExtraMustJSON_Map(t *testing.T) {
	got := mustJSON(map[string]any{"x": 1, "y": "z"})
	if !strings.Contains(string(got), `"x":1`) {
		t.Errorf("got %s", string(got))
	}
}

func TestExtraMustJSON_NilValue(t *testing.T) {
	got := mustJSON(nil)
	if string(got) != "null" {
		t.Errorf("nil marshal: got %s, want null", got)
	}
}

func TestExtraMustJSON_String(t *testing.T) {
	got := mustJSON("hello")
	if string(got) != `"hello"` {
		t.Errorf("string marshal: got %s", got)
	}
}

// =============================================================================
// nullString
// =============================================================================

func TestExtraNullString_EmptyReturnsNil(t *testing.T) {
	if got := nullString(""); got != nil {
		t.Errorf("empty: got %v, want nil", got)
	}
}

func TestExtraNullString_NonEmptyReturnsString(t *testing.T) {
	got := nullString("hello")
	if s, ok := got.(string); !ok || s != "hello" {
		t.Errorf("got %v, want hello", got)
	}
}

func TestExtraNullString_WhitespaceNotEmpty(t *testing.T) {
	// Whitespace-only string is NOT empty.
	got := nullString(" ")
	if got == nil {
		t.Error("whitespace should be non-nil")
	}
}

// =============================================================================
// tsOrNil
// =============================================================================

func TestExtraTsOrNil_TrueReturnsPointer(t *testing.T) {
	got := tsOrNil(true)
	if got == nil {
		t.Fatal("expected non-nil when ok=true")
	}
	tp, ok := got.(*time.Time)
	if !ok {
		t.Fatalf("got %T, want *time.Time", got)
	}
	if tp.IsZero() {
		t.Error("expected non-zero time")
	}
}

func TestExtraTsOrNil_FalseReturnsNil(t *testing.T) {
	if got := tsOrNil(false); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// =============================================================================
// nilIfZero
// =============================================================================

func TestExtraNilIfZero_NilPointer(t *testing.T) {
	if got := nilIfZero(nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestExtraNilIfZero_ZeroTime(t *testing.T) {
	zero := time.Time{}
	if got := nilIfZero(&zero); got != nil {
		t.Errorf("expected nil for zero time, got %v", got)
	}
}

func TestExtraNilIfZero_NonZeroTime(t *testing.T) {
	now := time.Now()
	if got := nilIfZero(&now); got == nil {
		t.Error("expected non-nil for non-zero time")
	}
}

// =============================================================================
// firstNonEmpty
// =============================================================================

func TestExtraFirstNonEmptyHandler(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"a", "b"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", ""}, ""},
		{[]string{"", "", "x"}, "x"},
		{nil, ""},
	}
	for _, c := range cases {
		if got := firstNonEmpty(c.in...); got != c.want {
			t.Errorf("firstNonEmpty(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// =============================================================================
// broadcastReq / broadcastAudience edge cases
// =============================================================================

func TestExtraBroadcastRequest_BodyOnlyRequired(t *testing.T) {
	// Only type/title/body required by struct (no validation tags), so
	// any valid JSON parses.
	body := `{"type":"x","title":"y","body":"z"}`
	var req broadcastReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.Type != "x" || req.Title != "y" || req.Body != "z" {
		t.Errorf("unexpected: %+v", req)
	}
	if req.Priority != "" {
		t.Errorf("Priority default should be empty, got %q", req.Priority)
	}
}

func TestExtraBroadcastAudience_AllUserIDs(t *testing.T) {
	body := `{"type":"x","title":"y","body":"z","audience":{"user_ids":["a","b","c"]}}`
	var req broadcastReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if len(req.Audience.UserIDs) != 3 {
		t.Errorf("UserIDs: %v", req.Audience.UserIDs)
	}
}

func TestExtraBroadcastAudience_AllRole(t *testing.T) {
	body := `{"type":"x","title":"y","body":"z","audience":{"role":"admin","department":"engineering"}}`
	var req broadcastReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.Audience.Role != "admin" {
		t.Errorf("Role: %q", req.Audience.Role)
	}
	if req.Audience.Department != "engineering" {
		t.Errorf("Department: %q", req.Audience.Department)
	}
}

func TestExtraMustJSON_ValidJSON(t *testing.T) {
	// The result of mustJSON should always be parseable JSON.
	got := mustJSON(map[string]string{"k": "v"})
	var back map[string]string
	if err := json.Unmarshal(got, &back); err != nil {
		t.Errorf("mustJSON output not parseable: %v", err)
	}
	if back["k"] != "v" {
		t.Errorf("round-trip: %v", back)
	}
}

func TestExtraMustJSON_EmptySlice(t *testing.T) {
	got := mustJSON([]int{})
	// Empty slice marshals to "[]".
	if !bytes.Equal(got, []byte("[]")) {
		t.Errorf("got %s", string(got))
	}
}
