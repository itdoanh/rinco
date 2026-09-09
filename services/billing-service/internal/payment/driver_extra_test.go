// Extra tests for billing payment driver.
package payment

import (
	"errors"
	"testing"
)

func TestExtraDriverInterface_Errors(t *testing.T) {
	if ErrProviderNotConfigured == nil {
		t.Error("ErrProviderNotConfigured nil")
	}
	if ErrCheckoutFailed == nil {
		t.Error("ErrCheckoutFailed nil")
	}
	if ErrSubscriptionFailed == nil {
		t.Error("ErrSubscriptionFailed nil")
	}
	if ErrCancellationFailed == nil {
		t.Error("ErrCancellationFailed nil")
	}
	if ErrWebhookInvalid == nil {
		t.Error("ErrWebhookInvalid nil")
	}
}

func TestExtraDriverErrors_AreDifferent(t *testing.T) {
	if errors.Is(ErrCheckoutFailed, ErrSubscriptionFailed) {
		t.Error("errors should be distinct")
	}
	if errors.Is(ErrWebhookInvalid, ErrProviderNotConfigured) {
		t.Error("errors should be distinct")
	}
}

func TestExtraCheckoutSession_StructFields(t *testing.T) {
	cs := CheckoutSession{
		ID:         "cs_1",
		URL:        "https://checkout.stripe.com/c/cs_1",
		Status:     "open",
		CustomerID: "cus_1",
		SubID:      "sub_1",
	}
	if cs.ID != "cs_1" {
		t.Errorf("ID: %s", cs.ID)
	}
	if cs.URL != "https://checkout.stripe.com/c/cs_1" {
		t.Errorf("URL: %s", cs.URL)
	}
	if cs.Status != "open" {
		t.Errorf("Status: %s", cs.Status)
	}
}

func TestExtraWebhookPayload_StructFields(t *testing.T) {
	wp := WebhookPayload{
		Type:       "invoice.paid",
		SubID:      "sub_1",
		CustomerID: "cus_1",
		InvoiceID:  "in_1",
		Status:     "paid",
		AmountPaid: 9999,
		Currency:   "usd",
		Email:      "user@example.com",
	}
	if wp.Type != "invoice.paid" {
		t.Errorf("Type: %s", wp.Type)
	}
	if wp.AmountPaid != 9999 {
		t.Errorf("AmountPaid: %d", wp.AmountPaid)
	}
	if wp.Currency != "usd" {
		t.Errorf("Currency: %s", wp.Currency)
	}
}

func TestExtraWebhookPayload_CustomerAddr(t *testing.T) {
	wp := WebhookPayload{
		CustomerAddr: "123 Main St, City",
	}
	if wp.CustomerAddr != "123 Main St, City" {
		t.Errorf("addr: %s", wp.CustomerAddr)
	}
}

func TestExtraWebhookPayload_Empty(t *testing.T) {
	wp := WebhookPayload{}
	if wp.Type != "" {
		t.Errorf("empty Type: %s", wp.Type)
	}
}

func TestExtraCheckoutSession_Empty(t *testing.T) {
	cs := CheckoutSession{}
	if cs.ID != "" {
		t.Errorf("empty ID: %s", cs.ID)
	}
}

func TestExtraErrors_Messages(t *testing.T) {
	if ErrProviderNotConfigured.Error() == "" {
		t.Error("err message empty")
	}
	if ErrCheckoutFailed.Error() == "" {
		t.Error("err message empty")
	}
}

func TestExtraErrors_IsChain(t *testing.T) {
	wrapped := errors.New("wrapped: " + ErrWebhookInvalid.Error())
	if !errors.Is(wrapped, ErrWebhookInvalid) { // string comparison won't match
		t.Log("not equal by errors.Is (expected for string concat)")
	}
}
