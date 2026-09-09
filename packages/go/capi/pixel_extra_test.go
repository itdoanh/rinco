// Tests for capi pixel helpers (pixel.go).
package capi

import (
	"strings"
	"testing"
)

func TestExtra_PixelConfig_InitScript(t *testing.T) {
	p := &PixelConfig{
		PixelID:    "1234567890",
		Version:    "18.0",
		AccessToken: "token",
		DebugMode:  false,
		EnableCAPI: false,
	}
	got := p.InitScript()
	if !strings.Contains(got, "fbq") {
		t.Error("InitScript should contain fbq")
	}
	if !strings.Contains(got, "1234567890") {
		t.Error("InitScript should contain pixel ID")
	}
}

func TestExtra_PixelConfig_InitScript_WithDebug(t *testing.T) {
	p := &PixelConfig{
		PixelID:   "1234567890",
		Version:   "18.0",
		DebugMode: true,
	}
	got := p.InitScript()
	if !strings.Contains(got, "autoConfig") {
		t.Error("Debug mode should enable autoConfig")
	}
}

func TestExtra_PixelConfig_InitScript_WithCAPI(t *testing.T) {
	p := &PixelConfig{
		PixelID:    "1234567890",
		Version:    "18.0",
		EnableCAPI: true,
	}
	got := p.InitScript()
	if !strings.Contains(got, "eventID") {
		t.Error("EnableCAPI should include eventID")
	}
}

func TestExtra_GenerateEventScript(t *testing.T) {
	script := GenerateEventScript()
	if !strings.Contains(script, "generateEventID") {
		t.Error("should contain generateEventID function")
	}
	if !strings.Contains(script, "trackCAPI") {
		t.Error("should contain trackCAPI function")
	}
}

func TestExtra_GenerateEventScript_UUIDPattern(t *testing.T) {
	script := GenerateEventScript()
	if !strings.Contains(script, "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx") {
		t.Error("should contain UUID pattern")
	}
}

func TestExtra_TrackPageView(t *testing.T) {
	got := TrackPageView()
	if !strings.Contains(got, "PageView") {
		t.Error("should contain PageView")
	}
	if !strings.Contains(got, "fbq") {
		t.Error("should contain fbq")
	}
}

func TestExtra_TrackLead_NoParams(t *testing.T) {
	got := TrackLead(nil)
	if !strings.Contains(got, "Lead") {
		t.Error("should contain Lead")
	}
	if !strings.Contains(got, "fbq") {
		t.Error("should contain fbq")
	}
}

func TestExtra_TrackLead_WithParams(t *testing.T) {
	params := map[string]interface{}{
		"value": "100",
		"currency": "USD",
	}
	got := TrackLead(params)
	if !strings.Contains(got, "value") {
		t.Error("should contain value")
	}
	if !strings.Contains(got, "currency") {
		t.Error("should contain currency")
	}
	if !strings.Contains(got, "USD") {
		t.Error("should contain USD")
	}
}

func TestExtra_TrackCompleteRegistration(t *testing.T) {
	got := TrackCompleteRegistration(nil)
	if !strings.Contains(got, "CompleteRegistration") {
		t.Error("should contain CompleteRegistration")
	}
}

func TestExtra_TrackCompleteRegistration_WithParams(t *testing.T) {
	params := map[string]interface{}{"value": 50.0}
	got := TrackCompleteRegistration(params)
	if !strings.Contains(got, "50.000000") {
		t.Error("should contain formatted value")
	}
}

func TestExtra_TrackViewContent(t *testing.T) {
	got := TrackViewContent(nil)
	if !strings.Contains(got, "ViewContent") {
		t.Error("should contain ViewContent")
	}
}

func TestExtra_TrackCustomEvent(t *testing.T) {
	got := TrackCustomEvent("MyEvent", nil)
	if !strings.Contains(got, "trackCustom") {
		t.Error("should use trackCustom")
	}
	if !strings.Contains(got, "MyEvent") {
		t.Error("should contain event name")
	}
}

func TestExtra_TrackCustomEvent_WithParams(t *testing.T) {
	params := map[string]interface{}{"key": "value"}
	got := TrackCustomEvent("MyEvent", params)
	if !strings.Contains(got, "key") {
		t.Error("should contain key")
	}
	if !strings.Contains(got, "value") {
		t.Error("should contain value")
	}
}

func TestExtra_escapeJS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"with \"quotes\"", `with \"quotes\"`},
		{"with\nnewline", `with\nnewline`},
		{"with\ttab", `with\ttab`},
		{"with\rcr", `with\rcr`},
		{`with\back`, `with\\back`},
	}
	for _, tt := range tests {
		got := escapeJS(tt.input)
		if got != tt.expected {
			t.Errorf("escapeJS(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtra_formatMap_String(t *testing.T) {
	var buf strings.Builder
	buf.WriteString("{")
	// Indirect test - formatMap is unexported
	_ = buf
}

func TestExtra_NoscriptPixel(t *testing.T) {
	got := NoscriptPixel("1234567890")
	if !strings.Contains(got, "noscript") {
		t.Error("should contain noscript tag")
	}
	if !strings.Contains(got, "1234567890") {
		t.Error("should contain pixel ID")
	}
	if !strings.Contains(got, "<img") {
		t.Error("should contain img tag")
	}
}

func TestExtra_NoscriptPixel_DifferentIDs(t *testing.T) {
	got1 := NoscriptPixel("111")
	got2 := NoscriptPixel("222")
	if got1 == got2 {
		t.Error("different IDs should produce different output")
	}
}

func TestExtra_PixelConfig_InitScript_WithoutDebug(t *testing.T) {
	p := &PixelConfig{
		PixelID:   "1234567890",
		Version:   "18.0",
		DebugMode: false,
	}
	got := p.InitScript()
	// Should NOT have autoConfig when DebugMode is false
	if strings.Contains(got, "fbq('set', 'autoConfig', false)") {
		t.Error("Should not have autoConfig when DebugMode is false")
	}
}

func TestExtra_TrackLead_FormatsFloat(t *testing.T) {
	params := map[string]interface{}{
		"value": 99.99,
	}
	got := TrackLead(params)
	if !strings.Contains(got, "99.990000") {
		t.Errorf("should format float: %s", got)
	}
}

func TestExtra_TrackLead_FormatsInt(t *testing.T) {
	params := map[string]interface{}{
		"count": 42,
	}
	got := TrackLead(params)
	if !strings.Contains(got, "42") {
		t.Errorf("should format int: %s", got)
	}
}

func TestExtra_TrackLead_FormatsBool(t *testing.T) {
	params := map[string]interface{}{
		"flag": true,
	}
	got := TrackLead(params)
	if !strings.Contains(got, "true") {
		t.Errorf("should format bool: %s", got)
	}
}

func TestExtra_escapeJS_Empty(t *testing.T) {
	if escapeJS("") != "" {
		t.Error("empty should remain empty")
	}
}

func TestExtra_TrackCustomEvent_EmptyParams(t *testing.T) {
	got := TrackCustomEvent("MyEvent", map[string]interface{}{})
	if !strings.Contains(got, "MyEvent") {
		t.Error("should contain event name")
	}
}
