package capi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DedupConfig holds deduplication settings.
type DedupConfig struct {
	// Redis client for dedup storage
	Redis *redis.Client

	// Time-to-live for deduplication keys
	TTL time.Duration

	// Key prefix for dedup keys
	KeyPrefix string
}

// DefaultDedupConfig returns a default dedup configuration.
func DefaultDedupConfig(redis *redis.Client) DedupConfig {
	return DedupConfig{
		Redis:     redis,
		TTL:       72 * time.Hour,
		KeyPrefix: "capi:dedup:",
	}
}

// DedupResult represents the result of a deduplication check.
type DedupResult struct {
	IsDuplicate bool
	EventID     string
}

// Deduper handles event deduplication using Redis.
type Deduper struct {
	config DedupConfig
}

// NewDeduper creates a new deduplication handler.
func NewDeduper(cfg DedupConfig) *Deduper {
	if cfg.TTL == 0 {
		cfg.TTL = 72 * time.Hour
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "capi:dedup:"
	}
	return &Deduper{config: cfg}
}

// Check checks if an event_id is a duplicate.
// Returns DedupResult with IsDuplicate = true if this is a duplicate.
// If event_id is empty, generates a new one.
func (d *Deduper) Check(ctx context.Context, eventID string) (DedupResult, error) {
	if eventID == "" {
		eventID = GenerateEventID("", "", "", time.Now().Unix())
	}

	key := d.config.KeyPrefix + eventID

	// Try to set the key with NX (only if not exists)
	set, err := d.config.Redis.SetNX(ctx, key, "1", d.config.TTL).Result()
	if err != nil {
		return DedupResult{EventID: eventID, IsDuplicate: false}, fmt.Errorf("redis setnx: %w", err)
	}

	return DedupResult{
		EventID:     eventID,
		IsDuplicate: !set, // If set failed (key exists), it's a duplicate
	}, nil
}

// CheckBatch checks multiple event_ids for duplicates.
func (d *Deduper) CheckBatch(ctx context.Context, eventIDs []string) ([]DedupResult, error) {
	results := make([]DedupResult, 0, len(eventIDs))

	// Use pipeline for efficiency
	pipe := d.config.Redis.Pipeline()
	cmds := make(map[string]*redis.BoolCmd, len(eventIDs))

	for _, id := range eventIDs {
		if id == "" {
			id = GenerateEventID("", "", "", time.Now().Unix())
		}
		key := d.config.KeyPrefix + id
		cmds[id] = pipe.SetNX(ctx, key, "1", d.config.TTL)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		// If pipeline fails, fall back to individual checks
		for _, id := range eventIDs {
			result, _ := d.Check(ctx, id)
			results = append(results, result)
		}
		return results, err
	}

	for _, id := range eventIDs {
		if id == "" {
			id = GenerateEventID("", "", "", time.Now().Unix())
		}
		isDuplicate, err := cmds[id].Result()
		if err != nil {
			isDuplicate = false // Assume not duplicate on error
		}
		results = append(results, DedupResult{
			EventID:     id,
			IsDuplicate: !isDuplicate,
		})
	}

	return results, nil
}

// FilterDuplicates removes duplicate events from a slice.
func (d *Deduper) FilterDuplicates(ctx context.Context, eventIDs []string) ([]string, error) {
	results, err := d.CheckBatch(ctx, eventIDs)
	if err != nil {
		return nil, err
	}

	unique := make([]string, 0, len(results))
	for _, r := range results {
		if !r.IsDuplicate {
			unique = append(unique, r.EventID)
		}
	}
	return unique, nil
}

// MarkDuplicate explicitly marks an event as a duplicate.
// Use this when you want to prevent an event from being processed even without the key existing.
func (d *Deduper) MarkDuplicate(ctx context.Context, eventID string) error {
	key := d.config.KeyPrefix + eventID
	return d.config.Redis.Set(ctx, key, "duplicate", d.config.TTL).Err()
}

// Clear removes a dedup key (for testing or manual override).
func (d *Deduper) Clear(ctx context.Context, eventID string) error {
	key := d.config.KeyPrefix + eventID
	return d.config.Redis.Del(ctx, key).Err()
}

// GenerateEventIDFromFields creates a deterministic event ID from key fields.
// This is used to generate a consistent event ID for the same lead submission.
func GenerateEventIDFromFields(tenantID, formSlug, email, phone string) string {
	// Combine fields
	data := fmt.Sprintf("%s|%s|%s|%s", tenantID, formSlug, email, phone)
	
	// Hash to get consistent ID
	h := sha256.Sum256([]byte(data))
	
	// Use first 16 bytes (32 hex chars) for UUID format
	hash := hex.EncodeToString(h[:])
	
	// Format as UUID: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%s-%s-4%s-%s-%s",
		hash[0:8],
		hash[8:12],
		hash[12:16],
		hash[16:20],
		hash[20:32],
	)
}

// EventIDConfig holds configuration for event ID generation.
type EventIDConfig struct {
	TenantID string
	FormSlug string
	Email    string
	Phone    string
	// Additional fields for uniqueness
	Timestamp int64
	IP        string
}

// NewEventID creates a new event ID based on configuration.
func NewEventID(cfg EventIDConfig) string {
	if cfg.Timestamp == 0 {
		cfg.Timestamp = time.Now().Unix()
	}
	return GenerateEventIDFromFields(cfg.TenantID, cfg.FormSlug, cfg.Email, cfg.Phone)
}
