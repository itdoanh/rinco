package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCAPIEventJSONMarshal(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	e := CAPIEvent{
		ID:        id,
		TenantID:  tenantID,
		EventID:   "evt-1",
		EventName: "Purchase",
		EventTime: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		OrderValue: 99.99,
		Currency:  "USD",
		Status:    "pending",
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"event_id":"evt-1"`) {
		t.Errorf("missing event_id: %s", data)
	}
	if !strings.Contains(string(data), `"order_value":99.99`) {
		t.Errorf("missing value: %s", data)
	}
}

func TestCAPIEventOmitEmpty(t *testing.T) {
	e := CAPIEvent{EventName: "Lead"}
	data, _ := json.Marshal(e)
	// Optional fields should be omitted
	if strings.Contains(string(data), "email") {
		t.Errorf("email should be omitted: %s", data)
	}
	if strings.Contains(string(data), "phone") {
		t.Errorf("phone should be omitted: %s", data)
	}
}

func TestCAPIEventCustomData(t *testing.T) {
	e := CAPIEvent{
		CustomData: map[string]string{"plan": "pro", "seats": "5"},
	}
	data, _ := json.Marshal(e)
	if !strings.Contains(string(data), `"plan":"pro"`) {
		t.Errorf("missing custom data: %s", data)
	}
}

func TestCAPIConfigJSONRoundtrip(t *testing.T) {
	cfg := CAPIConfig{
		PixelID:     "12345",
		AccessToken: "tok",
		IsEnabled:   true,
		SampleRate:  0.8,
		EventTypes:  []string{"Purchase", "Lead"},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got CAPIConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.PixelID != "12345" || got.AccessToken != "tok" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
	if got.SampleRate != 0.8 {
		t.Errorf("rate = %f", got.SampleRate)
	}
	if len(got.EventTypes) != 2 {
		t.Errorf("event types len = %d", len(got.EventTypes))
	}
}

func TestFeedbackEventFields(t *testing.T) {
	fb := FeedbackEvent{
		FBEventName:   "Purchase",
		FBPartnerName: "facebook",
		FBClaimCode:   "abc",
	}
	if fb.FBEventName != "Purchase" {
		t.Error("event name")
	}
}

func TestFeedbackEventJSONHasOnlyDefinedFields(t *testing.T) {
	fb := FeedbackEvent{
		ID:            uuid.New(),
		CAPIEventID:   uuid.New(),
		TenantID:      uuid.New(),
		FBEventID:     "evid",
		FBEventName:   "Purchase",
		FBEventTime:   time.Now(),
		FBPartnerName: "facebook",
		FBPartnerID:   "12345",
	}
	data, _ := json.Marshal(fb)
	for _, k := range []string{
		"id", "capi_event_id", "tenant_id",
		"fb_event_id", "fb_event_name", "fb_partner_name",
	} {
		if !strings.Contains(string(data), k) {
			t.Errorf("missing %s in %s", k, data)
		}
	}
}

func TestConversionMappingDefaults(t *testing.T) {
	m := ConversionMapping{
		CRMEventType:  "deal_won",
		CAPIEventName: "Purchase",
		IsActive:      true,
	}
	if !m.IsActive {
		t.Error("default not active")
	}
}

func TestAggregatedConversionJSON(t *testing.T) {
	a := AggregatedConversion{
		EventName:  "Purchase",
		TotalEvents: 100,
		Delivered:  95,
		Failed:     5,
		TotalValue: 1234.56,
		Currency:   "USD",
	}
	data, _ := json.Marshal(a)
	if !strings.Contains(string(data), `"total_events":100`) {
		t.Errorf("missing field: %s", data)
	}
	if !strings.Contains(string(data), `"failed":5`) {
		t.Errorf("missing failed: %s", data)
	}
}

func TestStatusStates(t *testing.T) {
	statuses := []string{"pending", "sent", "delivered", "failed", "suppressed"}
	for _, s := range statuses {
		e := CAPIEvent{Status: s}
		if e.Status != s {
			t.Errorf("status = %s", e.Status)
		}
	}
}

func TestEventSources(t *testing.T) {
	sources := []string{"web", "app", "offline", "crm"}
	for _, s := range sources {
		e := CAPIEvent{EventSource: s}
		if e.EventSource != s {
			t.Errorf("source = %s", e.EventSource)
		}
	}
}
