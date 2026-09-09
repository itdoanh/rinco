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
	s := &Server{queue: make(chan capiEvent, 4)}
	if err := s.enqueueCAPI(capiEvent{EventID: "e1", EventName: "Lead"}); err != nil {
		t.Fatalf("enqueueCAPI: %v", err)
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
	s := &Server{queue: make(chan capiEvent, 1)}
	// First one fills the slot.
	if err := s.enqueueCAPI(capiEvent{EventID: "a"}); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	// Second one should fail because nobody is draining the channel.
	if err := s.enqueueCAPI(capiEvent{EventID: "b"}); err == nil {
		t.Fatal("expected queue-full error")
	}
}

// TestEnqueueCAPI_NilQueue returns a clear error instead of panicking.
func TestEnqueueCAPI_NilQueue(t *testing.T) {
	s := &Server{}
	if err := s.enqueueCAPI(capiEvent{EventID: "x"}); err == nil {
		t.Fatal("expected error when queue is nil")
	}
}

// TestEnqueueCAPI_ClosedDoesNotPanic ensures Close() followed by
// enqueueCAPI doesn't crash the handler goroutine.
func TestEnqueueCAPI_ClosedDoesNotPanic(t *testing.T) {
	s := New(nil, nil, nil, "", "", "")
	// Drain the dispatch goroutine by closing.
	s.Close()
	// enqueueCAPI should swallow the panic from the closed channel.
	err := s.enqueueCAPI(capiEvent{EventID: "x"})
	// The send may either land before close (returning nil) or trigger
	// the recovered panic (returning the recovered error message).
	if err != nil && !strings.Contains(err.Error(), "queue") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCAPISend_EnqueuesEvent exercises CAPISend and confirms the event
// arrives on the channel with the supplied fields.
func TestCAPISend_EnqueuesEvent(t *testing.T) {
	s := &Server{queue: make(chan capiEvent, 4)}
	e := echo.New()
	body := `{"EventName":"Lead","EventID":"abc","ActionSource":"website","TenantID":"t1","PixelID":"px1"}`
	req := httptest.NewRequest(http.MethodPost, "/capi/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := s.CAPISend(c); err != nil {
		t.Fatalf("CAPISend: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	select {
	case got := <-s.queue:
		if got.EventID != "abc" || got.EventName != "Lead" || got.PixelID != "px1" {
			t.Errorf("unexpected event: %+v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event not queued")
	}
}

// TestCAPISend_BadJSON returns 400.
func TestCAPISend_BadJSON(t *testing.T) {
	s := &Server{queue: make(chan capiEvent, 4)}
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
	s := &Server{queue: make(chan capiEvent, 4)}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/capi/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := s.CAPITest(c); err != nil {
		t.Fatalf("CAPITest: %v", err)
	}
	select {
	case got := <-s.queue:
		if got.EventName != "TestEvent" {
			t.Errorf("EventName: %s", got.EventName)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event not queued")
	}
}

// TestCAPITest_QueueFullReturns503 ensures we don't block when the
// queue is full.
func TestCAPITest_QueueFullReturns503(t *testing.T) {
	s := &Server{queue: make(chan capiEvent, 1)}
	// Pre-fill the only slot.
	s.queue <- capiEvent{EventID: "filler"}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/capi/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = s.CAPITest(c)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when queue is full, got %d", rec.Code)
	}
}

// TestSendCAPI_MissingPixel returns an error when neither per-event
// nor server-level pixel is set.
func TestSendCAPI_MissingPixel(t *testing.T) {
	s := &Server{}
	if err := s.sendCAPI(capiEvent{EventID: "x"}); err == nil {
		t.Fatal("expected error for missing pixel id")
	}
}

// TestSendCAPI_PassesAccessTokenInURL ensures the appSecret is sent as
// the access_token URL query parameter (not in the body) per FB docs.
func TestSendCAPI_PassesAccessTokenInURL(t *testing.T) {
	h := &captureCAPIHandler{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	// Temporarily redirect sendCAPI to the test server by patching the
	// endpoint construction.  We do this by exposing a sendCAPIAtURL
	// helper below.
	s := &Server{appSecret: "supersecret-token"}
	err := s.sendCAPIAt(capiEvent{EventID: "evt-1", EventName: "Lead", PixelID: "px-1", ActionSource: "website"}, srv.URL+"/events")
	if err != nil {
		t.Fatalf("sendCAPIAt: %v", err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.calls != 1 {
		t.Fatalf("expected 1 call, got %d", h.calls)
	}
	if got := h.queries["access_token"]; len(got) == 0 || got[0] != "supersecret-token" {
		t.Errorf("access_token query: got %v", h.queries["access_token"])
	}
	// Body must NOT contain the token (it lives in the URL only).
	if body, _ := json.Marshal(h.payload); strings.Contains(string(body), "supersecret-token") {
		t.Errorf("token leaked into body: %s", body)
	}
}

// TestSendCAPI_PropagatesHTTPError ensures we return an error when FB
// returns 4xx/5xx.
func TestSendCAPI_PropagatesHTTPError(t *testing.T) {
	h := &captureCAPIHandler{failNext: true}
	srv := httptest.NewServer(h)
	defer srv.Close()
	s := &Server{}
	err := s.sendCAPIAt(capiEvent{EventID: "evt-1", EventName: "Lead", PixelID: "px"}, srv.URL+"/events")
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}

// TestDispatchCAPI_DrainsQueue verifies the dispatchCAPI loop reads
// queued events and forwards them to sendCAPIAt.
func TestDispatchCAPI_DrainsQueue(t *testing.T) {
	h := &captureCAPIHandler{}
	srv := httptest.NewServer(h)
	defer srv.Close()
	s := &Server{
		queue:        make(chan capiEvent, 2),
		dispatchURL:  srv.URL + "/events",
	}
	s.queue <- capiEvent{EventID: "a", EventName: "Lead", PixelID: "px"}
	s.queue <- capiEvent{EventID: "b", EventName: "Purchase", PixelID: "px"}

	done := make(chan struct{})
	go func() {
		s.dispatchCAPIAt()
		close(done)
	}()
	// Wait for the goroutine to drain both events.
	deadline := time.After(500 * time.Millisecond)
	for {
		h.mu.Lock()
		n := h.calls
		h.mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected 2 calls, got %d", n)
		case <-time.After(10 * time.Millisecond):
		}
	}
	close(s.queue)
	<-done
}
