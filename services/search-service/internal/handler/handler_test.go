// Tests for search-service handler.
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itdoanh/rinco/services/search-service/internal/models"
	"github.com/labstack/echo/v4"
)

func TestNew(t *testing.T) {
	s := New(nil, nil)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.store != nil {
		t.Error("expected store to be nil")
	}
}

func TestIndexName_Constant(t *testing.T) {
	if indexName != "rinco_documents" {
		t.Errorf("indexName: got %s", indexName)
	}
}

func TestIndexDocument_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/index", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.IndexDocument(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestBulkIndex_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bulk", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.BulkIndex(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSearch_Defaults(t *testing.T) {
	t.Skip("Search requires non-nil store; covered in integration tests")
}

func TestDeleteDocument_EmptyID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/delete/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.DeleteDocument(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Empty ID should be caught by store call before panic.
	// The handler returns 400 for empty id; otherwise 500 from nil store.
	// Either way, the handler should not panic.
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 400 or 500, got %d", rec.Code)
	}
}

func TestReindex_NoPool(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/reindex", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.Reindex(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (no pool), got %d", rec.Code)
	}
}

func TestDocument_JSON(t *testing.T) {
	doc := models.Document{
		ID:    "doc-1",
		Title: "Test",
		Body:  "Body content",
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var back models.Document
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.ID != doc.ID {
		t.Errorf("ID: got %s", back.ID)
	}
	if back.Title != doc.Title {
		t.Errorf("Title: got %s", back.Title)
	}
}
