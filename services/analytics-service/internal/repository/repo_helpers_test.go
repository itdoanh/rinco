// Tests for analytics-service repository internal helpers.
package repository

import (
	"strings"
	"testing"
)

func TestFormatProps_Empty(t *testing.T) {
	got := formatProps(nil)
	if got != "{}" {
		t.Errorf("got %q", got)
	}
}

func TestFormatProps_EmptyMap(t *testing.T) {
	got := formatProps(map[string]string{})
	if got != "{}" {
		t.Errorf("got %q", got)
	}
}

func TestFormatProps_Single(t *testing.T) {
	got := formatProps(map[string]string{"k": "v"})
	if got != `{"k":"v"}` {
		t.Errorf("got %q", got)
	}
}

func TestFormatProps_Multiple(t *testing.T) {
	got := formatProps(map[string]string{"a": "1", "b": "2"})
	if !strings.Contains(got, `"a":"1"`) || !strings.Contains(got, `"b":"2"`) {
		t.Errorf("got %q", got)
	}
	if !strings.HasPrefix(got, "{") || !strings.HasSuffix(got, "}") {
		t.Errorf("got %q", got)
	}
}

func TestFormatProps_QuotesValue(t *testing.T) {
	got := formatProps(map[string]string{"k": `hello "world"`})
	// Value should be escaped
	if !strings.Contains(got, `\"world\"`) {
		t.Errorf("got %q", got)
	}
}

func TestMapGranularity_Minute(t *testing.T) {
	if mapGranularity("minute") != "minute" {
		t.Errorf("got %s", mapGranularity("minute"))
	}
}

func TestMapGranularity_Min(t *testing.T) {
	if mapGranularity("min") != "minute" {
		t.Errorf("got %s", mapGranularity("min"))
	}
}

func TestMapGranularity_Hour(t *testing.T) {
	if mapGranularity("hour") != "hour" {
		t.Errorf("got %s", mapGranularity("hour"))
	}
}

func TestMapGranularity_Hr(t *testing.T) {
	if mapGranularity("hr") != "hour" {
		t.Errorf("got %s", mapGranularity("hr"))
	}
}

func TestMapGranularity_Day(t *testing.T) {
	if mapGranularity("day") != "day" {
		t.Errorf("got %s", mapGranularity("day"))
	}
}

func TestMapGranularity_Week(t *testing.T) {
	if mapGranularity("week") != "week" {
		t.Errorf("got %s", mapGranularity("week"))
	}
}

func TestMapGranularity_Month(t *testing.T) {
	if mapGranularity("month") != "month" {
		t.Errorf("got %s", mapGranularity("month"))
	}
}

func TestMapGranularity_UnknownDefaultsDay(t *testing.T) {
	if mapGranularity("xyz") != "day" {
		t.Errorf("got %s", mapGranularity("xyz"))
	}
}

func TestMapGranularity_EmptyDefaultsDay(t *testing.T) {
	if mapGranularity("") != "day" {
		t.Errorf("got %s", mapGranularity(""))
	}
}

func TestMapGranularity_CaseInsensitive(t *testing.T) {
	if mapGranularity("DAY") != "day" {
		t.Errorf("got %s", mapGranularity("DAY"))
	}
	if mapGranularity("Hour") != "hour" {
		t.Errorf("got %s", mapGranularity("Hour"))
	}
}

func TestNew(t *testing.T) {
	r := New(nil)
	if r == nil {
		t.Fatal("nil")
	}
	if r.ch != nil {
		t.Errorf("expected nil")
	}
}
