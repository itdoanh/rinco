// Tests for timex package (timex.go).
package timex

import (
	"testing"
	"time"
)

func TestExtra2_RFC3339Nano_Value(t *testing.T) {
	if RFC3339Nano == "" {
		t.Error("RFC3339Nano should not be empty")
	}
}

func TestExtra2_CommonLocation_Valid(t *testing.T) {
	loc := CommonLocation("America/New_York")
	if loc == nil {
		t.Error("CommonLocation should not return nil")
	}
}

func TestExtra2_CommonLocation_Invalid(t *testing.T) {
	loc := CommonLocation("Invalid/Timezone")
	if loc != time.UTC {
		t.Error("Invalid timezone should return UTC")
	}
}

func TestExtra2_NowUTC(t *testing.T) {
	before := time.Now().UTC()
	now := NowUTC()
	after := time.Now().UTC()
	if now.Before(before) || now.After(after) {
		t.Error("NowUTC should be between before and after")
	}
}

func TestExtra2_FormatRFC3339Nano(t *testing.T) {
	tm := time.Date(2024, 1, 15, 12, 30, 45, 123456789, time.UTC)
	got := FormatRFC3339Nano(tm)
	if got == "" {
		t.Error("FormatRFC3339Nano should not return empty")
	}
	if got[:4] != "2024" {
		t.Errorf("Expected 2024, got %s", got[:4])
	}
}

func TestExtra2_ParseRFC3339Nano(t *testing.T) {
	input := "2024-01-15T12:30:45Z"
	tm, err := ParseRFC3339Nano(input)
	if err != nil {
		t.Errorf("ParseRFC3339Nano error: %v", err)
	}
	if tm.Year() != 2024 || tm.Month() != time.January || tm.Day() != 15 {
		t.Error("Date mismatch")
	}
}

func TestExtra2_ParseRFC3339Nano_Invalid(t *testing.T) {
	_, err := ParseRFC3339Nano("invalid")
	if err == nil {
		t.Error("Expected error for invalid input")
	}
}

func TestExtra2_StartOfDay(t *testing.T) {
	tm := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	start := StartOfDay(tm)
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("StartOfDay should be midnight: %v", start)
	}
}

func TestExtra2_EndOfDay(t *testing.T) {
	tm := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	end := EndOfDay(tm)
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("EndOfDay should be 23:59:59: %v", end)
	}
}

func TestExtra2_StartOfWeek(t *testing.T) {
	// Wednesday Jan 17, 2024
	tm := time.Date(2024, 1, 17, 12, 0, 0, 0, time.UTC)
	start := StartOfWeek(tm)
	// Should be Monday Jan 15, 2024
	if start.Weekday() != time.Monday {
		t.Errorf("StartOfWeek should be Monday, got %v", start.Weekday())
	}
}

func TestExtra2_StartOfWeek_Sunday(t *testing.T) {
	// Sunday Jan 21, 2024
	tm := time.Date(2024, 1, 21, 12, 0, 0, 0, time.UTC)
	start := StartOfWeek(tm)
	// Should be Monday Jan 15, 2024 (same week)
	if start.Weekday() != time.Monday {
		t.Errorf("StartOfWeek should be Monday, got %v", start.Weekday())
	}
}

func TestExtra2_EndOfWeek(t *testing.T) {
	tm := time.Date(2024, 1, 17, 12, 0, 0, 0, time.UTC)
	end := EndOfWeek(tm)
	if end.Weekday() != time.Sunday {
		t.Errorf("EndOfWeek should be Sunday, got %v", end.Weekday())
	}
}

func TestExtra2_StartOfMonth(t *testing.T) {
	tm := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	start := StartOfMonth(tm)
	if start.Day() != 1 {
		t.Errorf("StartOfMonth should be day 1, got %d", start.Day())
	}
}

func TestExtra2_EndOfMonth(t *testing.T) {
	// January 31
	tm := time.Date(2024, 1, 31, 12, 0, 0, 0, time.UTC)
	end := EndOfMonth(tm)
	if end.Day() != 31 {
		t.Errorf("EndOfMonth for Jan should be 31, got %d", end.Day())
	}
}

func TestExtra2_EndOfMonth_February(t *testing.T) {
	// Feb 15
	tm := time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC)
	end := EndOfMonth(tm)
	// 2024 is leap year, so Feb has 29 days
	if end.Day() != 29 {
		t.Errorf("EndOfMonth for Feb 2024 should be 29, got %d", end.Day())
	}
}

func TestExtra2_ISODate(t *testing.T) {
	tm := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)
	got := ISODate(tm)
	if got != "2024-01-05" {
		t.Errorf("ISODate: got %s", got)
	}
}

func TestExtra2_ISODateTime(t *testing.T) {
	tm := time.Date(2024, 1, 5, 14, 5, 30, 0, time.UTC)
	got := ISODateTime(tm)
	if got != "2024-01-05T14:05:30" {
		t.Errorf("ISODateTime: got %s", got)
	}
}

func TestExtra2_ISOWeekYear(t *testing.T) {
	// Jan 1, 2024 is in week 1 of 2024
	tm := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	year := ISOWeekYear(tm)
	if year != 2024 {
		t.Errorf("ISOWeekYear: got %d", year)
	}
}

func TestExtra2_ISOWeek(t *testing.T) {
	// Jan 1, 2024 is week 1
	tm := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	week := ISOWeek(tm)
	if week != 1 {
		t.Errorf("ISOWeek: got %d", week)
	}
}

func TestExtra2_ISODayOfWeek(t *testing.T) {
	// Monday
	monday := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) // Jan 15 is Monday
	if ISODayOfWeek(monday) != 1 {
		t.Errorf("Monday should be 1, got %d", ISODayOfWeek(monday))
	}
	
	// Sunday
	sunday := time.Date(2024, 1, 21, 12, 0, 0, 0, time.UTC)
	if ISODayOfWeek(sunday) != 7 {
		t.Errorf("Sunday should be 7, got %d", ISODayOfWeek(sunday))
	}
}

func TestExtra2_DurationSeconds(t *testing.T) {
	d := 5 * time.Second
	if DurationSeconds(d) != 5.0 {
		t.Errorf("DurationSeconds: got %f", DurationSeconds(d))
	}
}

func TestExtra2_SecondsDuration(t *testing.T) {
	d := SecondsDuration(5.0)
	if d != 5*time.Second {
		t.Errorf("SecondsDuration: got %v", d)
	}
}

func TestExtra2_MustParseDuration(t *testing.T) {
	d := MustParseDuration("5s")
	if d != 5*time.Second {
		t.Errorf("MustParseDuration: got %v", d)
	}
}

func TestExtra2_HumanizeDuration_Zero(t *testing.T) {
	got := HumanizeDuration(0)
	if got != "0s" {
		t.Errorf("HumanizeDuration(0): got %s", got)
	}
}

func TestExtra2_HumanizeDuration_Seconds(t *testing.T) {
	got := HumanizeDuration(30 * time.Second)
	if got != "30s" {
		t.Errorf("HumanizeDuration(30s): got %s", got)
	}
}

func TestExtra2_HumanizeDuration_Minutes(t *testing.T) {
	got := HumanizeDuration(5 * time.Minute)
	if got != "5m" {
		t.Errorf("HumanizeDuration(5m): got %s", got)
	}
}

func TestExtra2_HumanizeDuration_Hours(t *testing.T) {
	got := HumanizeDuration(3 * time.Hour)
	if got != "3h" {
		t.Errorf("HumanizeDuration(3h): got %s", got)
	}
}

func TestExtra2_HumanizeDuration_Days(t *testing.T) {
	got := HumanizeDuration(2 * 24 * time.Hour)
	if got != "2d" {
		t.Errorf("HumanizeDuration(2d): got %s", got)
	}
}

func TestExtra2_HumanizeDuration_Weeks(t *testing.T) {
	got := HumanizeDuration(2 * 7 * 24 * time.Hour)
	if got != "2w" {
		t.Errorf("HumanizeDuration(2w): got %s", got)
	}
}

func TestExtra2_HumanizeDuration_Months(t *testing.T) {
	// 45 days = 1 month + 15 days
	got := HumanizeDuration(45 * 24 * time.Hour)
	if got == "" {
		t.Error("HumanizeDuration should not be empty for 45 days")
	}
	// Should contain "1mo"
	if len(got) < 3 {
		t.Errorf("HumanizeDuration(45d) too short: %s", got)
	}
}

func TestExtra2_HumanizeDuration_Negative(t *testing.T) {
	got := HumanizeDuration(-30 * time.Second)
	if got != "-30s" {
		t.Errorf("HumanizeDuration(-30s): got %s", got)
	}
}

func TestExtra2_HumanizeDuration_Complex(t *testing.T) {
	// 1 year, 2 months, 3 days, 4 hours
	got := HumanizeDuration(1*365*24*time.Hour + 2*30*24*time.Hour + 3*24*time.Hour + 4*time.Hour)
	if got == "" {
		t.Error("Complex duration should not be empty")
	}
}

func TestExtra2_ParseRange_Valid(t *testing.T) {
	since, until, err := ParseRange("2024-01-01T00:00:00Z..2024-12-31T23:59:59Z")
	if err != nil {
		t.Errorf("ParseRange error: %v", err)
	}
	if since.IsZero() {
		t.Error("since should not be zero")
	}
	if until.IsZero() {
		t.Error("until should not be zero")
	}
}

func TestExtra2_ParseRange_SinceOnly(t *testing.T) {
	since, until, err := ParseRange("2024-01-01T00:00:00Z..")
	if err != nil {
		t.Errorf("ParseRange error: %v", err)
	}
	if since.IsZero() {
		t.Error("since should not be zero")
	}
	if !until.IsZero() {
		t.Error("until should be zero for since-only range")
	}
}

func TestExtra2_ParseRange_Empty(t *testing.T) {
	_, _, err := ParseRange("")
	if err == nil {
		t.Error("Expected error for empty range")
	}
}

func TestExtra2_QuoteISOWeek(t *testing.T) {
	tm := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	got := QuoteISOWeek(tm)
	if got == "" {
		t.Error("QuoteISOWeek should not return empty")
	}
	if len(got) < 8 {
		t.Errorf("QuoteISOWeek too short: %s", got)
	}
}

func TestExtra2_AddBusinessDays(t *testing.T) {
	// Friday Jan 19, 2024
	friday := time.Date(2024, 1, 19, 12, 0, 0, 0, time.UTC)
	
	// Add 1 business day -> Monday Jan 22, 2024
	monday := AddBusinessDays(friday, 1)
	if monday.Weekday() != time.Monday {
		t.Errorf("AddBusinessDays(1) from Friday should be Monday, got %v", monday.Weekday())
	}
}

func TestExtra2_AddBusinessDays_Zero(t *testing.T) {
	friday := time.Date(2024, 1, 19, 12, 0, 0, 0, time.UTC)
	result := AddBusinessDays(friday, 0)
	if !result.Equal(friday) {
		t.Error("AddBusinessDays(0) should return same day")
	}
}

func TestExtra2_AddBusinessDays_Multiple(t *testing.T) {
	// Friday Jan 19, 2024
	friday := time.Date(2024, 1, 19, 12, 0, 0, 0, time.UTC)
	
	// Add 3 business days -> Wednesday Jan 24, 2024
	result := AddBusinessDays(friday, 3)
	if result.Weekday() != time.Wednesday {
		t.Errorf("AddBusinessDays(3) from Friday should be Wednesday, got %v", result.Weekday())
	}
}

func TestExtra2_AddBusinessDays_Negative(t *testing.T) {
	// Monday Jan 22, 2024
	monday := time.Date(2024, 1, 22, 12, 0, 0, 0, time.UTC)
	
	// Subtract 1 business day -> Friday Jan 19, 2024
	friday := AddBusinessDays(monday, -1)
	if friday.Weekday() != time.Friday {
		t.Errorf("AddBusinessDays(-1) from Monday should be Friday, got %v", friday.Weekday())
	}
}

func TestExtra2_twoDigit(t *testing.T) {
	if twoDigit(1) != "01" {
		t.Errorf("twoDigit(1): got %s", twoDigit(1))
	}
	if twoDigit(10) != "10" {
		t.Errorf("twoDigit(10): got %s", twoDigit(10))
	}
	if twoDigit(0) != "00" {
		t.Errorf("twoDigit(0): got %s", twoDigit(0))
	}
}

func TestExtra2_AgeSeconds(t *testing.T) {
	now := time.Now()
	age := AgeSeconds(now)
	if age < 0 {
		t.Errorf("AgeSeconds for now should be >= 0, got %f", age)
	}
}
