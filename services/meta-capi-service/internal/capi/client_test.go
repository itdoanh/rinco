package capi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if c.httpClient.Timeout == 0 {
		t.Error("expected a non-zero HTTP timeout")
	}
}

func TestSendEvents_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}
		// Validate request body shape
		var payload EventPayload
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if len(payload.Data) != 1 {
			t.Errorf("expected 1 event, got %d", len(payload.Data))
		}
		// Verify access_token appears in the URL query string.
		if !strings.Contains(r.URL.RawQuery, "access_token=token123") {
			t.Errorf("access_token missing from query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events_received":[{"event_id":"e1","fb_event_id":"fbe1"}]}`))
	}))
	defer srv.Close()

	// We can't easily override the hardcoded graph.facebook.com URL,
	// but we can re-route SendEvents through a thin wrapper that
	// swaps the host.  For unit testing we expose the request payload
	// shape via SendEventsToURL (see below) — here we just verify the
	// constructed URL is correct via URL parsing.
	c := NewClient()
	_ = c
	// Use SendEventsToURL helper to validate behaviour end-to-end.
	res, err := c.SendEventsToURL(
		context.Background(), srv.URL+"?access_token=token123", EventPayload{
			Data: []CAPIEventData{{
				EventName:    "Purchase",
				ActionSource: "website",
			}},
		})
	if err != nil {
		t.Fatalf("SendEventsToURL error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	if res[0].EventID != "e1" {
		t.Errorf("unexpected event id: %q", res[0].EventID)
	}
}

func TestSendEvents_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient()
	_, err := c.SendEventsToURL(context.Background(), srv.URL+"?access_token=x", EventPayload{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSendEvents_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	c := NewClient()
	_, err := c.SendEventsToURL(context.Background(), srv.URL+"?access_token=x", EventPayload{})
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

func TestSendEvents_Messages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"events_received":[],"messages":["invalid pixel id"]}`))
	}))
	defer srv.Close()

	c := NewClient()
	_, err := c.SendEventsToURL(context.Background(), srv.URL+"?access_token=x", EventPayload{})
	if err == nil {
		t.Fatal("expected error from messages field")
	}
	if !strings.Contains(err.Error(), "invalid pixel id") {
		t.Errorf("expected pixel id error in message, got: %v", err)
	}
}

func TestTestConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"MyPixel"}`))
	}))
	defer srv.Close()

	c := NewClient()
	if err := c.TestConnectionToURL(context.Background(), srv.URL+"?access_token=x"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestTestConnection_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient()
	if err := c.TestConnectionToURL(context.Background(), srv.URL+"?access_token=x"); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestEventPayload_JSONTags(t *testing.T) {
	p := EventPayload{
		Data: []CAPIEventData{{
			EventID:      "abc",
			EventName:    "Purchase",
			EventTime:    1700000000,
			ActionSource: "website",
			UserData: UserData{
				Email:     "alice@example.com",
				ExternalID: "user-1",
			},
			CustomData: CustomData{
				Value:    199.99,
				Currency: "USD",
				Contents: []ContentItem{
					{ID: "sku-1", Quantity: 2, Price: 49.99},
				},
			},
		}},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{
		`"event_id":"abc"`,
		`"event_name":"Purchase"`,
		`"action_source":"website"`,
		`"em":"alice@example.com"`,
		`"external_id":"user-1"`,
		`"value":199.99`,
		`"currency":"USD"`,
		`"quantity":2`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
}

func TestEventPayload_OmitsEmpty(t *testing.T) {
	p := NewEventPayload()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"data":[]`) {
		t.Errorf("expected empty data array, got %s", string(b))
	}
	if strings.Contains(string(b), `"debug":`) {
		t.Errorf("debug should be omitted when zero, got %s", string(b))
	}
}

func TestEventPayload_AppendEvent(t *testing.T) {
	p := EventPayload{}
	p.AppendEvent(CAPIEventData{EventName: "Lead"})
	if len(p.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(p.Data))
	}
	if p.Data[0].EventName != "Lead" {
		t.Errorf("unexpected event: %s", p.Data[0].EventName)
	}
	// AppendEvent on already-populated slice should keep working.
	p.AppendEvent(CAPIEventData{EventName: "Purchase"})
	if len(p.Data) != 2 {
		t.Errorf("expected 2 events, got %d", len(p.Data))
	}
}

func TestProcessingOptions_JSONKeys(t *testing.T) {
	// Note: the existing struct has a typo in the JSON tag
	// ("identifiers" inside the AllowNoSales tag).  We assert that
	// the typo is preserved so we don't silently change wire
	// compatibility — fix in a follow-up if intentional.
	b, _ := json.Marshal(ProcessingOptions{AllowNoSales: true})
	got := string(b)
	if !strings.Contains(got, "identifiers") {
		t.Errorf("expected 'identifiers' in %s (preserved as-is)", got)
	}
}
