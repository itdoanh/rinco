package models

import (
	"time"

	"github.com/google/uuid"
)

// Event represents a raw analytics event
type Event struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    string    `json:"user_id,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	EventType string    `json:"event_type"` // page_view, click, form_submit, conversion
	Source    string    `json:"source"`     // fb_ad, google, organic, direct
	Campaign  string    `json:"campaign,omitempty"`
	URL       string    `json:"url,omitempty"`
	Referrer  string    `json:"referrer,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Country   string    `json:"country,omitempty"`
	Device    string    `json:"device,omitempty"` // mobile, desktop, tablet
	Browser   string    `json:"browser,omitempty"`
	OS        string    `json:"os,omitempty"`
	Value     float64   `json:"value,omitempty"`
	Props     map[string]string `json:"props,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Aggregation represents a time-bucketed aggregation
type Aggregation struct {
	Metric      string    `json:"metric"`
	Bucket      time.Time `json:"bucket"`
	Granularity string    `json:"granularity"` // minute, hour, day, week, month
	Value       float64   `json:"value"`
	Count       int64     `json:"count"`
}

// FunnelStep represents a step in a conversion funnel
type FunnelStep struct {
	Name      string  `json:"name"`
	Order     int     `json:"order"`
	Users     int64   `json:"users"`
	Conversion float64 `json:"conversion"` // percentage from previous step
}

// DashboardSummary provides top-level metrics for the dashboard
type DashboardSummary struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	TotalPageViews   int64     `json:"total_page_views"`
	UniqueVisitors   int64     `json:"unique_visitors"`
	TotalConversions int64     `json:"total_conversions"`
	ConversionRate   float64   `json:"conversion_rate"`
	BounceRate       float64   `json:"bounce_rate"`
	AvgSessionSec    float64   `json:"avg_session_seconds"`
	TopSources       []SourceStat `json:"top_sources"`
	TopPages         []PageStat   `json:"top_pages"`
	Trend            []TrendPoint `json:"trend"`
}

// SourceStat represents traffic source statistics
type SourceStat struct {
	Source    string  `json:"source"`
	Visitors  int64   `json:"visitors"`
	Conversions int64 `json:"conversions"`
	Revenue   float64 `json:"revenue"`
}

// PageStat represents per-page statistics
type PageStat struct {
	URL        string  `json:"url"`
	PageViews  int64   `json:"page_views"`
	Visitors   int64   `json:"visitors"`
	AvgTimeSec float64 `json:"avg_time_seconds"`
	BounceRate float64 `json:"bounce_rate"`
}

// TrendPoint represents a time-series point
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Count     int64     `json:"count"`
}

// CohortRow represents a single row in cohort analysis
type CohortRow struct {
	CohortDate    time.Time `json:"cohort_date"`
	Size          int       `json:"size"`
	RetentionByDay map[int]float64 `json:"retention_by_day"` // day N -> retention rate
}
