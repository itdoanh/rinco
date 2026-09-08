package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/analytics-service/internal/models"
)

// Repository handles ClickHouse analytics data
type Repository struct {
	ch clickhouse.Conn
}

func New(ch clickhouse.Conn) *Repository {
	return &Repository{ch: ch}
}

// TrackEvent ingests a single analytics event into ClickHouse
func (r *Repository) TrackEvent(ctx context.Context, ev *models.Event) error {
	const sql = `
		INSERT INTO analytics.events (
			id, tenant_id, user_id, session_id, event_type, source, campaign,
			url, referrer, user_agent, country, device, browser, os, value,
			props, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	propsJSON := ""
	if ev.Props != nil {
		// props handled as JSON string
		propsJSON = formatProps(ev.Props)
	}

	return r.ch.Exec(ctx, sql,
		ev.ID, ev.TenantID, ev.UserID, ev.SessionID, ev.EventType,
		ev.Source, ev.Campaign, ev.URL, ev.Referrer, ev.UserAgent,
		ev.Country, ev.Device, ev.Browser, ev.OS, ev.Value,
		propsJSON, ev.Timestamp,
	)
}

// TrackEventsBatch ingests multiple events in one batch
func (r *Repository) TrackEventsBatch(ctx context.Context, events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	const sql = `
		INSERT INTO analytics.events (
			id, tenant_id, user_id, session_id, event_type, source, campaign,
			url, referrer, user_agent, country, device, browser, os, value,
			props, timestamp
		) VALUES `
	vals := make([]string, 0, len(events))
	args := make([]any, 0, len(events)*17)
	for i, ev := range events {
		offset := i * 17
		vals = append(vals, fmt.Sprintf("(%s)",
			strings.Join(strings.Split(strings.Repeat("?", 17), ""), ",")))
		_ = offset
		args = append(args,
			ev.ID, ev.TenantID, ev.UserID, ev.SessionID, ev.EventType,
			ev.Source, ev.Campaign, ev.URL, ev.Referrer, ev.UserAgent,
			ev.Country, ev.Device, ev.Browser, ev.OS, ev.Value,
			formatProps(ev.Props), ev.Timestamp,
		)
	}
	return r.ch.Exec(ctx, sql+strings.Join(vals, ","), args...)
}

func formatProps(props map[string]string) string {
	if len(props) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(props))
	for k, v := range props {
		parts = append(parts, fmt.Sprintf("%q:%q", k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// GetPageViews returns page view count over time
func (r *Repository) GetPageViews(ctx context.Context, tenantID uuid.UUID, from, to time.Time, granularity string) ([]models.Aggregation, error) {
	col := mapGranularity(granularity)
	sql := fmt.Sprintf(`
		SELECT '%s' as metric, toStartOfInterval(timestamp, INTERVAL 1 %s) as bucket,
			sum(value) as val, count() as cnt
		FROM analytics.events
		WHERE tenant_id = ? AND event_type = 'page_view'
			AND timestamp BETWEEN ? AND ?
		GROUP BY bucket ORDER BY bucket`, col, col)

	var results []models.Aggregation
	rows, err := r.ch.Query(ctx, sql, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Aggregation
		if err := rows.Scan(&a.Metric, &a.Bucket, &a.Value, &a.Count); err != nil {
			return nil, err
		}
		a.Granularity = granularity
		results = append(results, a)
	}
	return results, rows.Err()
}

// GetUniqueVisitors returns distinct visitor count over time
func (r *Repository) GetUniqueVisitors(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (int64, error) {
	sql := `
		SELECT uniqCombined(user_id) FROM analytics.events
		WHERE tenant_id = ? AND timestamp BETWEEN ? AND ? AND user_id != ''`
	var count int64
	if err := r.ch.QueryRow(ctx, sql, tenantID, from, to).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetConversions returns conversion count and rate
func (r *Repository) GetConversions(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (int64, float64, error) {
	sql := `
		SELECT count(), if(uniqCombined(user_id) > 0,
			count() / uniqCombined(user_id) * 100, 0)
		FROM analytics.events
		WHERE tenant_id = ? AND event_type = 'conversion'
			AND timestamp BETWEEN ? AND ?`
	var count int64
	var rate float64
	if err := r.ch.QueryRow(ctx, sql, tenantID, from, to).Scan(&count, &rate); err != nil {
		return 0, 0, err
	}
	return count, rate, nil
}

// GetTopSources returns traffic source breakdown
func (r *Repository) GetTopSources(ctx context.Context, tenantID uuid.UUID, from, to time.Time, limit int) ([]models.SourceStat, error) {
	if limit <= 0 {
		limit = 10
	}
	sql := fmt.Sprintf(`
		SELECT source, uniqCombined(user_id) as visitors,
			countIf(event_type = 'conversion') as convs,
			sumIf(value, event_type = 'conversion') as rev
		FROM analytics.events
		WHERE tenant_id = ? AND timestamp BETWEEN ? AND ?
		GROUP BY source ORDER BY visitors DESC LIMIT %d`, limit)

	var results []models.SourceStat
	rows, err := r.ch.Query(ctx, sql, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s models.SourceStat
		if err := rows.Scan(&s.Source, &s.Visitors, &s.Conversions, &s.Revenue); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetTopPages returns top pages by views
func (r *Repository) GetTopPages(ctx context.Context, tenantID uuid.UUID, from, to time.Time, limit int) ([]models.PageStat, error) {
	if limit <= 0 {
		limit = 10
	}
	sql := fmt.Sprintf(`
		SELECT url,
			count() as page_views,
			uniqCombined(user_id) as visitors,
			avgIf(toUnixTimestamp(max(timestamp)) - toUnixTimestamp(min(timestamp)),
				event_type = 'page_view') as avg_time,
			countIf(event_type = 'bounce') / countIf(event_type = 'page_view') * 100 as bounce
		FROM analytics.events
		WHERE tenant_id = ? AND timestamp BETWEEN ? AND ? AND event_type = 'page_view'
		GROUP BY url ORDER BY page_views DESC LIMIT %d`, limit)

	var results []models.PageStat
	rows, err := r.ch.Query(ctx, sql, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.PageStat
		if err := rows.Scan(&p.URL, &p.PageViews, &p.Visitors, &p.AvgTimeSec, &p.BounceRate); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// GetTrend returns time-series data for a specific metric
func (r *Repository) GetTrend(ctx context.Context, tenantID uuid.UUID, eventType string, from, to time.Time, granularity string) ([]models.TrendPoint, error) {
	col := mapGranularity(granularity)
	sql := fmt.Sprintf(`
		SELECT toStartOfInterval(timestamp, INTERVAL 1 %s) as ts,
			sum(value) as val, count() as cnt
		FROM analytics.events
		WHERE tenant_id = ? AND event_type = ? AND timestamp BETWEEN ? AND ?
		GROUP BY ts ORDER BY ts`, col)

	var results []models.TrendPoint
	rows, err := r.ch.Query(ctx, sql, tenantID, eventType, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t models.TrendPoint
		if err := rows.Scan(&t.Timestamp, &t.Value, &t.Count); err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	return results, rows.Err()
}

func mapGranularity(g string) string {
	switch strings.ToLower(g) {
	case "minute", "min":
		return "minute"
	case "hour", "hr":
		return "hour"
	case "day":
		return "day"
	case "week":
		return "week"
	case "month":
		return "month"
	default:
		return "day"
	}
}
