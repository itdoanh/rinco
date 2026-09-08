// Tests for analytics-service models.
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEvent_JSON(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	now := time.Now()
	e := Event{
		ID:        id,
		TenantID:  tenantID,
		EventType: "page_view",
		Source:    "organic",
		Value:     1.5,
		Props:     map[string]string{"key": "value"},
		Timestamp: now,
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
		t.Errorf("ID: got %v", got.ID)
	}
	if got.TenantID != tenantID {
		t.Errorf("TenantID: got %v", got.TenantID)
	}
	if got.EventType != "page_view" {
		t.Errorf("EventType: got %s", got.EventType)
	}
	if got.Value != 1.5 {
		t.Errorf("Value: got %f", got.Value)
	}
	if got.Props["key"] != "value" {
		t.Errorf("Props: got %v", got.Props)
	}
}

func TestEvent_OmitEmptyFields(t *testing.T) {
	e := Event{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		EventType: "click",
		Timestamp: time.Now(),
	}
	b, _ := json.Marshal(e)
	// Should omit empty optional fields.
	s := string(b)
	if !contains(s, "event_type") {
		t.Error("event_type should be present")
	}
	// user_id is omitempty; should be absent when empty.
	if contains(s, "user_id") {
		t.Error("user_id should be omitted when empty")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestAggregation_JSON(t *testing.T) {
	a := Aggregation{
		Metric:      "page_views",
		Bucket:      time.Now(),
		Granularity: "hour",
		Value:       100.5,
		Count:       50,
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var got Aggregation
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Metric != a.Metric {
		t.Errorf("Metric: got %s", got.Metric)
	}
	if got.Count != a.Count {
		t.Errorf("Count: got %d", got.Count)
	}
}

func TestFunnelStep_JSON(t *testing.T) {
	f := FunnelStep{
		Name:       "View",
		Order:      1,
		Users:      100,
		Conversion: 100.0,
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var got FunnelStep
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "View" {
		t.Errorf("Name: got %s", got.Name)
	}
	if got.Users != 100 {
		t.Errorf("Users: got %d", got.Users)
	}
}

func TestDashboardSummary_JSON(t *testing.T) {
	d := DashboardSummary{
		TenantID:         uuid.New(),
		TotalPageViews:   1000,
		UniqueVisitors:   500,
		TotalConversions: 50,
		ConversionRate:   10.0,
		TopSources:       []SourceStat{{Source: "fb_ad", Visitors: 100}},
		TopPages:         []PageStat{{URL: "/home", PageViews: 500}},
		Trend:            []TrendPoint{{Timestamp: time.Now(), Value: 100, Count: 50}},
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var got DashboardSummary
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.TotalPageViews != 1000 {
		t.Errorf("TotalPageViews: got %d", got.TotalPageViews)
	}
	if len(got.TopSources) != 1 {
		t.Errorf("TopSources: got %d", len(got.TopSources))
	}
	if len(got.TopPages) != 1 {
		t.Errorf("TopPages: got %d", len(got.TopPages))
	}
}

func TestSourceStat_JSON(t *testing.T) {
	s := SourceStat{
		Source:      "fb_ad",
		Visitors:    100,
		Conversions: 10,
		Revenue:     99.99,
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got SourceStat
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Source != "fb_ad" {
		t.Errorf("Source: got %s", got.Source)
	}
	if got.Revenue != 99.99 {
		t.Errorf("Revenue: got %f", got.Revenue)
	}
}

func TestPageStat_JSON(t *testing.T) {
	p := PageStat{
		URL:        "/home",
		PageViews:  500,
		Visitors:   300,
		AvgTimeSec: 45.5,
		BounceRate: 30.0,
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var got PageStat
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.URL != "/home" {
		t.Errorf("URL: got %s", got.URL)
	}
	if got.AvgTimeSec != 45.5 {
		t.Errorf("AvgTimeSec: got %f", got.AvgTimeSec)
	}
}

func TestTrendPoint_JSON(t *testing.T) {
	tp := TrendPoint{
		Timestamp: time.Now(),
		Value:     100.5,
		Count:     50,
	}
	b, err := json.Marshal(tp)
	if err != nil {
		t.Fatal(err)
	}
	var got TrendPoint
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 50 {
		t.Errorf("Count: got %d", got.Count)
	}
	if got.Value != 100.5 {
		t.Errorf("Value: got %f", got.Value)
	}
}

func TestCohortRow_JSON(t *testing.T) {
	c := CohortRow{
		CohortDate:     time.Now(),
		Size:           100,
		RetentionByDay: map[int]float64{0: 100.0, 1: 50.0, 7: 25.0},
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
		t.Errorf("Size: got %d", got.Size)
	}
	if got.RetentionByDay[1] != 50.0 {
		t.Errorf("RetentionByDay[1]: got %f", got.RetentionByDay[1])
	}
}
