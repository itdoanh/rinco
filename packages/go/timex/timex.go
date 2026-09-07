// Package timex provides time helpers focused on RINCO's common needs:
// RFC3339Nano formatting, ISO 8601 week-of-year arithmetic, time-zone aware
// date math, and range parsing.
package timex

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RFC3339Nano is the canonical timestamp format used across services and APIs.
const RFC3339Nano = time.RFC3339Nano

// CommonLocation loads a time.Location by IANA name, returning UTC on error.
func CommonLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// NowUTC returns time.Now() in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// FormatRFC3339Nano formats t in RFC3339Nano UTC form.
func FormatRFC3339Nano(t time.Time) string {
	return t.UTC().Format(RFC3339Nano)
}

// ParseRFC3339Nano parses an RFC3339Nano timestamp.
func ParseRFC3339Nano(s string) (time.Time, error) {
	return time.Parse(RFC3339Nano, s)
}

// StartOfDay returns midnight UTC of the given day.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// EndOfDay returns the last nanosecond of the day (23:59:59.999999999).
func EndOfDay(t time.Time) time.Time {
	return StartOfDay(t).Add(24*time.Hour - time.Nanosecond)
}

// StartOfWeek returns Monday 00:00 UTC of the ISO week containing t.
func StartOfWeek(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // ISO treats Sunday as 7
	}
	return StartOfDay(t).Add(-time.Duration(weekday-1) * 24 * time.Hour)
}

// EndOfWeek returns the last instant of the ISO week containing t.
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).Add(6 * 24 * time.Hour))
}

// StartOfMonth returns midnight UTC on the first day of the month.
func StartOfMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// EndOfMonth returns the last instant of the month.
func EndOfMonth(t time.Time) time.Time {
	return EndOfDay(StartOfMonth(t).AddDate(0, 1, -1))
}

// ISODate formats a date as YYYY-MM-DD.
func ISODate(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// ISODateTime formats a date-time without timezone suffix for compact APIs.
func ISODateTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05")
}

// ISOWeekYear returns the ISO 8601 week-numbering year (may differ from
// calendar year near boundaries).
func ISOWeekYear(t time.Time) int {
	y, _ := t.UTC().ISOWeek()
	return y
}

// ISOWeek returns the ISO 8601 week number (1..53).
func ISOWeek(t time.Time) int {
	_, w := t.UTC().ISOWeek()
	return w
}

// ISODayOfWeek returns the ISO day-of-week (Monday=1..Sunday=7).
func ISODayOfWeek(t time.Time) int {
	d := int(t.UTC().Weekday())
	if d == 0 {
		return 7
	}
	return d
}

// DurationSeconds converts a duration to a float64 seconds value.
func DurationSeconds(d time.Duration) float64 { return d.Seconds() }

// SecondsDuration converts seconds (float64) back to a duration.
func SecondsDuration(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

// MustParseDuration is like time.ParseDuration but panics on error.
func MustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Errorf("timex: parse duration %q: %w", s, err))
	}
	return d
}

// AgeSeconds returns how many seconds elapsed since t.
func AgeSeconds(t time.Time) float64 {
	return time.Since(t).Seconds()
}

// HumanizeDuration returns a compact human-readable duration like "3d 4h" or
// "-12s". Used in audit/log messages.
func HumanizeDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	neg := false
	if d < 0 {
		neg = true
		d = -d
	}
	var sb strings.Builder
	if neg {
		sb.WriteString("-")
	}
	switch {
	case d >= 365*24*time.Hour:
		years := int(d / (365 * 24 * time.Hour))
		fmt.Fprintf(&sb, "%dy", years)
		d -= time.Duration(years) * 365 * 24 * time.Hour
		fallthrough
	case d >= 30*24*time.Hour:
		months := int(d / (30 * 24 * time.Hour))
		fmt.Fprintf(&sb, "%dmo", months)
		d -= time.Duration(months) * 30 * 24 * time.Hour
		fallthrough
	case d >= 7*24*time.Hour:
		weeks := int(d / (7 * 24 * time.Hour))
		fmt.Fprintf(&sb, "%dw", weeks)
		d -= time.Duration(weeks) * 7 * 24 * time.Hour
		fallthrough
	case d >= 24*time.Hour:
		days := int(d / (24 * time.Hour))
		fmt.Fprintf(&sb, "%dd", days)
		d -= time.Duration(days) * 24 * time.Hour
		fallthrough
	case d >= time.Hour:
		hours := int(d / time.Hour)
		fmt.Fprintf(&sb, "%dh", hours)
		d -= time.Duration(hours) * time.Hour
		fallthrough
	case d >= time.Minute:
		mins := int(d / time.Minute)
		fmt.Fprintf(&sb, "%dm", mins)
		d -= time.Duration(mins) * time.Minute
		fallthrough
	case d >= time.Second:
		fmt.Fprintf(&sb, "%ds", int(d/time.Second))
	}
	return sb.String()
}

// ParseRange parses a "since..until" timestamp range. Either side may be
// empty; the format is RFC3339Nano.
func ParseRange(s string) (since, until time.Time, err error) {
	parts := strings.SplitN(s, "..", 2)
	if len(parts) == 0 || s == "" {
		return time.Time{}, time.Time{}, errors.New("timex: empty range")
	}
	if parts[0] != "" {
		since, err = ParseRFC3339Nano(parts[0])
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("timex: parse since: %w", err)
		}
	}
	if len(parts) == 2 && parts[1] != "" {
		until, err = ParseRFC3339Nano(parts[1])
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("timex: parse until: %w", err)
		}
	}
	return since, until, nil
}

// QuoteISOWeek returns the calendar week formatted as "YYYY-Www" (e.g. "2026-W01").
func QuoteISOWeek(t time.Time) string {
	y, w := t.UTC().ISOWeek()
	return strconv.Itoa(y) + "-W" + twoDigit(w)
}

func twoDigit(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// AddBusinessDays adds N business days (Mon-Fri) to t.
func AddBusinessDays(t time.Time, days int) time.Time {
	if days == 0 {
		return t
	}
	step := 1
	if days < 0 {
		step = -1
		days = -days
	}
	for i := 0; i < days; {
		t = t.AddDate(0, 0, step)
		wd := t.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			i++
		}
	}
	return t
}