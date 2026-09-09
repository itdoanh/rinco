// Extra tests for meta-capi service CAPI client helpers.
package capi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExtraNewEventPayload_Empty(t *testing.T) {
	p := NewEventPayload()
	if p.Data == nil {
		t.Error("Data should be empty slice, not nil")
	}
	if len(p.Data) != 0 {
		t.Errorf("Data length: got %d", len(p.Data))
	}
}

func TestExtraEventPayload_AppendEvent(t *testing.T) {
	p := EventPayload{}
	p.AppendEvent(CAPIEventData{EventName: "PageView"})
	p.AppendEvent(CAPIEventData{EventName: "Purchase"})
	if len(p.Data) != 2 {
		t.Errorf("Data length: %d", len(p.Data))
	}
}

func TestExtraEventPayload_AppendEvent_AllocatesIfNil(t *testing.T) {
	var p EventPayload
	p.AppendEvent(CAPIEventData{EventName: "x"})
	if p.Data == nil {
		t.Error("Data should be allocated")
	}
}

func TestExtraEventPayload_JSONMarshal(t *testing.T) {
	p := NewEventPayload()
	p.AppendEvent(CAPIEventData{
		EventName:    "Purchase",
		EventTime:    time.Now().Unix(),
		ActionSource: "website",
	})
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"data":[`)) {
		t.Error("data should be array")
	}
}

func TestExtraCAPIEventData_Minimal(t *testing.T) {
	e := CAPIEventData{
		EventName:    "Lead",
		EventTime:    1700000000,
		ActionSource: "website",
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("event_name")) {
		t.Error("event_name missing")
	}
}

func TestExtraUserData_JSON(t *testing.T) {
	u := UserData{
		Email:      "test@example.com",
		Phone:      "0901234567",
		FBCookieID: "fb.1.123.456",
		FBPIDCookieID: "fb.1.789.012",
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"em":"test@example.com"`)) {
		t.Error("em field missing")
	}
	if !bytes.Contains(b, []byte(`"ph":"0901234567"`)) {
		t.Error("ph field missing")
	}
	if !bytes.Contains(b, []byte(`"fbc":"fb.1.123.456"`)) {
		t.Error("fbc field missing")
	}
}

func TestExtraCustomData_JSON(t *testing.T) {
	c := CustomData{
		Value:    99.99,
		Currency: "USD",
		OrderID:  "order-1",
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"value":99.99`)) {
		t.Error("value missing")
	}
}

func TestExtraContentItem(t *testing.T) {
	c := ContentItem{ID: "p1", Quantity: 2, Price: 50.0}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"id":"p1"`)) {
		t.Error("id missing")
	}
}

func TestExtraAPIResponse_Parse(t *testing.T) {
	body := `{"events_received":[{"event_id":"x","fb_event_id":"fb_x"}],"messages":["ok"]}`
	var r APIResponse
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Events) != 1 {
		t.Errorf("events: %d", len(r.Events))
	}
	if r.Events[0].FBEventID != "fb_x" {
		t.Errorf("fb_event_id: %s", r.Events[0].FBEventID)
	}
}

func TestExtraEventResult_Fields(t *testing.T) {
	r := EventResult{
		EventID:      "x",
		ErrorCode:    100,
		ErrorMessage: "invalid",
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"error_code":100`)) {
		t.Error("error_code missing")
	}
}

func TestExtraClient_NewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("nil")
	}
	if c.httpClient == nil {
		t.Error("http client nil")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout: %v", c.httpClient.Timeout)
	}
}

func TestExtraSendEventsToURL_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"events_received":[{"event_id":"e1","fb_event_id":"fb_e1"}]}`))
	}))
	defer srv.Close()

	c := NewClient()
	p := NewEventPayload()
	p.AppendEvent(CAPIEventData{EventName: "Lead", ActionSource: "website"})
	results, err := c.SendEventsToURL(context.Background(), srv.URL+"/events", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestExtraSendEventsToURL_WithMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events_received":[],"messages":["error message"]}`))
	}))
	defer srv.Close()

	c := NewClient()
	results, err := c.SendEventsToURL(context.Background(), srv.URL+"/events", NewEventPayload())
	if err == nil {
		t.Error("expected error when messages present")
	}
	_ = results
}

func TestExtraSendEventsToURL_BadServer(t *testing.T) {
	c := NewClient()
	_, err := c.SendEventsToURL(context.Background(), "http://127.0.0.1:1/events", NewEventPayload())
	if err == nil {
		t.Error("expected error from bad server")
	}
}

func TestExtraSendEventsToURL_ResponseMethodIsPOST(t *testing.T) {
	methodGot := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodGot = r.Method
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte("data")) {
			t.Error("body should have data field")
		}
		w.Write([]byte(`{"events_received":[]}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.SendEventsToURL(context.Background(), srv.URL+"/events", NewEventPayload())
	if methodGot != http.MethodPost {
		t.Errorf("expected POST, got %s", methodGot)
	}
}

func TestExtraTestConnection_Ok(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	}))
	defer srv.Close()

	c := NewClient()
	if err := c.TestConnectionToURL(context.Background(), srv.URL); err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestExtraTestConnection_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid_token"}}`))
	}))
	defer srv.Close()

	c := NewClient()
	if err := c.TestConnectionToURL(context.Background(), srv.URL); err == nil {
		t.Error("expected error from 401")
	}
}

func TestExtraProcessingOptions_JSON(t *testing.T) {
	p := ProcessingOptions{AllowNoSales: true}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("true")) {
		t.Error("expected true in JSON")
	}
}

func TestExtraDebugMode_JSON(t *testing.T) {
	d := DebugMode{Mode: 1}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"mode":1`)) {
		t.Error("mode field missing")
	}
}

func TestExtraCustomProps_JSON(t *testing.T) {
	c := CustomData{
		CustomProps: map[string]string{"k1": "v1"},
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"k1":"v1"`)) {
		t.Error("custom_properties missing")
	}
}

func TestExtraContents_JSON(t *testing.T) {
	c := CustomData{
		Contents: []ContentItem{{ID: "p1"}},
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"contents":`)) {
		t.Error("contents missing")
	}
}

func TestExtraStatusCheck(t *testing.T) {
	// Verify SendEventsToURL response is parsed via httptest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			t.Error("missing content type")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events_received":[]}`))
	}))
	defer srv.Close()

	c := NewClient()
	_, err := c.SendEventsToURL(context.Background(), srv.URL, NewEventPayload())
	if err != nil {
		t.Errorf("expected ok, got %v", err)
	}
}
