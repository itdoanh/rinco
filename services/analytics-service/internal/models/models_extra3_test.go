// Extra tests for analytics-service models.
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEvent_JSONRoundTrip(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	e := Event{
		ID:        id,
		TenantID:  tenantID,
		UserID:    "u1",
		EventType: "page_view",
		Source:    "fb_ad",
		URL:       "https://example.com",
		Props:     map[string]string{"plan": "pro"},
		Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got Event
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != id {
		t.Error("id mismatch")
	}
	if got.UserID != "u1" {
		t.Error("user mismatch")
	}
	if got.Props["plan"] != "pro" {
		t.Error("props mismatch")
	}
}

func TestEvent_DeviceTypes(t *testing.T) {
	devices := []string{"mobile", "desktop", "tablet", ""}
	for _, d := range devices {
		e := Event{Device: d}
		if e.Device != d {
			t.Errorf("device: %s", e.Device)
		}
	}
}

func TestAggregation_Granularities(t *testing.T) {
	grans := []string{"minute", "hour", "day", "week", "month"}
	for _, g := range grans {
		a := Aggregation{Granularity: g}
		if a.Granularity != g {
			t.Errorf("gran: %s", a.Granularity)
		}
	}
}

func TestAggregation_JSONExtra(t *testing.T) {
	a := Aggregation{
		Metric:      "page_views",
		Bucket:      time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		Granularity: "hour",
		Value:       100.5,
		Count:       200,
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var got Aggregation
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Metric != "page_views" {
		t.Error("metric")
	}
	if got.Count != 200 {
		t.Error("count")
	}
}

func TestFunnelStep_Default(t *testing.T) {
	f := FunnelStep{Name: "view"}
	if f.Order != 0 {
		t.Error("default order")
	}
	if f.Conversion != 0 {
		t.Error("default conversion")
	}
}

func TestFunnelStep_Conversion(t *testing.T) {
	f := FunnelStep{Name: "buy", Order: 3, Users: 50, Conversion: 0.25}
	if f.Conversion != 0.25 {
		t.Errorf("conv: %f", f.Conversion)
	}
}

func TestDashboardSummary_Fields(t *testing.T) {
	d := DashboardSummary{
		TenantID:         uuid.New(),
		TotalPageViews:   1000,
		UniqueVisitors:   500,
		TotalConversions: 50,
		ConversionRate:   0.10,
		BounceRate:       0.30,
	}
	if d.TotalPageViews != 1000 {
		t.Errorf("pv: %d", d.TotalPageViews)
	}
	if d.BounceRate != 0.30 {
		t.Errorf("bounce: %f", d.BounceRate)
	}
}

func TestSourceStat_Fields(t *testing.T) {
	s := SourceStat{Source: "fb", Visitors: 100, Conversions: 5, Revenue: 500.0}
	if s.Source != "fb" {
		t.Error("source")
	}
	if s.Revenue != 500.0 {
		t.Errorf("rev: %f", s.Revenue)
	}
}

func TestPageStat_Fields(t *testing.T) {
	p := PageStat{URL: "/home", PageViews: 100, Visitors: 80, AvgTimeSec: 30.5, BounceRate: 0.4}
	if p.AvgTimeSec != 30.5 {
		t.Errorf("time: %f", p.AvgTimeSec)
	}
}

func TestTrendPoint_Fields(t *testing.T) {
	tp := TrendPoint{Timestamp: time.Now(), Value: 100, Count: 50}
	if tp.Value != 100 {
		t.Error("value")
	}
}

func TestCohortRow_JSONExtra(t *testing.T) {
	c := CohortRow{
		CohortDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Size:            100,
		RetentionByDay:  map[int]float64{1: 0.8, 7: 0.5, 30: 0.3},
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var got CohortRow
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Size != 100 {
		t.Error("size")
	}
	if got.RetentionByDay[1] != 0.8 {
		t.Error("retention")
	}
}

func TestEvent_EmptyProps(t *testing.T) {
	e := Event{}
	b, _ := json.Marshal(e)
	// Should be valid JSON
	var got map[string]interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
}

func TestAggregation_ZeroValue(t *testing.T) {
	a := Aggregation{}
	if a.Value != 0 {
		t.Error("zero value")
	}
}

func TestDashboardSummary_EmptyTrend(t *testing.T) {
	d := DashboardSummary{}
	if len(d.Trend) != 0 {
		t.Error("empty trend")
	}
	if len(d.TopSources) != 0 {
		t.Error("empty sources")
	}
}

func TestFunnelStep_NegativeConversion(t *testing.T) {
	f := FunnelStep{Conversion: -0.1}
	if f.Conversion != -0.1 {
		t.Error("negative should be preserved")
	}
}
