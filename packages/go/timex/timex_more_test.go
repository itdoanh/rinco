package timex

import (
	"strings"
	"testing"
	"time"
)

func TestCommonLocationUTC(t *testing.T) {
	if CommonLocation("UTC") != time.UTC {
		t.Error("UTC should return UTC loc")
	}
}

func TestCommonLocationUnknown(t *testing.T) {
	// Unknown tz falls back to UTC, not crash
	got := CommonLocation("Not/AZone")
	if got != time.UTC {
		t.Errorf("expected UTC fallback, got %v", got)
	}
}

func TestCommonLocationValid(t *testing.T) {
	loc := CommonLocation("Asia/Ho_Chi_Minh")
	if loc == nil {
		t.Fatal("returned nil")
	}
	if loc.String() != "Asia/Ho_Chi_Minh" {
		t.Errorf("expected HCM, got %s", loc.String())
	}
}

func TestNowUTCIsUTC(t *testing.T) {
	now := NowUTC()
	if now.Location() != time.UTC {
		t.Errorf("location = %v, want UTC", now.Location())
	}
}

func TestFormatParseRFC3339Nano(t *testing.T) {
	original := time.Date(2026, 6, 15, 12, 30, 45, 123456789, time.UTC)
	formatted := FormatRFC3339Nano(original)
	if formatted == "" {
		t.Fatal("empty format")
	}
	parsed, err := ParseRFC3339Nano(formatted)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !parsed.Equal(original) {
		t.Errorf("roundtrip mismatch: %v vs %v", parsed, original)
	}
}

func TestParseRFC3339NanoInvalid(t *testing.T) {
	_, err := ParseRFC3339Nano("not-a-time")
	if err == nil {
		t.Error("expected error")
	}
}

func TestStartOfDay(t *testing.T) {
	day := time.Date(2026, 6, 15, 14, 30, 45, 999, time.UTC)
	start := StartOfDay(day)
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("not midnight: %v", start)
	}
	if start.Day() != 15 {
		t.Errorf("day = %d", start.Day())
	}
}

func TestEndOfDayAfterStart(t *testing.T) {
	day := time.Date(2026, 6, 15, 14, 30, 45, 999, time.UTC)
	end := EndOfDay(day)
	start := StartOfDay(day)
	if !end.After(start) {
		t.Error("end should be after start")
	}
	if end.Day() != 15 {
		t.Errorf("end day = %d, want 15", end.Day())
	}
}

func TestStartOfWeekMonday(t *testing.T) {
	// 2026-06-15 is Monday
	monday := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	start := StartOfWeek(monday)
	if start.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", start.Weekday())
	}
}

func TestStartOfWeekSunday(t *testing.T) {
	// 2026-06-21 is Sunday — should move to previous Monday
	sunday := time.Date(2026, 6, 21, 14, 30, 0, 0, time.UTC)
	start := StartOfWeek(sunday)
	if start.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", start.Weekday())
	}
	if start.Day() != 15 {
		t.Errorf("day = %d, want 15", start.Day())
	}
}

func TestEndOfWeek(t *testing.T) {
	monday := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	end := EndOfWeek(monday)
	if end.Weekday() != time.Sunday {
		t.Errorf("expected Sunday, got %v", end.Weekday())
	}
}

func TestStartOfMonth(t *testing.T) {
	mid := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	start := StartOfMonth(mid)
	if start.Day() != 1 {
		t.Errorf("day = %d, want 1", start.Day())
	}
	if start.Month() != time.June {
		t.Errorf("month = %v", start.Month())
	}
}

func TestEndOfMonthJune(t *testing.T) {
	mid := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	end := EndOfMonth(mid)
	if end.Day() != 30 {
		t.Errorf("June day = %d, want 30", end.Day())
	}
}

func TestEndOfMonthFebruaryNonLeap(t *testing.T) {
	mid := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	end := EndOfMonth(mid)
	if end.Day() != 28 {
		t.Errorf("Feb 2025 day = %d, want 28", end.Day())
	}
}

func TestEndOfMonthFebruaryLeap(t *testing.T) {
	mid := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
	end := EndOfMonth(mid)
	if end.Day() != 29 {
		t.Errorf("Feb 2024 day = %d, want 29", end.Day())
	}
}

func TestISODate(t *testing.T) {
	d := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	got := ISODate(d)
	if got != "2026-06-15" {
		t.Errorf("ISODate = %q", got)
	}
}

func TestISODateTime(t *testing.T) {
	d := time.Date(2026, 6, 15, 14, 30, 45, 0, time.UTC)
	got := ISODateTime(d)
	if !strings.HasPrefix(got, "2026-06-15T14:30:45") {
		t.Errorf("ISODateTime = %q", got)
	}
}

func TestISOWeekYear(t *testing.T) {
	d := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	if ISOWeekYear(d) != 2026 {
		t.Errorf("ISOWeekYear = %d", ISOWeekYear(d))
	}
}

func TestISOWeek(t *testing.T) {
	d := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	w := ISOWeek(d)
	if w < 1 || w > 53 {
		t.Errorf("ISO week out of range: %d", w)
	}
}

func TestISODayOfWeekMonday(t *testing.T) {
	d := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	if ISODayOfWeek(d) != 1 {
		t.Errorf("Monday = %d, want 1", ISODayOfWeek(d))
	}
}

func TestISODayOfWeekSunday(t *testing.T) {
	d := time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)
	if ISODayOfWeek(d) != 7 {
		t.Errorf("Sunday = %d, want 7", ISODayOfWeek(d))
	}
}

func TestDurationSeconds(t *testing.T) {
	if DurationSeconds(2 * time.Second) != 2.0 {
		t.Error("DurationSeconds mismatch")
	}
}

func TestSecondsDuration(t *testing.T) {
	d := SecondsDuration(3.5)
	if d != 3500*time.Millisecond {
		t.Errorf("SecondsDuration = %v", d)
	}
}

func TestMustParseDurationPanicBad(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	MustParseDuration("not-a-duration")
}

func TestHumanizeDurationZero(t *testing.T) {
	if got := HumanizeDuration(0); got != "0s" {
		t.Errorf("zero = %q, want '0s'", got)
	}
}

func TestHumanizeDurationSeconds(t *testing.T) {
	got := HumanizeDuration(45 * time.Second)
	if !strings.Contains(got, "s") {
		t.Errorf("seconds: %q", got)
	}
}

func TestHumanizeDurationMinutes(t *testing.T) {
	got := HumanizeDuration(5 * time.Minute)
	if !strings.Contains(got, "m") {
		t.Errorf("minutes: %q", got)
	}
}

func TestHumanizeDurationHours(t *testing.T) {
	got := HumanizeDuration(3 * time.Hour)
	if !strings.Contains(got, "h") {
		t.Errorf("hours: %q", got)
	}
}

func TestHumanizeDurationDays(t *testing.T) {
	got := HumanizeDuration(48 * time.Hour)
	if !strings.Contains(got, "d") {
		t.Errorf("days: %q", got)
	}
}

func TestHumanizeDurationNegative(t *testing.T) {
	got := HumanizeDuration(-2 * time.Hour)
	if !strings.HasPrefix(got, "-") {
		t.Errorf("negative: %q should start with -", got)
	}
}

func TestParseRangeEmpty(t *testing.T) {
	_, _, err := ParseRange("")
	if err == nil {
		t.Error("expected error for empty")
	}
}

func TestParseRangeOnlySince(t *testing.T) {
	since, until, err := ParseRange("2026-01-01T00:00:00Z..")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if since.IsZero() {
		t.Error("since zero")
	}
	if !until.IsZero() {
		t.Error("until not zero")
	}
}

func TestParseRangeBothSides(t *testing.T) {
	since, until, err := ParseRange("2026-01-01T00:00:00Z..2026-12-31T23:59:59Z")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if since.IsZero() || until.IsZero() {
		t.Error("both empty")
	}
	if until.Before(since) {
		t.Error("until before since")
	}
}

func TestParseRangeInvalidSince(t *testing.T) {
	_, _, err := ParseRange("invalid..2026-01-01T00:00:00Z")
	if err == nil {
		t.Error("expected error")
	}
}

func TestParseRangeInvalidUntil(t *testing.T) {
	_, _, err := ParseRange("2026-01-01T00:00:00Z..notvalid")
	if err == nil {
		t.Error("expected error")
	}
}

func TestQuoteISOWeek(t *testing.T) {
	d := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	got := QuoteISOWeek(d)
	if !strings.HasPrefix(got, "2026-W") {
		t.Errorf("QuoteISOWeek = %q", got)
	}
}

func TestAddBusinessDaysZero(t *testing.T) {
	d := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	got := AddBusinessDays(d, 0)
	if !got.Equal(d) {
		t.Error("zero days should not change")
	}
}

func TestAddBusinessDaysOne(t *testing.T) {
	// 2026-06-15 (Monday) + 1 business day = Tuesday
	d := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	got := AddBusinessDays(d, 1)
	if got.Weekday() != time.Tuesday {
		t.Errorf("got %v, expected Tuesday", got.Weekday())
	}
}

func TestAddBusinessDaysSkipWeekend(t *testing.T) {
	// 2026-06-19 Friday + 1 business day = Monday 06-22
	d := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	got := AddBusinessDays(d, 1)
	if got.Weekday() != time.Monday {
		t.Errorf("got %v, expected Monday", got.Weekday())
	}
	if got.Day() != 22 {
		t.Errorf("got day %d, want 22", got.Day())
	}
}

func TestAddBusinessDaysNegative(t *testing.T) {
	// 2026-06-15 (Monday) - 1 business day = previous Friday
	d := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	got := AddBusinessDays(d, -1)
	if got.Weekday() != time.Friday {
		t.Errorf("got %v, expected Friday", got.Weekday())
	}
}
