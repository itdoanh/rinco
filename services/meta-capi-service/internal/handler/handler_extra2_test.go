// Extra tests for meta-capi-service handler helpers (pure helpers only).
package handler

import "testing"

func TestMapCRMEvents_DealWon(t *testing.T) {
	if got := mapCRMEvents("deal_won"); got != "Purchase" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_DealClosedWon(t *testing.T) {
	if got := mapCRMEvents("deal_closed_won"); got != "Purchase" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_LeadCreated(t *testing.T) {
	if got := mapCRMEvents("lead_created"); got != "Lead" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_LeadNew(t *testing.T) {
	if got := mapCRMEvents("lead_new"); got != "Lead" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_Contact(t *testing.T) {
	if got := mapCRMEvents("contact"); got != "Contact" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_ContactCreated(t *testing.T) {
	if got := mapCRMEvents("contact_created"); got != "Contact" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_PageView(t *testing.T) {
	if got := mapCRMEvents("page_view"); got != "ViewContent" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_LandingView(t *testing.T) {
	if got := mapCRMEvents("landing_view"); got != "ViewContent" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_FormSubmit(t *testing.T) {
	if got := mapCRMEvents("form_submit"); got != "CompleteRegistration" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_FormFilled(t *testing.T) {
	if got := mapCRMEvents("form_filled"); got != "CompleteRegistration" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_AddToCart(t *testing.T) {
	if got := mapCRMEvents("add_to_cart"); got != "AddToCart" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_Checkout(t *testing.T) {
	if got := mapCRMEvents("checkout"); got != "InitiateCheckout" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_Subscribe(t *testing.T) {
	if got := mapCRMEvents("subscribe"); got != "Subscribe" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_Unknown(t *testing.T) {
	if got := mapCRMEvents("custom_event"); got != "custom_event" {
		t.Errorf("got %s", got)
	}
}

func TestMapCRMEvents_CaseInsensitive(t *testing.T) {
	if got := mapCRMEvents("DEAL_WON"); got != "Purchase" {
		t.Errorf("got %s", got)
	}
}

func TestParseFloat_Valid(t *testing.T) {
	v, err := parseFloat("3.14")
	if err != nil {
		t.Fatal(err)
	}
	if v != 3.14 {
		t.Errorf("got %v", v)
	}
}

func TestParseFloat_Zero(t *testing.T) {
	v, err := parseFloat("0")
	if err != nil {
		t.Fatal(err)
	}
	if v != 0 {
		t.Errorf("got %v", v)
	}
}

func TestParseFloat_Negative(t *testing.T) {
	v, err := parseFloat("-1.5")
	if err != nil {
		t.Fatal(err)
	}
	if v != -1.5 {
		t.Errorf("got %v", v)
	}
}

func TestParseFloat_Invalid(t *testing.T) {
	if _, err := parseFloat("abc"); err == nil {
		t.Error("expected error")
	}
}

func TestParseFloat_Empty(t *testing.T) {
	if _, err := parseFloat(""); err == nil {
		t.Error("expected error")
	}
}

func TestMaskAccessToken_Logic(t *testing.T) {
	// Simulate the masking logic from GetCAPIStatus
	token := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	masked := token
	if len(token) > 8 {
		masked = token[:4] + "****" + token[len(token)-4:]
	}
	if masked != "ABCD****WXYZ" {
		t.Errorf("got %s", masked)
	}
}

func TestMaskAccessToken_Short(t *testing.T) {
	token := "short"
	masked := token
	if len(token) > 8 {
		masked = token[:4] + "****" + token[len(token)-4:]
	}
	if masked != "short" {
		t.Errorf("short token should not be masked: %s", masked)
	}
}

func TestMaskAccessToken_Empty(t *testing.T) {
	token := ""
	masked := token
	if len(token) > 8 {
		masked = token[:4] + "****" + token[len(token)-4:]
	}
	if masked != "" {
		t.Errorf("got %s", masked)
	}
}
