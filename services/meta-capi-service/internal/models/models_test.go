// Tests for meta-capi-service models.
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCAPIEvent_JSON(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	now := time.Now()
	e := CAPIEvent{
		ID:          id,
		TenantID:    tenantID,
		EventID:     "event-1",
		EventName:   "Purchase",
		EventTime:   now,
		EventSource: "web",
		Email:       "test@example.com",
		OrderValue:  99.99,
		Currency:    "USD",
		Status:      "pending",
		CreatedAt:   now,
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got CAPIEvent
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != id {
		t.Errorf("ID: got %v", got.ID)
	}
	if got.TenantID != tenantID {
		t.Errorf("TenantID: got %v", got.TenantID)
	}
	if got.EventName != "Purchase" {
		t.Errorf("EventName: got %s", got.EventName)
	}
	if got.OrderValue != 99.99 {
		t.Errorf("OrderValue: got %f", got.OrderValue)
	}
}

func TestCAPIConfig_JSON(t *testing.T) {
	c := CAPIConfig{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		PixelID:     "1234567890",
		AccessToken: "secret-token",
		IsEnabled:   true,
		EventTypes:  []string{"Purchase", "Lead"},
		SampleRate:  1.0,
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var got CAPIConfig
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.PixelID != "1234567890" {
		t.Errorf("PixelID: got %s", got.PixelID)
	}
	if !got.IsEnabled {
		t.Error("IsEnabled should be true")
	}
	if got.SampleRate != 1.0 {
		t.Errorf("SampleRate: got %f", got.SampleRate)
	}
}

func TestCAPIConfig_Disabled(t *testing.T) {
	c := CAPIConfig{IsEnabled: false}
	if c.IsEnabled {
		t.Error("IsEnabled should be false")
	}
}

func TestCAPIConfig_SampleRates(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"full", 1.0},
		{"half", 0.5},
		{"quarter", 0.25},
		{"none", 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CAPIConfig{SampleRate: tt.rate}
			if c.SampleRate != tt.rate {
				t.Errorf("SampleRate: got %f, want %f", c.SampleRate, tt.rate)
			}
		})
	}
}

func TestFeedbackEvent_JSON(t *testing.T) {
	f := FeedbackEvent{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		FBEventName:  "Purchase",
		FBPartnerName: "rinco",
		ProcessedAt:  time.Now(),
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var got FeedbackEvent
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.FBEventName != "Purchase" {
		t.Errorf("FBEventName: got %s", got.FBEventName)
	}
	if got.FBPartnerName != "rinco" {
		t.Errorf("FBPartnerName: got %s", got.FBPartnerName)
	}
}

func TestConversionMapping_JSON(t *testing.T) {
	m := ConversionMapping{
		ID:            uuid.New(),
		TenantID:      uuid.New(),
		CRMEventType:  "deal_won",
		CAPIEventName: "Purchase",
		IsActive:      true,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got ConversionMapping
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.CRMEventType != "deal_won" {
		t.Errorf("CRMEventType: got %s", got.CRMEventType)
	}
	if got.CAPIEventName != "Purchase" {
		t.Errorf("CAPIEventName: got %s", got.CAPIEventName)
	}
}

func TestCAPIEvent_StatusValues(t *testing.T) {
	statuses := []string{"pending", "sent", "delivered", "failed", "suppressed"}
	for _, s := range statuses {
		t.Run(s, func(t *testing.T) {
			e := CAPIEvent{Status: s}
			if e.Status != s {
				t.Errorf("Status: got %s, want %s", e.Status, s)
			}
		})
	}
}

func TestCAPIEvent_SentAtPointer(t *testing.T) {
	now := time.Now()
	e := CAPIEvent{SentAt: &now}
	if e.SentAt == nil {
		t.Fatal("SentAt nil")
	}
	if !e.SentAt.Equal(now) {
		t.Error("SentAt mismatch")
	}
}
