// Package preferences provides per-user channel and type preference resolution.
// It checks quiet hours, per-type settings, and applies default priority maps
// to determine which channels a notification should be delivered through.
package preferences

import (
	"log/slog"
	"strings"
	"time"
)

// DefaultPriorityChannels maps each priority level to the default channel list.
var DefaultPriorityChannels = map[string][]string{
	"high":   {"in_app", "email", "push"},
	"normal": {"in_app", "email"},
	"low":    {"in_app"},
}

// Preference holds the per-(user, type, channel) settings loaded from the DB.
type Preference struct {
	UserID      string
	NotifType   string
	Channel     string
	Enabled     bool
	QuietStart  *int // hour 0-23, nil = no quiet hours
	QuietEnd    *int
	DigestMode  string // none | daily | weekly
}

// Resolver computes which channels to use for a given notification.
type Resolver struct {
	prefs    []Preference
	now      func() time.Time
}

func defaultNow() time.Time { return time.Now() }

// NewResolver creates a resolver with the given preferences.
func NewResolver(prefs []Preference) *Resolver {
	return &Resolver{prefs: prefs, now: defaultNow}
}

// Resolve returns the list of channels to deliver through, after applying
// user preferences, priority defaults, and quiet-hours filtering.
func (r *Resolver) Resolve(priority, notifType, userID string) []string {
	now := r.now()
	defaults, ok := DefaultPriorityChannels[priority]
	if !ok {
		defaults = DefaultPriorityChannels["normal"]
	}

	// Check quiet hours across all channels (except in_app)
	// if currently in quiet period, exclude non-in_app channels
	isQuiet := r.isQuietHours(now)
	if isQuiet {
		result := make([]string, 0, 1)
		for _, ch := range defaults {
			if ch == "in_app" {
				result = append(result, ch)
			}
		}
		if len(result) == 0 {
			result = []string{"in_app"}
		}
		return result
	}

	// Filter by user preferences
	enabled := make(map[string]bool)
	for _, ch := range defaults {
		enabled[ch] = true
	}
	for _, p := range r.prefs {
		if p.UserID != userID {
			continue
		}
		if p.NotifType != "" && p.NotifType != notifType && p.NotifType != "*" {
			continue
		}
		// If type-specific setting found, respect it for that type
		if p.NotifType == notifType || p.NotifType == "*" {
			if !p.Enabled {
				delete(enabled, p.Channel)
			} else {
				enabled[p.Channel] = true
			}
		}
	}

	result := make([]string, 0, len(defaults))
	for _, ch := range defaults {
		if enabled[ch] {
			result = append(result, ch)
		}
	}
	return result
}

// isQuietHours returns true if the current time falls within any quiet period.
func (r *Resolver) isQuietHours(now time.Time) bool {
	hour := now.Hour()
	for _, p := range r.prefs {
		if p.QuietStart == nil || p.QuietEnd == nil {
			continue
		}
		if r.inQuietPeriod(hour, *p.QuietStart, *p.QuietEnd) {
			return true
		}
	}
	return false
}

// inQuietPeriod handles overnight quiet periods (e.g., 22:00-07:00).
func (r *Resolver) inQuietPeriod(hour, start, end int) bool {
	if start <= end {
		return hour >= start && hour < end
	}
	// Overnight: start > end (e.g., 22:00-07:00)
	return hour >= start || hour < end
}

// ParseQuietHours parses a [start, end] pair from a JSON array.
func ParseQuietHours(v []int) (start, end *int) {
	if len(v) < 2 {
		return nil, nil
	}
	if v[0] >= 0 && v[0] <= 23 {
		start = &v[0]
	}
	if v[1] >= 0 && v[1] <= 23 {
		end = &v[1]
	}
	return start, end
}

// Merge merges the given preference updates into the resolver's preference list.
func (r *Resolver) Merge(updates []Preference) {
	seen := map[string]bool{}
	for _, p := range r.prefs {
		seen[p.UserID+p.NotifType+p.Channel] = true
	}
	for _, u := range updates {
		k := u.UserID + u.NotifType + u.Channel
		if !seen[k] {
			r.prefs = append(r.prefs, u)
		} else {
			for i, p := range r.prefs {
				if p.UserID+u.NotifType+p.Channel == k {
					r.prefs[i] = u
					break
				}
			}
		}
	}
	_ = slog.Default()
	_ = strings.TrimSpace
}
