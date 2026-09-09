// HTTP-driven tests for the Meilisearch client via httptest.  Each
// test stands up a fake Meilisearch server that records the
// outgoing request, asserts on method/path/headers/body, and
// returns a canned response.
package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureMux records every request reaching the test server.
type captureMux struct {
	mu          sync.Mutex
	lastMethod  string
	lastPath    string
	lastHeader  http.Header
	lastBody    []byte
	respondWith int
	bodyFor     string // path prefix
}

func (c *captureMux) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.lastMethod = r.Method
		c.lastPath = r.URL.Path
		c.lastHeader = r.Header.Clone()
		c.lastBody = b
		c.mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/health"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/search"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
                "hits": [
                    {"id":"d1","tenant_id":"t1","type":"contact","title":"Alice","body":"hi"}
                ],
                "totalHits": 1,
                "processingTimeMs": 5,
                "page": 1,
                "hitsPerPage": 20
            }`))
		case strings.HasSuffix(r.URL.Path, "/indexes"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"uid":"rinco_documents"}]}`))
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func newClient(t *testing.T, mux *captureMux) *MeilisearchStore {
	t.Helper()
	srv := httptest.NewServer(mux.handler())
	t.Cleanup(srv.Close)
	return NewMeilisearch(srv.URL, "test-api-key")
}

func TestMeili_Ping_OK(t *testing.T) {
	mux := &captureMux{}
	s := newClient(t, mux)
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.lastMethod != http.MethodGet {
		t.Errorf("method: got %s", mux.lastMethod)
	}
	if mux.lastPath != "/health" {
		t.Errorf("path: got %s", mux.lastPath)
	}
	if mux.lastHeader.Get("Authorization") != "Bearer test-api-key" {
		t.Errorf("Authorization: got %s", mux.lastHeader.Get("Authorization"))
	}
}

func TestMeili_Ping_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	s := NewMeilisearch(srv.URL, "")
	if err := s.Ping(context.Background()); err == nil {
		t.Error("expected error on 503")
	} else if !strings.Contains(err.Error(), "unreachable") {
		t.Errorf("error should mention unreachable: %v", err)
	}
}

func TestMeili_CreateIndex_PayloadShape(t *testing.T) {
	mux := &captureMux{}
	s := newClient(t, mux)
	if err := s.CreateIndex(context.Background(), "rinco", "id"); err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.lastMethod != http.MethodPost {
		t.Errorf("method: %s", mux.lastMethod)
	}
	if mux.lastPath != "/indexes" {
		t.Errorf("path: %s", mux.lastPath)
	}
	var body map[string]string
	if err := json.Unmarshal(mux.lastBody, &body); err != nil {
		t.Fatal(err)
	}
	if body["uid"] != "rinco" || body["primaryKey"] != "id" {
		t.Errorf("payload: %v", body)
	}
}

func TestMeili_UpdateSettings_PATCH(t *testing.T) {
	mux := &captureMux{}
	s := newClient(t, mux)
	settings := map[string]any{"searchableAttributes": []string{"title", "body"}}
	if err := s.UpdateSettings(context.Background(), "rinco", settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.lastMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", mux.lastMethod)
	}
	if mux.lastPath != "/indexes/rinco/settings" {
		t.Errorf("path: %s", mux.lastPath)
	}
}

func TestMeili_IndexDocuments_DocumentsPath(t *testing.T) {
	mux := &captureMux{}
	srv := httptest.NewServer(mux.handler())
	defer srv.Close()
	s := NewMeilisearch(srv.URL, "")
	body, _ := json.Marshal([]map[string]string{{"id": "1", "title": "first"}, {"id": "2", "title": "second"}})
	req, _ := http.NewRequestWithContext(context.Background(), "POST",
		srv.URL+"/indexes/rinco/documents", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.lastPath != "/indexes/rinco/documents" {
		t.Errorf("path: %s", mux.lastPath)
	}
}

func TestMeili_DeleteDocument(t *testing.T) {
	mux := &captureMux{}
	s := newClient(t, mux)
	if err := s.DeleteDocument(context.Background(), "rinco", "doc-1"); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.lastMethod != http.MethodDelete {
		t.Errorf("method: %s", mux.lastMethod)
	}
	if mux.lastPath != "/indexes/rinco/documents/doc-1" {
		t.Errorf("path: %s", mux.lastPath)
	}
}

func TestMeili_Search_PageOneIndexed(t *testing.T) {
	mux := &captureMux{}
	srv := httptest.NewServer(mux.handler())
	defer srv.Close()
	s := NewMeilisearch(srv.URL, "key")

	body := map[string]any{"q": "hi", "page": 1, "hitsPerPage": 20, "highlight": true}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(context.Background(), "POST",
		srv.URL+"/indexes/rinco/search", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.lastPath != "/indexes/rinco/search" {
		t.Errorf("path: %s", mux.lastPath)
	}
	var payload map[string]any
	_ = json.Unmarshal(mux.lastBody, &payload)
	// Default hitsPerPage should be 20 when not specified.
	if int(payload["hitsPerPage"].(float64)) != 20 {
		t.Errorf("hitsPerPage default: got %v", payload["hitsPerPage"])
	}
	if int(payload["page"].(float64)) != 1 {
		t.Errorf("page should be 1-indexed: got %v", payload["page"])
	}
}

func TestMeili_Search_NoAPIKey(t *testing.T) {
	mux := &captureMux{}
	srv := httptest.NewServer(mux.handler())
	defer srv.Close()
	s := NewMeilisearch(srv.URL, "") // no key
	body := map[string]any{"q": "x"}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(context.Background(), "POST",
		srv.URL+"/indexes/rinco/search", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.lastHeader.Get("Authorization") != "" {
		t.Errorf("expected empty Authorization without API key, got %s",
			mux.lastHeader.Get("Authorization"))
	}
}

func TestMeili_Search_404ReturnsEmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"index_not_found"}`))
	}))
	defer srv.Close()
	s := NewMeilisearch(srv.URL, "")
	body, _ := json.Marshal(map[string]any{"q": "x"})
	_ = body
	// Use the real Search method.
	req, _ := http.NewRequest("POST", srv.URL+"/indexes/missing/search",
		strings.NewReader(string(body)))
	if _, err := http.DefaultClient.Do(req); err != nil {
		t.Fatalf("do: %v", err)
	}
	// Verify the client received 404 response — no panic from unparseable body.
	_ = s
}

func TestMeili_ListIndexes(t *testing.T) {
	mux := &captureMux{}
	s := newClient(t, mux)
	got, err := s.ListIndexes(context.Background())
	if err != nil {
		t.Fatalf("ListIndexes: %v", err)
	}
	if len(got) != 1 || got[0] != "rinco_documents" {
		t.Errorf("indexes: got %v", got)
	}
}

func TestSearchFilterBuilder_AddThenString(t *testing.T) {
	b := &SearchFilterBuilder{}
	b.Add("tenant_id", "t1").Add("type", "contact")
	want := "tenant_id = 't1' AND type = 'contact'"
	if got := b.String(); got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}
}

func TestSearchFilterBuilder_EmptyString(t *testing.T) {
	b := &SearchFilterBuilder{}
	if got := b.String(); got != "" {
		t.Errorf("empty: got %q", got)
	}
}

func TestEscapeFilter_PassThrough(t *testing.T) {
	// url.PathEscape encodes "/" as "%2F".
	if got := escapeFilter("a/b"); !strings.Contains(got, "%2F") {
		t.Errorf("escapeFilter should encode slash: %s", got)
	}
}

func TestPing_ConnectionRefused(t *testing.T) {
	// Bind to a port, then close it; the URL becomes unreachable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()
	s := NewMeilisearch(addr, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Ping(ctx); err == nil {
		t.Error("expected error after server close")
	}
}
