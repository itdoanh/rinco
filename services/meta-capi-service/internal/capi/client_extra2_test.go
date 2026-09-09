// Extra tests for meta-capi-service client payload structures.
package capi

import (
	"encoding/json"
	"testing"
)

func TestNewEventPayload_Empty(t *testing.T) {
	p := NewEventPayload()
	if p.Data == nil {
		t.Error("Data should not be nil")
	}
	if len(p.Data) != 0 {
		t.Errorf("Data len: %d", len(p.Data))
	}
}

func TestEventPayload_AppendEvent_Ex2(t *testing.T) {
	p := NewEventPayload()
	p.AppendEvent(CAPIEventData{EventName: "Lead"})
	if len(p.Data) != 1 {
		t.Errorf("Data len: %d", len(p.Data))
	}
}

func TestEventPayload_AppendMultiple(t *testing.T) {
	p := NewEventPayload()
	for i := 0; i < 5; i++ {
		p.AppendEvent(CAPIEventData{EventName: "Lead"})
	}
	if len(p.Data) != 5 {
		t.Errorf("Data len: %d", len(p.Data))
	}
}

func TestEventPayload_JSONMarshal_Empty(t *testing.T) {
	p := NewEventPayload()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	// Should contain "data":[] not "data":null
	s := string(b)
	if s == `{"Data":null}` {
		t.Error("Data field should not be null")
	}
}

func TestCAPIEventData_JSONRoundTrip(t *testing.T) {
	ev := CAPIEventData{
		EventID:      "evt-1",
		EventName:    "Purchase",
		EventTime:    1700000000,
		ActionSource: "website",
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var got CAPIEventData
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.EventID != ev.EventID {
		t.Errorf("EventID: %s vs %s", got.EventID, ev.EventID)
	}
}

func TestUserData_Fields(t *testing.T) {
	u := UserData{
		Email: "test@example.com",
		Phone: "+1234567890",
	}
	if u.Email == "" {
		t.Error("Email not set")
	}
}

func TestCustomData_WithValue(t *testing.T) {
	c := CustomData{
		Value:    99.99,
		Currency: "USD",
	}
	if c.Value != 99.99 {
		t.Errorf("Value: %f", c.Value)
	}
}

func TestContentItem_Fields(t *testing.T) {
	c := ContentItem{
		ID:       "item-1",
		Quantity: 2,
		Price:    9.99,
	}
	if c.ID != "item-1" {
		t.Error("ID")
	}
}

func TestProcessingOptions_Fields(t *testing.T) {
	p := ProcessingOptions{
		OverrideUserData: true,
	}
	if !p.OverrideUserData {
		t.Error("OverrideUserData")
	}
}

func TestDebugMode_Fields(t *testing.T) {
	d := DebugMode{Mode: 1}
	if d.Mode != 1 {
		t.Error("Mode")
	}
}

func TestAPIResponse_JSONUnmarshal(t *testing.T) {
	jsonStr := `{
		"events_received": [{"event_id": "evt-1", "fb_event_id": "fb-1"}],
		"messages": ["ok"],
		"id": "test-id"
	}`
	var resp APIResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Events) != 1 {
		t.Errorf("Events: %d", len(resp.Events))
	}
	if resp.Events[0].EventID != "evt-1" {
		t.Errorf("EventID: %s", resp.Events[0].EventID)
	}
}

func TestEventResult_Fields(t *testing.T) {
	er := EventResult{
		EventID:      "e1",
		FBEventID:    "fb-1",
		ErrorCode:    100,
		ErrorMessage: "rate limit",
	}
	if er.EventID != "e1" {
		t.Error("EventID")
	}
	if er.ErrorCode != 100 {
		t.Errorf("ErrorCode: %d", er.ErrorCode)
	}
}

func TestNewClient_Ex2(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("nil")
	}
	if c.httpClient == nil {
		t.Error("nil client")
	}
	if c.httpClient.Timeout == 0 {
		t.Error("timeout should be set")
	}
}

func TestEventPayload_AppendToNil(t *testing.T) {
	p := EventPayload{Data: nil}
	p.AppendEvent(CAPIEventData{EventName: "Lead"})
	if p.Data == nil {
		t.Error("Data should be allocated")
	}
	if len(p.Data) != 1 {
		t.Errorf("len: %d", len(p.Data))
	}
}
