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

// RealtimeStats captures in-flight live metrics for a tenant.
type RealtimeStats struct {
	TenantID   string    `json:"tenant_id"`
	Visitors   int64     `json:"visitors"`
	PageViews  int64     `json:"page_views"`
	Events     int64     `json:"events"`
	WindowFrom time.Time `json:"window_from"`
	WindowTo   time.Time `json:"window_to"`
}

// LeadReport summarises lead lifecycle metrics.
type LeadReport struct {
	TenantID      string       `json:"tenant_id"`
	TotalLeads    int64        `json:"total_leads"`
	NewLeads      int64        `json:"new_leads"`
	Qualified     int64        `json:"qualified"`
	Converted     int64        `json:"converted"`
	ConversionPct float64      `json:"conversion_pct"`
	BySource      []SourceStat `json:"by_source"`
}

// StageStat captures deals grouped by pipeline stage.
type StageStat struct {
	Stage string  `json:"stage"`
	Count int64   `json:"count"`
	Value float64 `json:"value"`
}

// DealReport is the deal pipeline rollup.
type DealReport struct {
	TenantID   string      `json:"tenant_id"`
	Open       int64       `json:"open"`
	Won        int64       `json:"won"`
	Lost       int64       `json:"lost"`
	TotalValue float64     `json:"total_value"`
	ByStage    []StageStat `json:"by_stage"`
}

// UserActivity captures one active user's metrics.
type UserActivity struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name,omitempty"`
	Events    int64  `json:"events"`
	LastSeen  time.Time `json:"last_seen"`
}

// UserActivityReport groups per-user engagement metrics.
type UserActivityReport struct {
	TenantID    string        `json:"tenant_id"`
	ActiveUsers int64         `json:"active_users"`
	TopUsers    []UserActivity `json:"top_users"`
}

// RevenueByMonth captures monthly revenue.
type RevenueByMonth struct {
	Month   string  `json:"month"`
	Revenue float64 `json:"revenue"`
	Deals   int64   `json:"deals"`
}

// RevenueReport groups revenue by period.
type RevenueReport struct {
	TenantID string           `json:"tenant_id"`
	Total    float64          `json:"total"`
	MRR      float64          `json:"mrr"`
	ARR      float64          `json:"arr"`
	ByMonth  []RevenueByMonth `json:"by_month"`
}
