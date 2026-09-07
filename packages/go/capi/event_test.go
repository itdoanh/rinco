package capi

import (
	"testing"
)

func TestHashEmail(t *testing.T) {
	h := HashEmail("Test@Example.COM")
	if h == "" || len(h) != 64 {
		t.Errorf("HashEmail length = %d, want 64", len(h))
	}

	// Lowercase + trim should give same hash
	h2 := HashEmail("  test@example.com  ")
	if h != h2 {
		t.Errorf("HashEmail not normalized: %s vs %s", h, h2)
	}
}

func TestHashPhone(t *testing.T) {
	h := HashPhone("+84 901 234 567")
	if h == "" || len(h) != 64 {
		t.Errorf("HashPhone length = %d", len(h))
	}
	// Should strip non-digit characters
	h2 := HashPhone("84901234567")
	if h != h2 {
		t.Errorf("HashPhone should normalize: %s vs %s", h, h2)
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"+84 901 234 567", "84901234567"},
		{"(090) 123-4567", "0901234567"},
		{"", ""},
	}
	for _, c := range cases {
		got := NormalizePhone(c.in)
		if got != c.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNewUserData(t *testing.T) {
	ud := NewUserData("User@Example.com", "+1-555-1234")
	if ud.Email == "" {
		t.Error("expected hashed email")
	}
	if ud.Phone == "" {
		t.Error("expected hashed phone")
	}
	if ud.Email == "User@Example.com" {
		t.Error("email should be hashed, not plain")
	}
}

func TestNewLeadEvent(t *testing.T) {
	event := NewLeadEvent("user@example.com", "+84901234567", "1.2.3.4", "Mozilla/5.0", 100.50, "VND")
	if event.EventName != EventLead {
		t.Errorf("EventName = %q, want %q", event.EventName, EventLead)
	}
	if event.ActionSource != ActionSourceWebsite {
		t.Errorf("ActionSource = %q", event.ActionSource)
	}
	if event.CustomData == nil || event.CustomData.Value != 100.50 {
		t.Error("CustomData.Value not set")
	}
	if event.CustomData.Currency != "VND" {
		t.Errorf("Currency = %q", event.CustomData.Currency)
	}
	if event.UserData == nil {
		t.Fatal("UserData nil")
	}
	if event.UserData.ClientIPAddress != "1.2.3.4" {
		t.Errorf("ClientIPAddress = %q", event.UserData.ClientIPAddress)
	}
}

func TestNewPurchaseEvent(t *testing.T) {
	contents := []ContentItem{
		{ID: "SKU1", Quantity: 2, ItemPrice: 50.0},
	}
	event := NewPurchaseEvent("buyer@example.com", "5.6.7.8", "Mozilla/5.0", 100.0, "USD", "ORDER-1", contents)
	if event.EventName != EventPurchase {
		t.Errorf("EventName = %q", event.EventName)
	}
	if event.CustomData.NumItems != 1 {
		t.Errorf("NumItems = %d, want 1", event.CustomData.NumItems)
	}
	if event.CustomData.OrderID != "ORDER-1" {
		t.Errorf("OrderID = %q", event.CustomData.OrderID)
	}
}

func TestEventIDGeneration(t *testing.T) {
	id1 := GenerateEventID("tenant-a", "contact-form", "user@x.com", 1700000000)
	id2 := GenerateEventID("tenant-a", "contact-form", "user@x.com", 1700000000)
	if id1 != id2 {
		t.Error("same inputs should give same id (deterministic)")
	}
	id3 := GenerateEventID("tenant-a", "contact-form", "user@x.com", 1700000001)
	if id1 == id3 {
		t.Error("different timestamps should give different ids")
	}
	if len(id1) != 64 {
		t.Errorf("id length = %d, want 64", len(id1))
	}
}

func TestNewCompleteRegistrationEvent(t *testing.T) {
	event := NewCompleteRegistrationEvent("u@x.com", "Nguyen", "Van A", "1.2.3.4", "Mozilla")
	if event.EventName != EventCompleteRegistration {
		t.Errorf("EventName = %q", event.EventName)
	}
	if event.UserData.FirstName == "" {
		t.Error("FirstName should be hashed")
	}
	if event.UserData.LastName == "" {
		t.Error("LastName should be hashed")
	}
}

func TestServerEventFluentAPI(t *testing.T) {
	event := NewServerEvent("Test", ActionSourceWebsite).
		WithEventID("evt-123").
		WithContext("9.9.9.9", "UA")

	if event.EventID != "evt-123" {
		t.Errorf("EventID = %q", event.EventID)
	}
	if event.IPAddress != "9.9.9.9" {
		t.Errorf("IPAddress = %q", event.IPAddress)
	}
}
