package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/search-service/internal/models"
)

// MeilisearchStore implements a Meilisearch-compatible search backend.
// Uses direct HTTP calls to Meilisearch REST API for portability.
type MeilisearchStore struct {
	host   string
	apiKey string
	client *http.Client
}

// NewMeilisearch creates a Meilisearch HTTP client.
func NewMeilisearch(host, apiKey string) *MeilisearchStore {
	return &MeilisearchStore{
		host:   strings.TrimRight(host, "/"),
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *MeilisearchStore) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.host+"/health", nil)
	if err != nil {
		return err
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ErrMeilisearchUnreachable
	}
	return nil
}

// CreateIndex creates a new searchable index.
func (s *MeilisearchStore) CreateIndex(ctx context.Context, uid string, primaryKey string) error {
	body := map[string]string{"uid": uid, "primaryKey": primaryKey}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", s.host+"/indexes", strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 200 = created, 4xx already exists = OK
	return nil
}

// UpdateSettings updates an index's search settings.
func (s *MeilisearchStore) UpdateSettings(ctx context.Context, uid string, settings map[string]any) error {
	jsonBody, _ := json.Marshal(settings)
	req, err := http.NewRequestWithContext(ctx, "PATCH", s.host+"/indexes/"+uid+"/settings", strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// IndexDocuments adds or replaces documents in an index.
func (s *MeilisearchStore) IndexDocuments(ctx context.Context, uid string, docs []models.Document) error {
	jsonBody, _ := json.Marshal(docs)
	req, err := http.NewRequestWithContext(ctx, "POST", s.host+"/indexes/"+uid+"/documents", strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("meilisearch indexing failed: status %d", resp.StatusCode)
	}
	return nil
}

// DeleteDocument removes a single document.
func (s *MeilisearchStore) DeleteDocument(ctx context.Context, uid, id string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", s.host+"/indexes/"+uid+"/documents/"+id, nil)
	if err != nil {
		return err
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// Search performs a search query.
func (s *MeilisearchStore) Search(ctx context.Context, uid string, query models.SearchQuery) (*models.SearchResult, error) {
	body := map[string]any{
		"q":            query.Query,
		"page":         query.Page + 1, // Meilisearch is 1-indexed
		"hitsPerPage":  query.HitsPerPage,
		"highlight":    query.Highlight,
	}
	if query.HitsPerPage == 0 {
		body["hitsPerPage"] = 20
	}
	if filter := query.BuildFilter(); filter != "" {
		body["filter"] = filter
	}
	if len(query.Types) > 0 {
		// Multi-index search is handled at service level
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", s.host+"/indexes/"+uid+"/search", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Index doesn't exist, return empty result
		return &models.SearchResult{Query: query.Query, Hits: []models.SearchHit{}}, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("meilisearch search failed: status %d", resp.StatusCode)
	}

	var raw struct {
		Hits                []map[string]any `json:"hits"`
		EstimatedTotalHits  int             `json:"estimatedTotalHits"`
		TotalHits           int             `json:"totalHits"`
		ProcessingTimeMs    int             `json:"processingTimeMs"`
		Page                int             `json:"page"`
		HitsPerPage         int             `json:"hitsPerPage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	hits := make([]models.SearchHit, 0, len(raw.Hits))
	for _, h := range raw.Hits {
		hit := models.SearchHit{}
		if id, ok := h["id"].(string); ok {
			hit.ID = id
		}
		if tID, ok := h["tenant_id"].(string); ok {
			hit.TenantID = tID
		}
		if t, ok := h["type"].(string); ok {
			hit.Type = models.SearchableType(t)
		}
		if title, ok := h["title"].(string); ok {
			hit.Title = title
		}
		if body, ok := h["body"].(string); ok {
			hit.Body = body
		}
		if meta, ok := h["metadata"].(map[string]any); ok {
			hit.Metadata = meta
		}
		if tags, ok := h["tags"].([]any); ok {
			for _, tag := range tags {
				if t, ok := tag.(string); ok {
					hit.Tags = append(hit.Tags, t)
				}
			}
		}
		hits = append(hits, hit)
	}

	result := &models.SearchResult{
		Query:        query.Query,
		Total:        raw.TotalHits,
		Page:         raw.Page - 1,
		HitsPerPage:  raw.HitsPerPage,
		ProcessingMs: raw.ProcessingTimeMs,
		Hits:         hits,
	}

	return result, nil
}

// Helper for tenant filter
type qString string

func (q qString) toFilterString() string {
	if q == "" {
		return ""
	}
	return fmt.Sprintf("tenant_id = '%s'", q)
}

// Errors
var (
	ErrMeilisearchUnreachable = errors.New("meilisearch unreachable")
	ErrInvalidQuery          = errors.New("invalid search query")
)

// Helper: extract types as strings (for matching index name)
func (s *MeilisearchStore) ListIndexes(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.host+"/indexes", nil)
	if err != nil {
		return nil, err
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, nil
	}
	var raw struct {
		Results []struct {
			UID string `json:"uid"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(raw.Results))
	for _, r := range raw.Results {
		out = append(out, r.UID)
	}
	return out, nil
}

// Helper for building filter
func (q SearchFilterBuilder) Build() string {
	return ""
}

// SearchFilterBuilder composes Meilisearch filter expressions
type SearchFilterBuilder struct {
	parts []string
}

func (b *SearchFilterBuilder) Add(field, value string) *SearchFilterBuilder {
	b.parts = append(b.parts, fmt.Sprintf("%s = '%s'", field, value))
	return b
}

func (b *SearchFilterBuilder) String() string {
	return strings.Join(b.parts, " AND ")
}

// Helper to escape values for filter
func escapeFilter(s string) string {
	return url.PathEscape(s)
}

// Suppress unused import warnings
var _ = uuid.Nil
