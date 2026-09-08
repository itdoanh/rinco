// Additional tests for capi package event functions.
package capi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHashEmail_CaseInsensitive(t *testing.T) {
	// Same email should always produce same hash.
	h1 := HashEmail("test@example.com")
	h2 := HashEmail("TEST@example.com")
	if h1 != h2 {
		t.Errorf("case-insensitive hash mismatch: %s != %s", h1, h2)
	}

	// Hash should be 64 hex chars (SHA256).
	if len(h1) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(h1))
	}
}

func TestHashEmail_Whitespace_Extra(t *testing.T) {
	h1 := HashEmail("test@example.com")
	h2 := HashEmail("  test@example.com  ")
	if h1 != h2 {
		t.Errorf("whitespace should be trimmed: %s != %s", h1, h2)
	}
}

func TestHashPhone_Normalization_Extra(t *testing.T) {
	h1 := HashPhone("0901234567")
	h2 := HashPhone("090-123-4567")

	// Both variants should produce same hash after digit-only normalization.
	if h1 != h2 {
		t.Errorf("phone normalization failed: %s != %s", h1, h2)
	}
}

func TestHashString_Deterministic_Extra(t *testing.T) {
	h1 := HashString("hello")
	h2 := HashString("hello")
	if h1 != h2 {
		t.Errorf("HashString not deterministic: %s != %s", h1, h2)
	}

	h3 := HashString("world")
	if h1 == h3 {
		t.Error("different strings should produce different hashes")
	}
}

func TestHashString_Empty_Extra(t *testing.T) {
	h := HashString("")
	if h == "" {
		t.Error("empty string should still produce a hash")
	}
	if len(h) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(h))
	}
}

func TestHashEmail_DifferentDomains_Extra(t *testing.T) {
	h1 := HashEmail("user@gmail.com")
	h2 := HashEmail("user@yahoo.com")
	if h1 == h2 {
		t.Error("different domains should produce different hashes")
	}
}

func TestHashPhone_Empty_Extra(t *testing.T) {
	h := HashPhone("")
	if h == "" {
		t.Error("empty phone should still produce hash")
	}
	if len(h) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(h))
	}
}

func TestEvent_JSON_Roundtrip(t *testing.T) {
	e := Event{
		EventID:      "evt-1",
		EventName:    "Purchase",
		EventTime:    1700000000,
		ActionSource: "website",
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got Event
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.EventName != "Purchase" {
		t.Errorf("EventName: got %s", got.EventName)
	}
}

func TestUserData_JSON_Extra(t *testing.T) {
	ud := &UserData{
		Email:       "test@example.com", // raw, will be sent as "em"
		ExternalID:  "ext-1",
		FBCookieID:  "fb.1.timestamp",
		FBPCookieID: "fbp.1.timestamp",
	}
	b, err := json.Marshal(ud)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "em") {
		t.Error("missing em field")
	}
	if !strings.Contains(s, "external_id") {
		t.Error("missing external_id field")
	}
}

func TestCustomData_JSON_Extra(t *testing.T) {
	cd := &CustomData{
		Value:    99.99,
		Currency: "USD",
		OrderID:  "order-123",
		NumItems: 2,
	}
	b, err := json.Marshal(cd)
	if err != nil {
		t.Fatal(err)
	}
	var got CustomData
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Value != 99.99 {
		t.Errorf("Value: got %f", got.Value)
	}
}

func TestContentItem_JSON_Extra(t *testing.T) {
	ci := ContentItem{
		ID:        "item-1",
		Quantity:  2,
		ItemPrice: 49.99,
		Title:     "Test Product",
		Brand:     "TestBrand",
	}
	b, err := json.Marshal(ci)
	if err != nil {
		t.Fatal(err)
	}
	var got ContentItem
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Quantity != 2 {
		t.Errorf("Quantity: got %d", got.Quantity)
	}
}

func TestHashString_SHA256Format(t *testing.T) {
	h := HashString("test")
	// SHA256 produces 32 bytes = 64 hex chars
	if len(h) != 64 {
		t.Errorf("expected 64 chars, got %d", len(h))
	}
	// Should be lowercase hex
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex char in hash: %c", c)
			break
		}
	}
}

func TestHashEmail_SpecialChars(t *testing.T) {
	h1 := HashEmail("user+tag@example.com")
	h2 := HashEmail("usertag@example.com")
	// Different emails should produce different hashes
	if h1 == h2 {
		t.Error("plus-tagged email should hash differently")
	}
}

func TestHashPhone_International(t *testing.T) {
	h1 := HashPhone("+1-555-123-4567")
	// After normalization should be just digits
	if len(h1) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(h1))
	}
}
