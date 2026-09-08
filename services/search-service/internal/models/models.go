package models

import "time"

// SearchableType defines the document type for indexing
type SearchableType string

const (
	TypeContact   SearchableType = "contact"
	TypeCompany   SearchableType = "company"
	TypeLead      SearchableType = "lead"
	TypeDeal      SearchableType = "deal"
	TypeActivity  SearchableType = "activity"
	TypeUser      SearchableType = "user"
	TypeDocument  SearchableType = "document"
)

// Document represents a single indexed document
type Document struct {
	ID       string                 `json:"id"`
	TenantID string                 `json:"tenant_id"`
	Type     SearchableType         `json:"type"`
	Title    string                 `json:"title"`
	Body     string                 `json:"body"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// BuildFilter constructs the Meilisearch filter string for tenant isolation.
func (q SearchQuery) BuildFilter() string {
	if q.TenantID == "" {
		return ""
	}
	return "tenant_id = '" + q.TenantID + "'"
}

// SearchQuery represents a search request
type SearchQuery struct {
	Query     string                 `json:"q"`
	TenantID  string                 `json:"tenant_id,omitempty"` // empty = cross-tenant (admin only)
	Types     []SearchableType       `json:"types,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Filters   map[string]string      `json:"filters,omitempty"`
	Sort      string                 `json:"sort,omitempty"`
	Page      int                    `json:"page,omitempty"`
	HitsPerPage int                  `json:"hits_per_page,omitempty"`
	Highlight bool                   `json:"highlight,omitempty"`
}

// SearchResult represents search response
type SearchResult struct {
	Query        string         `json:"query"`
	Total        int            `json:"total"`
	Page         int            `json:"page"`
	HitsPerPage  int            `json:"hits_per_page"`
	ProcessingMs int            `json:"processing_ms"`
	Hits         []SearchHit    `json:"hits"`
	FacetStats   map[string]Facet `json:"facets,omitempty"`
}

type SearchHit struct {
	Document
	Highlight map[string]string `json:"highlight,omitempty"`
	Score     float64           `json:"score"`
}

type Facet struct {
	Field string         `json:"field"`
	Values []FacetValue   `json:"values"`
}

type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// IndexConfig defines searchable index settings
type IndexConfig struct {
	Name              string
	SearchableAttrs   []string
	FilterableAttrs   []string
	SortableAttrs     []string
	RankingRules      []string
	StopWords         []string
	Synonyms          map[string][]string
}

// DefaultIndexConfigs provides settings for each indexable type
var DefaultIndexConfigs = map[SearchableType]IndexConfig{
	TypeContact: {
		Name: "contacts",
		SearchableAttrs: []string{"title", "body", "metadata.email", "metadata.phone"},
		FilterableAttrs: []string{"tenant_id", "tags", "type"},
		SortableAttrs:   []string{"created_at", "updated_at"},
		RankingRules:    []string{"words", "typo", "proximity", "attribute", "sort", "exactness"},
	},
	TypeLead: {
		Name: "leads",
		SearchableAttrs: []string{"title", "body"},
		FilterableAttrs: []string{"tenant_id", "tags", "metadata.status", "metadata.source"},
		SortableAttrs:   []string{"created_at", "metadata.score"},
	},
	TypeDeal: {
		Name: "deals",
		SearchableAttrs: []string{"title", "body"},
		FilterableAttrs: []string{"tenant_id", "tags", "metadata.stage"},
	},
	TypeDocument: {
		Name: "documents",
		SearchableAttrs: []string{"title", "body"},
		FilterableAttrs: []string{"tenant_id", "tags"},
	},
}

// SynonymsDictionary for Vietnamese support
var SynonymsDictionary = map[string][]string{
	"công ty":  {"doanh nghiệp", "company"},
	"khách hàng": {"customer", "client"},
	"nhân viên": {"staff", "employee"},
	"hợp đồng": {"contract", "agreement"},
}
