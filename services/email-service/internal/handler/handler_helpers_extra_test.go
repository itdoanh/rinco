// Additional tests for email-service handler.go helpers.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestMustJSON_Slice(t *testing.T) {
	got := mustJSON([]string{"a", "b"})
	if string(got) != `["a","b"]` {
		t.Errorf("got %s", string(got))
	}
}

func TestMustJSON_Map(t *testing.T) {
	got := mustJSON(map[string]any{"x": 1})
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["x"].(float64) != 1 {
		t.Errorf("got %v", m["x"])
	}
}

func TestMustJSON_Empty(t *testing.T) {
	// nil marshals as "null", not "[]"
	got := mustJSON(nil)
	if string(got) != "null" {
		t.Errorf("got %s", string(got))
	}
}

func TestMustJSON_InvalidFallsBack(t *testing.T) {
	// channels can't be marshalled; should fall back to [].
	got := mustJSON(make(chan int))
	if string(got) != "[]" {
		t.Errorf("got %s", string(got))
	}
}

func TestNullableString_Empty(t *testing.T) {
	if nullableString("") != nil {
		t.Errorf("expected nil")
	}
}

func TestNullableString_NonEmpty(t *testing.T) {
	v := nullableString("hello")
	if v == nil {
		t.Fatal("expected non-nil")
	}
	if v.(string) != "hello" {
		t.Errorf("got %v", v)
	}
}

func TestErrMsg_Nil(t *testing.T) {
	if errMsg(nil) != "" {
		t.Errorf("expected empty")
	}
}

func TestErrMsg_WithError(t *testing.T) {
	if errMsg(errors.New("boom")) != "boom" {
		t.Errorf("got %v", errMsg(errors.New("boom")))
	}
}

func TestFirstNonEmpty_PicksFirst(t *testing.T) {
	if firstNonEmpty("a", "b", "c") != "a" {
		t.Errorf("got %v", firstNonEmpty("a", "b", "c"))
	}
}

func TestFirstNonEmpty_SkipsEmpty(t *testing.T) {
	if firstNonEmpty("", "", "b") != "b" {
		t.Errorf("got %v", firstNonEmpty("", "", "b"))
	}
}

func TestFirstNonEmpty_AllEmpty(t *testing.T) {
	if firstNonEmpty("", "", "") != "" {
		t.Errorf("got %v", firstNonEmpty("", "", ""))
	}
}

func TestFirstNonEmpty_NoArgs(t *testing.T) {
	if firstNonEmpty() != "" {
		t.Errorf("got %v", firstNonEmpty())
	}
}

func TestDecodeBase64_Empty(t *testing.T) {
	_, err := decodeBase64("")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeBase64_Invalid(t *testing.T) {
	_, err := decodeBase64("@@@@")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeBase64_Valid(t *testing.T) {
	got, err := decodeBase64("aGVsbG8=")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("got %s", string(got))
	}
}

func TestEnsureQueryEscape(t *testing.T) {
	got := ensureQueryEscape("hello world")
	if got != "hello+world" {
		t.Errorf("got %s", got)
	}
}

func TestReadMsgID_MsgIDKey(t *testing.T) {
	id, ok := readMsgID([]byte(`{"msg_id":"abc"}`))
	if !ok || id != "abc" {
		t.Errorf("got %s %v", id, ok)
	}
}

func TestReadMsgID_MessageIDKey(t *testing.T) {
	id, ok := readMsgID([]byte(`{"MessageID":"xyz"}`))
	if !ok || id != "xyz" {
		t.Errorf("got %s %v", id, ok)
	}
}

func TestReadMsgID_MessageIDLower(t *testing.T) {
	id, ok := readMsgID([]byte(`{"message_id":"lmn"}`))
	if !ok || id != "lmn" {
		t.Errorf("got %s %v", id, ok)
	}
}

func TestReadMsgID_IDKey(t *testing.T) {
	id, ok := readMsgID([]byte(`{"id":"qrs"}`))
	if !ok || id != "qrs" {
		t.Errorf("got %s %v", id, ok)
	}
}

func TestReadMsgID_NotFound(t *testing.T) {
	_, ok := readMsgID([]byte(`{"other":"x"}`))
	if ok {
		t.Errorf("expected not found")
	}
}

func TestReadMsgID_InvalidJSON(t *testing.T) {
	_, ok := readMsgID([]byte(`not json`))
	if ok {
		t.Errorf("expected not found")
	}
}

func TestReadMsgID_EmptyString(t *testing.T) {
	_, ok := readMsgID([]byte(`{"msg_id":""}`))
	if ok {
		t.Errorf("expected not found for empty msg_id")
	}
}

func TestGuessEventType_EventType(t *testing.T) {
	if guessEventType([]byte(`{"event_type":"delivered"}`)) != "delivered" {
		t.Errorf("got %s", guessEventType([]byte(`{"event_type":"delivered"}`)))
	}
}

func TestGuessEventType_Event(t *testing.T) {
	if guessEventType([]byte(`{"event":"open"}`)) != "open" {
		t.Errorf("got %s", guessEventType([]byte(`{"event":"open"}`)))
	}
}

func TestGuessEventType_Type(t *testing.T) {
	if guessEventType([]byte(`{"type":"click"}`)) != "click" {
		t.Errorf("got %s", guessEventType([]byte(`{"type":"click"}`)))
	}
}

func TestGuessEventType_NotificationType(t *testing.T) {
	if guessEventType([]byte(`{"notificationType":"bounce"}`)) != "bounce" {
		t.Errorf("got %s", guessEventType([]byte(`{"notificationType":"bounce"}`)))
	}
}

func TestGuessEventType_NotFound(t *testing.T) {
	if guessEventType([]byte(`{"foo":"bar"}`)) != "" {
		t.Errorf("got %s", guessEventType([]byte(`{"foo":"bar"}`)))
	}
}

func TestGuessEventType_InvalidJSON(t *testing.T) {
	if guessEventType([]byte(`not json`)) != "" {
		t.Errorf("expected empty")
	}
}

func TestReadAll_EOF(t *testing.T) {
	r := strings.NewReader("hello")
	got, err := readAll(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("got %s", string(got))
	}
}

func TestReadAll_Empty(t *testing.T) {
	r := strings.NewReader("")
	got, err := readAll(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d bytes", len(got))
	}
}

func TestReadAll_LargeData(t *testing.T) {
	data := strings.Repeat("a", 5000)
	r := strings.NewReader(data)
	got, err := readAll(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(got) != data {
		t.Errorf("lengths: got %d want %d", len(got), len(data))
	}
}

func TestGetPagination_Defaults(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	page, perPage, offset := getPagination(c)
	if page != 1 {
		t.Errorf("page %d", page)
	}
	if perPage != 25 {
		t.Errorf("perPage %d", perPage)
	}
	if offset != 0 {
		t.Errorf("offset %d", offset)
	}
}

func TestGetPagination_Custom(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=3&per_page=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	page, perPage, offset := getPagination(c)
	if page != 3 {
		t.Errorf("page %d", page)
	}
	if perPage != 10 {
		t.Errorf("perPage %d", perPage)
	}
	if offset != 20 {
		t.Errorf("offset %d", offset)
	}
}

func TestGetPagination_PerPageTooLarge(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=500", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_, perPage, _ := getPagination(c)
	if perPage != 25 {
		t.Errorf("expected fallback to 25, got %d", perPage)
	}
}

func TestGetPagination_PerPageZero(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_, perPage, _ := getPagination(c)
	if perPage != 25 {
		t.Errorf("expected fallback to 25, got %d", perPage)
	}
}

func TestGetPagination_InvalidValues(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=abc&per_page=xyz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	page, perPage, _ := getPagination(c)
	if page != 1 {
		t.Errorf("expected 1, got %d", page)
	}
	if perPage != 25 {
		t.Errorf("expected 25, got %d", perPage)
	}
}

func TestTenantFromCtx_Empty(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	t1, u1, a1 := s.tenantFromCtx(c)
	if t1 != "" || u1 != "" || a1 != false {
		t.Errorf("got %s %s %v", t1, u1, a1)
	}
}

func TestTenantFromCtx_Populated(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tid")
	c.Set("user_id", "uid")
	c.Set("is_admin", true)
	t1, u1, a1 := s.tenantFromCtx(c)
	if t1 != "tid" || u1 != "uid" || a1 != true {
		t.Errorf("got %s %s %v", t1, u1, a1)
	}
}

func TestServer_JSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.json(c, http.StatusOK, map[string]any{"ok": true}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if rec.Code != 200 {
		t.Errorf("got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("got %s", rec.Body.String())
	}
}

func TestServer_ErrorResp_WithErr(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.errorResp(c, 400, "msg", errors.New("boom")); err != nil {
		t.Fatalf("err: %v", err)
	}
	if rec.Code != 400 {
		t.Errorf("got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("got %s", rec.Body.String())
	}
}

func TestServer_ErrorResp_NoErr(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.errorResp(c, 500, "msg", nil); err != nil {
		t.Fatalf("err: %v", err)
	}
	if rec.Code != 500 {
		t.Errorf("got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "details") {
		t.Errorf("expected no details field for nil err")
	}
}

func TestRenderMarkdown_PlainText(t *testing.T) {
	got := renderMarkdown("plain text")
	if got != "plain text" {
		t.Errorf("got %q", got)
	}
}

func TestRenderMarkdown_Empty(t *testing.T) {
	got := renderMarkdown("")
	if got != "" {
		t.Errorf("got %q", got)
	}
}

func TestRenderMarkdown_HTML(t *testing.T) {
	got := renderMarkdown("# Hello")
	if !strings.Contains(got, "Hello") {
		t.Errorf("got %q", got)
	}
}

func TestRenderMarkdown_Template(t *testing.T) {
	got := renderMarkdown("Hello {{.Name}}")
	// RenderMarkdown doesn't supply data; engine substitutes with <no value>
	if !strings.Contains(got, "Hello") {
		t.Errorf("got %q", got)
	}
}

// io.Reader for testing
var _ io.Reader = (*strings.Reader)(nil)
