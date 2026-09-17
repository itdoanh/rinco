// Package handler provides additional pure helpers for the meta-capi-service.
//
// mapCRMEvents converts a CRM event type to a Meta CAPI event name.
// parseFloat converts a string to float64 with safe error handling.
package handler

import (
	"strconv"
	"strings"
)

// mapCRMEvents maps a CRM event type string to a Meta CAPI event name.
//
// The mapping is case-insensitive and falls back to the input string when
// no mapping exists (so callers can still record custom events).
func mapCRMEvents(crmEvent string) string {
	switch strings.ToLower(strings.TrimSpace(crmEvent)) {
	case "deal_won", "deal_closed_won", "purchase":
		return "Purchase"
	case "lead_created", "lead_new", "lead":
		return "Lead"
	case "contact", "contact_created":
		return "Contact"
	case "page_view", "landing_view", "view_content":
		return "ViewContent"
	case "form_submit", "form_filled", "complete_registration":
		return "CompleteRegistration"
	case "add_to_cart":
		return "AddToCart"
	case "checkout", "initiate_checkout":
		return "InitiateCheckout"
	case "subscribe":
		return "Subscribe"
	default:
		return crmEvent
	}
}

// parseFloat converts a string to float64. Returns 0 and an error when
// the input is invalid.
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, &parseError{s: s, msg: "empty string"}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, &parseError{s: s, msg: err.Error()}
	}
	return v, nil
}

type parseError struct {
	s, msg string
}

func (e *parseError) Error() string { return "parseFloat(" + e.s + "): " + e.msg }
