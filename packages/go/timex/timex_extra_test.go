// Additional tests for timex package.
package timex

import (
	"strings"
	"testing"
	"time"
)

func TestExtra_CommonLocation(t *testing.T) {
	loc := CommonLocation("America/New_York")
	if loc == nil {
		t.Fatal("nil location")
	}
	// Invalid location should fall back to UTC
	loc = CommonLocation("Invalid/Location")
	if loc != time.UTC {
		t.Error("invalid should fallback to UTC")
	}
}

func TestExtra_NowUTC(t *testing.T) {
	now := NowUTC()
	if now.Location() != time.UTC {
		t.Error("NowUTC should be UTC")
	}
}

func TestExtra_FormatRFC3339Nano(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC)
	got := FormatRFC3339Nano(t1)
	if !strings.Contains(got, "2026-01-15") {
		t.Errorf("expected date in format: got %s", got)
	}
}

func TestExtra_ParseRFC3339Nano(t *testing.T) {
	s := "2026-01-15T10:30:45Z"
	got, err := ParseRFC3339Nano(s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got.Year() != 2026 {
		t.Errorf("year: got %d", got.Year())
	}
}

func TestExtra_StartOfDay(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 10, 30, 45, 123, time.UTC)
	start := StartOfDay(t1)
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("StartOfDay not midnight: %v", start)
	}
	if start.Day() != 15 {
		t.Errorf("day: got %d", start.Day())
	}
}

func TestExtra_EndOfDay(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 10, 30, 45, 123, time.UTC)
	end := EndOfDay(t1)
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("EndOfDay not 23:59:59: %v", end)
	}
}

func TestExtra_StartOfWeek(t *testing.T) {
	// Wednesday Jan 14, 2026
	wed := time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC)
	start := StartOfWeek(wed)
	// Should be Monday Jan 12
	if start.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", start.Weekday())
	}
	if start.Day() != 12 {
		t.Errorf("expected day 12, got %d", start.Day())
	}
}

func TestExtra_StartOfWeek_Sunday(t *testing.T) {
	// Sunday Jan 18, 2026
	sun := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	start := StartOfWeek(sun)
	// ISO week treats Sunday as day 7, so previous Monday is Jan 12
	if start.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", start.Weekday())
	}
}

func TestExtra_StartOfMonth(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC)
	start := StartOfMonth(t1)
	if start.Day() != 1 {
		t.Errorf("day: got %d", start.Day())
	}
	if start.Month() != time.January {
		t.Errorf("month: got %v", start.Month())
	}
}

func TestExtra_EndOfMonth_January(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC)
	end := EndOfMonth(t1)
	if end.Day() != 31 {
		t.Errorf("January has 31 days, got %d", end.Day())
	}
}

func TestExtra_EndOfMonth_February(t *testing.T) {
	t1 := time.Date(2026, 2, 15, 10, 30, 45, 0, time.UTC)
	end := EndOfMonth(t1)
	if end.Day() != 28 {
		t.Errorf("Feb 2026 has 28 days, got %d", end.Day())
	}
}

func TestExtra_ISODate(t *testing.T) {
	t1 := time.Date(2026, 1, 5, 10, 30, 45, 0, time.UTC)
	got := ISODate(t1)
	if got != "2026-01-05" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_ISODateTime(t *testing.T) {
	t1 := time.Date(2026, 1, 5, 10, 30, 45, 0, time.UTC)
	got := ISODateTime(t1)
	if got != "2026-01-05T10:30:45" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_ISOWeekYear(t *testing.T) {
	t1 := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	got := ISOWeekYear(t1)
	if got != 2026 {
		t.Errorf("got %d", got)
	}
}

func TestExtra_ISOWeek(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := ISOWeek(t1)
	if got < 1 || got > 53 {
		t.Errorf("ISO week out of range: %d", got)
	}
}

func TestExtra_ISODayOfWeek(t *testing.T) {
	// Monday = 1
	mon := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
	if got := ISODayOfWeek(mon); got != 1 {
		t.Errorf("Monday: got %d, want 1", got)
	}

	// Sunday = 7
	sun := time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC)
	if got := ISODayOfWeek(sun); got != 7 {
		t.Errorf("Sunday: got %d, want 7", got)
	}
}

func TestExtra_DurationSeconds(t *testing.T) {
	if got := DurationSeconds(2 * time.Second); got != 2.0 {
		t.Errorf("got %f", got)
	}
}

func TestExtra_SecondsDuration(t *testing.T) {
	if got := SecondsDuration(2.5); got != 2500*time.Millisecond {
		t.Errorf("got %v", got)
	}
}

func TestExtra_MustParseDuration(t *testing.T) {
	got := MustParseDuration("5m")
	if got != 5*time.Minute {
		t.Errorf("got %v", got)
	}
}

func TestExtra_AgeSeconds(t *testing.T) {
	past := time.Now().Add(-2 * time.Second)
	got := AgeSeconds(past)
	if got < 1.5 || got > 3.0 {
		t.Errorf("expected ~2s, got %f", got)
	}
}

func TestExtra_HumanizeDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{2 * time.Minute, "2m"},
		{1 * time.Hour, "1h"},
		{24 * time.Hour, "1d"},
		{48 * time.Hour, "2d"},
		{-5 * time.Second, "-5s"},
	}
	for _, tt := range tests {
		got := HumanizeDuration(tt.d)
		if got != tt.want {
			t.Errorf("HumanizeDuration(%v): got %s, want %s", tt.d, got, tt.want)
		}
	}
}

func TestExtra_ParseRange(t *testing.T) {
	since, until, err := ParseRange("2026-01-01T00:00:00Z..2026-01-31T23:59:59Z")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if since.IsZero() {
		t.Error("since should not be zero")
	}
	if until.IsZero() {
		t.Error("until should not be zero")
	}
}

func TestExtra_ParseRange_Empty(t *testing.T) {
	_, _, err := ParseRange("")
	if err == nil {
		t.Error("empty range should error")
	}
}

func TestExtra_ParseRange_InvalidSince(t *testing.T) {
	_, _, err := ParseRange("not-a-time..2026-01-31T23:59:59Z")
	if err == nil {
		t.Error("invalid since should error")
	}
}

func TestExtra_QuoteISOWeek(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := QuoteISOWeek(t1)
	if !strings.HasPrefix(got, "2026-W") {
		t.Errorf("expected 2026-W prefix: got %s", got)
	}
}

func TestExtra_AddBusinessDays_Zero(t *testing.T) {
	t1 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := AddBusinessDays(t1, 0)
	if !got.Equal(t1) {
		t.Errorf("zero days should return same time")
	}
}

func TestExtra_AddBusinessDays_Positive(t *testing.T) {
	// Monday Jan 12 + 5 business days = next Monday Jan 19
	mon := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
	got := AddBusinessDays(mon, 5)
	if got.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", got.Weekday())
	}
	if got.Day() != 19 {
		t.Errorf("expected day 19, got %d", got.Day())
	}
}

func TestExtra_AddBusinessDays_SkipsWeekend(t *testing.T) {
	// Friday Jan 16 + 1 business day = Monday Jan 19 (skip weekend)
	fri := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)
	got := AddBusinessDays(fri, 1)
	if got.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", got.Weekday())
	}
}

func TestExtra_TwoDigit(t *testing.T) {
	if got := twoDigit(5); got != "05" {
		t.Errorf("got %s", got)
	}
	if got := twoDigit(15); got != "15" {
		t.Errorf("got %s", got)
	}
}
