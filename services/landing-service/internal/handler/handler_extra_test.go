// Extra tests for landing-service handler enqueue and CAPI dispatch.
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// captureCAPIHandler is an httptest handler that mimics the FB CAPI
// shape and records the last received payload for assertion.
type captureCAPIHandler struct {
	mu       sync.Mutex
	payload  map[string]interface{}
	queries  map[string][]string
	calls    int
	failNext bool
}

func (h *captureCAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls++
	h.queries = r.URL.Query()
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &h.payload)
	if h.failNext {
		h.failNext = false
		http.Error(w, "boom", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"events_received":[{"event_id":"ok"}]}`))
}

// TestEnqueueCAPI_Success verifies that an event reaches the channel.
func TestEnqueueCAPI_Success(t *testing.T) {
	q := make(chan capiEvent, 4)
	s := &Server{queue: q, capiQueue: q}
	ok := s.enqueueCAPI(capiEvent{EventID: "e1", EventName: "Lead"})
	if !ok {
		t.Fatalf("enqueueCAPI failed")
	}
	select {
	case got := <-s.queue:
		if got.EventID != "e1" || got.EventName != "Lead" {
			t.Errorf("unexpected event: %+v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected event in queue")
	}
}

// TestEnqueueCAPI_FullQueue verifies that the non-blocking send returns
// an error when the queue is full rather than blocking forever.
func TestEnqueueCAPI_FullQueue(t *testing.T) {
	q := make(chan capiEvent, 1)
	s := &Server{queue: q, capiQueue: q}
	// First one fills the slot.
	if ok := s.enqueueCAPI(capiEvent{EventID: "a"}); !ok {
		t.Fatalf("first enqueue failed")
	}
	// Second one should fail because nobody is draining the channel.
	if ok := s.enqueueCAPI(capiEvent{EventID: "b"}); ok {
		t.Fatal("expected queue-full error")
	}
}

// TestCAPISend_EnqueuesEvent exercises CAPISend and confirms the event
// arrives on the channel with the supplied fields.
func TestCAPISend_EnqueuesEvent(t *testing.T) {
	q := make(chan capiEvent, 4)
	s := &Server{queue: q, capiQueue: q}
	e := echo.New()
	body := `{"event_name":"Lead","event_id":"abc","url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/capi/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "t1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := s.CAPISend(c); err != nil {
		t.Fatalf("CAPISend: %v", err)
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
	select {
	case got := <-q:
		if got.EventID != "abc" || got.EventName != "Lead" {
			t.Errorf("unexpected event: %+v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event not queued")
	}
}

// TestCAPISend_BadJSON returns 400.
func TestCAPISend_BadJSON(t *testing.T) {
	q := make(chan capiEvent, 4)
	s := &Server{queue: q, capiQueue: q}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/capi/send", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = s.CAPISend(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestCAPITest_EnqueuesTestEvent confirms CAPITest emits a TestEvent.
func TestCAPITest_EnqueuesTestEvent(t *testing.T) {
	q := make(chan capiEvent, 4)
	s := &Server{queue: q, capiQueue: q}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/capi/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := s.CAPITest(c); err != nil {
		t.Fatalf("CAPITest: %v", err)
	}
	select {
	case got := <-q:
		if got.EventName != "TestEvent" {
			t.Errorf("EventName: %s", got.EventName)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event not queued")
	}
}

// TestNewServer_Legacy verifies backward compatibility with NewServer.
func TestNewServer_Legacy(t *testing.T) {
	s := NewServer([]byte("test-key"))
	if s == nil {
		t.Fatal("expected server")
	}
	if s.hmacKey != "test-key" {
		t.Errorf("hmacKey not set: %s", s.hmacKey)
	}
}

// TestNew_Full verifies the full constructor works.
func TestNew_Full(t *testing.T) {
	s := New(nil, nil, nil, "key1", "salt1")
	if s == nil {
		t.Fatal("expected server")
	}
	if s.hmacKey != "key1" {
		t.Errorf("hmacKey not set: %s", s.hmacKey)
	}
	if s.trackingSalt != "salt1" {
		t.Errorf("trackingSalt not set: %s", s.trackingSalt)
	}
}

// TestCAPIStatus verifies CAPIStatus returns tenant info.
func TestCAPIStatus(t *testing.T) {
	q := make(chan capiEvent, 4)
	s := &Server{queue: q, capiQueue: q}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/capi/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = s.CAPIStatus(c)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// TestHealth verifies Health endpoint.
func TestHealth(t *testing.T) {
	s := &Server{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = s.Health(c)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
