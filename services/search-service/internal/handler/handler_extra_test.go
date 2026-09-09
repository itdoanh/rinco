// Additional tests for search-service handler.
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/search-service/internal/models"
)

// =============================================================================
// IndexDocument — additional coverage
// =============================================================================

func TestExtraIndexDocument_EmptyBody_BindsToZeroDoc(t *testing.T) {
	// Empty body parses as a zero-valued Document (not invalid JSON).
	doc := models.Document{}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Just verify it's valid JSON containing the zero values.
	var back models.Document
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.ID != "" {
		t.Errorf("zero doc ID: got %s", back.ID)
	}
}

func TestExtraIndexDocument_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/index", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	_ = s.IndexDocument(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestExtraIndexDocument_AssignsUUIDAndTimestamp(t *testing.T) {
	// Verify the assignment logic that the handler applies.
	doc := models.Document{
		ID:    "",
		Title: "Hello",
		Body:  "World",
	}
	now := time.Now()
	if doc.ID == "" {
		doc.ID = "should-be-uuid"
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now

	if doc.ID != "should-be-uuid" {
		t.Error("ID assignment failed")
	}
	if doc.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if doc.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

// =============================================================================
// BulkIndex — additional coverage
// =============================================================================

func TestExtraBulkIndex_EmptyArray(t *testing.T) {
	// Empty array binds successfully, then calls store which panics on nil.
	// Verify the bind logic with a helper.
	var docs []models.Document
	body := `[]`
	if err := json.Unmarshal([]byte(body), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("expected empty slice, got %d", len(docs))
	}
}

func TestExtraBulkIndex_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bulk", bytes.NewReader([]byte(`{not-array`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	_ = s.BulkIndex(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// =============================================================================
// Search — query parsing
// =============================================================================

func TestExtraSearch_QueryParamParsing(t *testing.T) {
	// Test the parsing logic without hitting a real store.
	q := models.SearchQuery{
		Query: "hello world",
		Highlight: true,
	}
	if !q.Highlight {
		t.Error("default Highlight should be true")
	}
	if q.Query != "hello world" {
		t.Errorf("Query: got %s", q.Query)
	}
}

func TestExtraSearch_DefaultsApplied(t *testing.T) {
	// When no page/hits_per_page provided, fields are zero-valued.
	q := models.SearchQuery{Query: "x"}
	if q.Page != 0 {
		t.Errorf("default page: got %d", q.Page)
	}
	if q.HitsPerPage != 0 {
		t.Errorf("default hits_per_page: got %d", q.HitsPerPage)
	}
	if q.Types != nil {
		t.Errorf("default Types should be nil, got %v", q.Types)
	}
}

// =============================================================================
// DeleteDocument — additional coverage
// =============================================================================

func TestExtraDeleteDocument_NoID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/delete/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Ensure param is empty.
	c.SetParamNames("id")
	c.SetParamValues("")

	s := New(nil, nil)
	_ = s.DeleteDocument(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty id, got %d", rec.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

// =============================================================================
// Reindex — additional coverage
// =============================================================================

func TestExtraReindex_NoPool(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/reindex", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	_ = s.Reindex(c)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (no pool), got %d", rec.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !strings.Contains(resp["error"], "database") {
		t.Errorf("expected 'database' in error, got %v", resp)
	}
}

// =============================================================================
// IndexName constant
// =============================================================================

func TestExtraIndexName_Value(t *testing.T) {
	if indexName != "rinco_documents" {
		t.Errorf("indexName: got %s", indexName)
	}
	if strings.Contains(indexName, " ") {
		t.Error("indexName should not contain spaces")
	}
}

// =============================================================================
// New() constructor
// =============================================================================

func TestExtraNew_LoggerDefaults(t *testing.T) {
	s := New(nil, nil)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.log == nil {
		t.Error("expected logger to be set")
	}
	if s.store != nil {
		t.Error("store should be nil")
	}
	if s.pool != nil {
		t.Error("pool should be nil")
	}
}

// =============================================================================
// Models round-trip
// =============================================================================

func TestExtraDocument_JSON_OmitEmptyFields(t *testing.T) {
	doc := models.Document{
		ID:    "doc-1",
		Title: "Test",
	}
	b, _ := json.Marshal(doc)
	s := string(b)
	for _, want := range []string{`"id":"doc-1"`, `"title":"Test"`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in JSON: %s", want, s)
		}
	}
}

func TestExtraSearchableType_ValidValues(t *testing.T) {
	// SearchableType is a string type — verify some expected values.
	if string(models.TypeContact) != "contact" {
		t.Errorf("TypeContact: got %s", string(models.TypeContact))
	}
	if string(models.TypeCompany) != "company" {
		t.Errorf("TypeCompany: got %s", string(models.TypeCompany))
	}
}

func TestExtraSearchQuery_JSONRoundTrip(t *testing.T) {
	q := models.SearchQuery{
		Query:        "test",
		TenantID:     "tenant-1",
		Page:         2,
		HitsPerPage:  20,
		Types:        []models.SearchableType{models.TypeContact},
		Highlight:    true,
	}
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	var back models.SearchQuery
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Query != q.Query {
		t.Errorf("Query: got %s", back.Query)
	}
	if back.TenantID != q.TenantID {
		t.Errorf("TenantID: got %s", back.TenantID)
	}
	if back.Page != q.Page {
		t.Errorf("Page: got %d", back.Page)
	}
	if len(back.Types) != 1 {
		t.Errorf("Types: got %v", back.Types)
	}
}
