// Extra tests for lead-service handler helper functions and structs.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtra_JSON_OK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, "")
	if err := s.json(c, http.StatusOK, map[string]string{"k": "v"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	if !contains(rec.Body.String(), `"k":"v"`) {
		t.Errorf("body: got %s", rec.Body.String())
	}
}

func TestExtra_ErrorResp_WithError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, "")
	if err := s.errorResp(c, http.StatusBadRequest, "bad input", errors.New("oops")); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rec.Code)
	}
	if !contains(rec.Body.String(), "bad input") {
		t.Errorf("expected message in body")
	}
	if !contains(rec.Body.String(), "oops") {
		t.Errorf("expected error details")
	}
}

func TestExtra_ErrorResp_NoError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, "")
	if err := s.errorResp(c, http.StatusUnauthorized, "unauth", nil); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d", rec.Code)
	}
	if !contains(rec.Body.String(), "unauth") {
		t.Errorf("expected msg")
	}
}

func TestExtra_ListResp_Marshal(t *testing.T) {
	r := listResp{
		Data:       []string{"a", "b"},
		Total:      2,
		Page:       1,
		PerPage:    20,
		TotalPages: 1,
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !contains(s, `"total":2`) {
		t.Errorf("total field missing: %s", s)
	}
	if !contains(s, `"per_page":20`) {
		t.Errorf("per_page field missing")
	}
}

func TestExtra_ErrResp_Marshal(t *testing.T) {
	r := errResp{Error: "boom", Details: "explosion"}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !contains(s, `"error":"boom"`) {
		t.Errorf("error missing")
	}
	if !contains(s, `"details":"explosion"`) {
		t.Errorf("details missing")
	}
}

func TestExtra_AssignLeadReq_Fields(t *testing.T) {
	// Test that the request struct JSON-roundtrips
	b, err := json.Marshal(assignLeadReq{Reason: "r"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(b), `"reason":"r"`) {
		t.Errorf("reason missing: %s", string(b))
	}
}

func TestExtra_UpdateStatusReq_Fields(t *testing.T) {
	r := updateStatusReq{Status: "won"}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(b), `"status":"won"`) {
		t.Errorf("status missing")
	}
}

func TestExtra_LeadNoteReq_OmitsEmpty(t *testing.T) {
	b, err := json.Marshal(leadNoteReq{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if contains(s, "body") {
		t.Errorf("body should be omitted: %s", s)
	}
	if contains(s, "is_pinned") {
		t.Errorf("is_pinned should be omitted: %s", s)
	}
}

func TestExtra_NewServer_WithArgs(t *testing.T) {
	rdb := &RedisClient{Addr: "x", DB: 1}
	s := NewServer(nil, rdb, nil, "http://scoring")
	if s.rdb != rdb {
		t.Error("rdb not stored")
	}
	if s.scoringURL != "http://scoring" {
		t.Errorf("scoringURL: got %s", s.scoringURL)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
