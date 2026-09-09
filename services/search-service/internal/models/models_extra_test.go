// Extra tests for search-service models.
package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/itdoanh/rinco/services/search-service/internal/models"
)

func TestExtraSearchQuery_BuildFilter_EmptyTenant(t *testing.T) {
	q := models.SearchQuery{}
	if got := q.BuildFilter(); got != "" {
		t.Errorf("empty tenant: got %q", got)
	}
}

func TestExtraSearchQuery_BuildFilter_TenantOnly(t *testing.T) {
	q := models.SearchQuery{TenantID: "tenant-1"}
	want := "tenant_id = 'tenant-1'"
	if got := q.BuildFilter(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtraDocument_Defaults(t *testing.T) {
	doc := models.Document{}
	if doc.ID != "" {
		t.Error("default ID should be empty")
	}
	if doc.Metadata != nil {
		t.Error("default Metadata should be nil")
	}
	if doc.Tags != nil {
		t.Error("default Tags should be nil")
	}
}

func TestExtraDocument_JSONRoundTrip(t *testing.T) {
	now := time.Now()
	doc := models.Document{
		ID:        "doc1",
		TenantID:  "t1",
		Type:      models.TypeLead,
		Title:     "Lead Title",
		Body:      "Lead body",
		Metadata:  map[string]interface{}{"source": "web"},
		Tags:      []string{"vip"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out models.Document
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.ID != doc.ID {
		t.Errorf("ID mismatch: %s vs %s", out.ID, doc.ID)
	}
	if out.Type != doc.Type {
		t.Errorf("Type mismatch: %s vs %s", out.Type, doc.Type)
	}
}

func TestExtraSearchableType_AllTypes(t *testing.T) {
	expected := []string{"contact", "company", "lead", "deal", "activity", "user", "document"}
	got := []string{
		string(models.TypeContact),
		string(models.TypeCompany),
		string(models.TypeLead),
		string(models.TypeDeal),
		string(models.TypeActivity),
		string(models.TypeUser),
		string(models.TypeDocument),
	}
	for i, e := range expected {
		if got[i] != e {
			t.Errorf("type[%d]: got %s, want %s", i, got[i], e)
		}
	}
}

func TestExtraIndexConfigs_HaveCoreAttrs(t *testing.T) {
	// All configs should at minimum have searchable and filterable attrs.
	for typ, cfg := range models.DefaultIndexConfigs {
		if cfg.Name == "" {
			t.Errorf("type %s has empty index name", typ)
		}
		if len(cfg.SearchableAttrs) == 0 {
			t.Errorf("type %s has no searchable attributes", typ)
		}
		if len(cfg.FilterableAttrs) == 0 {
			t.Errorf("type %s has no filterable attributes", typ)
		}
	}
}

func TestExtraSynonymsDictionary_Valid(t *testing.T) {
	if len(models.SynonymsDictionary) == 0 {
		t.Error("synonyms dictionary should not be empty")
	}
	for k, v := range models.SynonymsDictionary {
		if k == "" {
			t.Error("empty key in synonyms")
		}
		if len(v) == 0 {
			t.Errorf("key %s has empty values", k)
		}
	}
}

func TestExtraSearchResult_FacetStats(t *testing.T) {
	r := models.SearchResult{
		FacetStats: map[string]models.Facet{
			"tags": {Field: "tags", Values: []models.FacetValue{{Value: "vip", Count: 5}}},
		},
	}
	if r.FacetStats["tags"].Values[0].Count != 5 {
		t.Error("facet stats not preserved")
	}
}

func TestExtraSearchHit_Score(t *testing.T) {
	h := models.SearchHit{Score: 0.95}
	if h.Score != 0.95 {
		t.Errorf("Score: %f", h.Score)
	}
}

func TestExtraSearchHit_Highlight(t *testing.T) {
	h := models.SearchHit{
		Highlight: map[string]string{"title": "<em>foo</em>"},
	}
	if h.Highlight["title"] != "<em>foo</em>" {
		t.Error("highlight not preserved")
	}
}

func TestExtraFacetStruct(t *testing.T) {
	f := models.Facet{Field: "f"}
	f.Values = append(f.Values, models.FacetValue{Value: "v", Count: 1})
	if len(f.Values) != 1 {
		t.Error("facet values not preserved")
	}
	if f.Values[0].Value != "v" {
		t.Error("facet value mismatch")
	}
}

func TestExtraDocument_EmptyMetadata(t *testing.T) {
	doc := models.Document{ID: "1", Metadata: map[string]interface{}{}}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty metadata should still marshal")
	}
}

func TestExtraIndexConfigs_NameUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for _, cfg := range models.DefaultIndexConfigs {
		if seen[cfg.Name] {
			t.Errorf("duplicate index name: %s", cfg.Name)
		}
		seen[cfg.Name] = true
	}
}

func TestExtraDocument_TimeFormat(t *testing.T) {
	now := time.Now()
	doc := models.Document{CreatedAt: now, UpdatedAt: now}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Verify time field present
	if !contains(b, `"created_at"`) {
		t.Error("created_at missing in JSON")
	}
	if !contains(b, `"updated_at"`) {
		t.Error("updated_at missing in JSON")
	}
}

func contains(haystack []byte, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && string(haystack) != "" && (len(haystack) >= len(needle) && findSubslice(haystack, []byte(needle)) >= 0)
}

func findSubslice(s, sub []byte) int {
outer:
	for i := 0; i+len(sub) <= len(s); i++ {
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}
