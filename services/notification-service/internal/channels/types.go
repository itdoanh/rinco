// Package channels implements notification delivery for each channel.
// Shared types are defined here; each channel file implements the Driver interface.
package channels

import (
	"context"
	"time"
)

// Notification is the canonical input used by every channel driver.
type Notification struct {
	TenantID  string
	UserID    string
	Type      string
	Title     string
	Body      string
	Icon      string
	Data      map[string]any
	ExpiresAt *time.Time
	// Channel-specific routing
	Email    string // email channel
	Phone    string // SMS channel
	Token    string // push / FCM channel
	Endpoint string // web-push endpoint
}

// DeliveryResult describes what happened during delivery.
type DeliveryResult struct {
	Status   string // sent | failed
	Provider string
}

// Driver is the interface every notification channel must implement.
type Driver interface {
	Name() string
	Send(ctx context.Context, n Notification) (DeliveryResult, error)
}
