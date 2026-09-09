// Tests for notification-service preferences.
package preferences

import (
	"testing"
	"time"
)

func TestExtra_NewResolver(t *testing.T) {
	r := NewResolver(nil)
	if r == nil {
		t.Fatal("nil resolver")
	}
}

func TestExtra_NewResolver_WithPrefs(t *testing.T) {
	prefs := []Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: true},
	}
	r := NewResolver(prefs)
	if len(r.prefs) != 1 {
		t.Errorf("expected 1 pref: got %d", len(r.prefs))
	}
}

func TestExtra_Resolve_HighPriority(t *testing.T) {
	r := NewResolver(nil)
	chans := r.Resolve("high", "alert", "u1")
	if len(chans) != 3 {
		t.Errorf("expected 3 channels for high: got %v", chans)
	}
}

func TestExtra_Resolve_NormalPriority(t *testing.T) {
	r := NewResolver(nil)
	chans := r.Resolve("normal", "info", "u1")
	if len(chans) != 2 {
		t.Errorf("expected 2 channels for normal: got %v", chans)
	}
}

func TestExtra_Resolve_LowPriority(t *testing.T) {
	r := NewResolver(nil)
	chans := r.Resolve("low", "info", "u1")
	if len(chans) != 1 {
		t.Errorf("expected 1 channel for low: got %v", chans)
	}
}

func TestExtra_Resolve_UnknownPriority(t *testing.T) {
	r := NewResolver(nil)
	chans := r.Resolve("unknown", "info", "u1")
	// Falls back to normal
	if len(chans) != 2 {
		t.Errorf("expected fallback to normal (2 channels): got %v", chans)
	}
}

func TestExtra_Resolve_DisabledChannel(t *testing.T) {
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: false},
	})
	chans := r.Resolve("normal", "info", "u1")
	for _, c := range chans {
		if c == "email" {
			t.Error("email should be disabled")
		}
	}
}

func TestExtra_Resolve_TypeSpecific(t *testing.T) {
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "alert", Channel: "email", Enabled: true},
	})
	// alert should get email; other types should not
	alertChans := r.Resolve("high", "alert", "u1")
	hasEmail := false
	for _, c := range alertChans {
		if c == "email" {
			hasEmail = true
		}
	}
	if !hasEmail {
		t.Error("alert should include email")
	}
}

func TestExtra_Resolve_DifferentUser(t *testing.T) {
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: false},
	})
	// u2 should not be affected by u1's preference
	chans := r.Resolve("normal", "info", "u2")
	hasEmail := false
	for _, c := range chans {
		if c == "email" {
			hasEmail = true
		}
	}
	if !hasEmail {
		t.Error("u2 should still get email")
	}
}

func TestExtra_Resolve_QuietHours_Daytime(t *testing.T) {
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: true},
	})
	r.now = func() time.Time {
		return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	}
	chans := r.Resolve("high", "alert", "u1")
	// During daytime, all channels should be allowed
	if len(chans) != 3 {
		t.Errorf("expected 3 channels daytime: got %v", chans)
	}
}

func TestExtra_Resolve_QuietHours_Nighttime(t *testing.T) {
	qStart, qEnd := 22, 7
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: true, QuietStart: &qStart, QuietEnd: &qEnd},
	})
	r.now = func() time.Time {
		return time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	}
	chans := r.Resolve("high", "alert", "u1")
	// At night, only in_app should be allowed
	for _, c := range chans {
		if c != "in_app" {
			t.Errorf("nighttime should only have in_app, got %s", c)
		}
	}
}

func TestExtra_Resolve_QuietHours_DaytimeOvernight(t *testing.T) {
	qStart, qEnd := 22, 7
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: true, QuietStart: &qStart, QuietEnd: &qEnd},
	})
	r.now = func() time.Time {
		return time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	}
	// Daytime: should not be in quiet hours
	chans := r.Resolve("high", "alert", "u1")
	if len(chans) != 3 {
		t.Errorf("expected 3 channels: got %v", chans)
	}
}

func TestExtra_inQuietPeriod_Simple(t *testing.T) {
	r := &Resolver{}
	// 22:00-07:00 overnight
	if !r.inQuietPeriod(23, 22, 7) {
		t.Error("23 should be in 22-7")
	}
	if !r.inQuietPeriod(2, 22, 7) {
		t.Error("2 should be in 22-7")
	}
	if r.inQuietPeriod(10, 22, 7) {
		t.Error("10 should NOT be in 22-7")
	}
}

func TestExtra_inQuietPeriod_Daytime(t *testing.T) {
	r := &Resolver{}
	// 12:00-14:00 simple range
	if !r.inQuietPeriod(12, 12, 14) {
		t.Error("12 should be in 12-14")
	}
	if !r.inQuietPeriod(13, 12, 14) {
		t.Error("13 should be in 12-14")
	}
	if r.inQuietPeriod(15, 12, 14) {
		t.Error("15 should NOT be in 12-14")
	}
}

func TestExtra_ParseQuietHours_Empty(t *testing.T) {
	s, e := ParseQuietHours(nil)
	if s != nil || e != nil {
		t.Error("empty should return nil")
	}
}

func TestExtra_ParseQuietHours_Short(t *testing.T) {
	s, e := ParseQuietHours([]int{8})
	if s != nil || e != nil {
		t.Error("single element should return nil")
	}
}

func TestExtra_ParseQuietHours_Valid(t *testing.T) {
	s, e := ParseQuietHours([]int{8, 17})
	if s == nil || e == nil {
		t.Fatal("nil")
	}
	if *s != 8 || *e != 17 {
		t.Errorf("got %d-%d", *s, *e)
	}
}

func TestExtra_ParseQuietHours_Invalid(t *testing.T) {
	// Out of range
	s, e := ParseQuietHours([]int{25, 30})
	if s != nil || e != nil {
		t.Error("invalid hours should return nil")
	}
}

func TestExtra_Merge_Add(t *testing.T) {
	r := NewResolver(nil)
	r.Merge([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: true},
	})
	if len(r.prefs) != 1 {
		t.Errorf("expected 1 pref: got %d", len(r.prefs))
	}
}

func TestExtra_Merge_Update(t *testing.T) {
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: true},
	})
	r.Merge([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: false},
	})
	if len(r.prefs) != 1 {
		t.Errorf("expected 1 pref after update: got %d", len(r.prefs))
	}
	for _, p := range r.prefs {
		if p.Enabled {
			t.Error("should be updated to disabled")
		}
	}
}

func TestExtra_Merge_Multiple(t *testing.T) {
	r := NewResolver(nil)
	r.Merge([]Preference{
		{UserID: "u1", NotifType: "*", Channel: "email", Enabled: true},
		{UserID: "u1", NotifType: "*", Channel: "push", Enabled: true},
	})
	if len(r.prefs) != 2 {
		t.Errorf("expected 2 prefs: got %d", len(r.prefs))
	}
}

func TestExtra_DefaultPriorityChannels(t *testing.T) {
	if len(DefaultPriorityChannels["high"]) != 3 {
		t.Error("high should have 3 channels")
	}
	if len(DefaultPriorityChannels["normal"]) != 2 {
		t.Error("normal should have 2 channels")
	}
	if len(DefaultPriorityChannels["low"]) != 1 {
		t.Error("low should have 1 channel")
	}
}

func TestExtra_Resolve_NoDefaults(t *testing.T) {
	r := NewResolver(nil)
	// Empty priority → fallback to "normal"
	chans := r.Resolve("", "info", "u1")
	if len(chans) != 2 {
		t.Errorf("expected 2: got %v", chans)
	}
}

func TestExtra_Resolve_TypeMismatch(t *testing.T) {
	// Preference for type=alert only, type=info should not be affected
	r := NewResolver([]Preference{
		{UserID: "u1", NotifType: "alert", Channel: "email", Enabled: false},
	})
	chans := r.Resolve("high", "info", "u1")
	// info should still get email (default)
	hasEmail := false
	for _, c := range chans {
		if c == "email" {
			hasEmail = true
		}
	}
	if !hasEmail {
		t.Error("info should get email (type-specific pref doesn't apply)")
	}
}
