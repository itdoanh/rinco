// Package nats provides NATS pub/sub for lead service.
package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
)

// Client wraps NATS connection.
type Client struct {
	conn *nats.Conn
	mu   sync.RWMutex
}

// NewClient creates a new NATS client.
func NewClient(url string) (*Client, error) {
	if url == "" {
		return &Client{}, nil
	}
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// Close closes the NATS connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Publish publishes a message to a subject.
func (c *Client) Publish(ctx context.Context, subject string, data any) error {
	if c.conn == nil {
		slog.Warn("nats not connected; skipping publish", slog.String("subject", subject))
		return nil
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.conn.Publish(subject, bytes)
}

// Subscribe subscribes to a subject.
func (c *Client) Subscribe(subject string, handler func(data []byte)) (*nats.Subscription, error) {
	if c.conn == nil {
		return nil, nil
	}
	return c.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	return c.conn.Status() == nats.CONNECTED
}

// Subjects
const (
	SubjectLeadCreated   = "lead.created"
	SubjectLeadScored    = "lead.scored"
	SubjectLeadAssigned  = "lead.assigned"
	SubjectLeadConverted = "lead.converted"
	SubjectLeadUpdated   = "lead.updated"
	SubjectUserCreated   = "user.created"
)
