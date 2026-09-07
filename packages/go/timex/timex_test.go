package timex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNowUTCAndFormat(t *testing.T) {
	now := NowUTC()
	assert.Equal(t, time.UTC, now.Location())
	assert.Equal(t, time.RFC3339Nano, time.RFC3339Nano) // sanity

	parsed, err := ParseRFC3339Nano(FormatRFC3339Nano(now))
	require.NoError(t, err)
	assert.WithinDuration(t, now, parsed, time.Microsecond)
}

func TestStartAndEndOfDay(t *testing.T) {
	t1 := time.Date(2026, 3, 15, 14, 30, 45, 123, time.UTC)
	start := StartOfDay(t1)
	assert.Equal(t, 0, start.Hour())
	assert.Equal(t, 0, start.Minute())
	assert.Equal(t, 0, start.Second())
	assert.Equal(t, 0, start.Nanosecond())
	assert.Equal(t, 2026, start.Year())
	assert.Equal(t, time.March, start.Month())
	assert.Equal(t, 15, start.Day())

	end := EndOfDay(t1)
	assert.Equal(t, 23, end.Hour())
	assert.Equal(t, 59, end.Minute())
	assert.Equal(t, 59, end.Second())
	assert.Equal(t, 999999999, end.Nanosecond())
}

func TestStartAndEndOfWeek(t *testing.T) {
	// Wednesday March 18, 2026
	w := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	start := StartOfWeek(w)
	assert.Equal(t, time.Monday, start.Weekday())
	assert.Equal(t, 16, start.Day())

	end := EndOfWeek(w)
	assert.Equal(t, time.Sunday, end.Weekday())
	assert.Equal(t, 22, end.Day())
	assert.Equal(t, 23, end.Hour())
}

func TestStartAndEndOfMonth(t *testing.T) {
	feb := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	start := StartOfMonth(feb)
	assert.Equal(t, 1, start.Day())

	end := EndOfMonth(feb)
	assert.Equal(t, 28, end.Day()) // 2026 not leap
	assert.Equal(t, 23, end.Hour())

	march := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	end2 := EndOfMonth(march)
	assert.Equal(t, 31, end2.Day())
}

func TestISOFormatting(t *testing.T) {
	d := time.Date(2026, 1, 5, 14, 30, 0, 0, time.UTC)
	assert.Equal(t, "2026-01-05", ISODate(d))
	assert.Equal(t, "2026-01-05T14:30:00", ISODateTime(d))
	assert.Equal(t, "2026-W02", QuoteISOWeek(d))

	sunday := time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC) // Sunday → previous ISO week
	assert.Equal(t, "2026-W01", QuoteISOWeek(sunday))
}

func TestISOWeekAndDayOfWeek(t *testing.T) {
	mon := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC) // Monday
	assert.Equal(t, 1, ISOWeek(mon))
	assert.Equal(t, 1, ISODayOfWeek(mon))

	sun := time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC) // Sunday
	assert.Equal(t, 7, ISODayOfWeek(sun))
}

func TestCommonLocation(t *testing.T) {
	utc := CommonLocation("UTC")
	assert.Equal(t, time.UTC, utc)

	bogus := CommonLocation("not-a-real-zone")
	assert.Equal(t, time.UTC, bogus)
}

func TestDurationConversions(t *testing.T) {
	d := 90 * time.Second
	assert.InDelta(t, 90.0, DurationSeconds(d), 0.001)

	back := SecondsDuration(90)
	assert.Equal(t, d, back)
}

func TestMustParseDuration(t *testing.T) {
	d := MustParseDuration("5m")
	assert.Equal(t, 5*time.Minute, d)
	assert.Panics(t, func() { MustParseDuration("garbage") })
}

func TestHumanizeDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m30s"},
		{2 * time.Hour, "2h"},
		{26 * time.Hour, "1d2h"},
		{7 * 24 * time.Hour, "1w"},
		{45 * 24 * time.Hour, "1mo15d"},
		{-90 * time.Second, "-1m30s"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, HumanizeDuration(c.d), "input=%v", c.d)
	}
}

func TestParseRange(t *testing.T) {
	s, u, err := ParseRange("2026-01-01T00:00:00Z..2026-02-01T00:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, 2026, s.Year())
	assert.Equal(t, time.February, u.Month())

	// Open ended ranges
	s, u, err = ParseRange("2026-01-01T00:00:00Z..")
	require.NoError(t, err)
	assert.True(t, s.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	assert.True(t, u.IsZero())

	_, _, err = ParseRange("")
	assert.Error(t, err)
}

func TestAddBusinessDays(t *testing.T) {
	// Start on Friday Mar 13 2026
	friday := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)
	mon := AddBusinessDays(friday, 1)
	assert.Equal(t, time.Monday, mon.Weekday())
	assert.Equal(t, 16, mon.Day())

	// 5 business days from Friday Mar 13 → next Friday Mar 20
	next := AddBusinessDays(friday, 5)
	assert.Equal(t, time.Friday, next.Weekday())
	assert.Equal(t, 20, next.Day())

	// 0 days → same day
	assert.Equal(t, friday, AddBusinessDays(friday, 0))

	// Negative -1 from Friday → previous Thursday
	prev := AddBusinessDays(friday, -1)
	assert.Equal(t, time.Thursday, prev.Weekday())
	assert.Equal(t, 12, prev.Day())
}

func TestAgeSeconds(t *testing.T) {
	past := time.Now().Add(-5 * time.Second)
	assert.InDelta(t, 5.0, AgeSeconds(past), 1.0)
}