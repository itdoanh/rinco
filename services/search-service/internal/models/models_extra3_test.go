// Extra tests for search-service models.
package models

import (
	"testing"
)

func TestSearchableType_Constants(t *testing.T) {
	expected := map[SearchableType]string{
		TypeContact:  "contact",
		TypeCompany:  "company",
		TypeLead:     "lead",
		TypeDeal:     "deal",
		TypeActivity: "activity",
		TypeUser:     "user",
		TypeDocument: "document",
	}
	for k, v := range expected {
		if string(k) != v {
			t.Errorf("%v: got %s", k, v)
		}
	}
}

func TestBuildFilter_EmptyTenant(t *testing.T) {
	q := SearchQuery{TenantID: ""}
	if got := q.BuildFilter(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildFilter_WithTenant(t *testing.T) {
	q := SearchQuery{TenantID: "t1"}
	got := q.BuildFilter()
	if got != "tenant_id = 't1'" {
		t.Errorf("got %q", got)
	}
}

func TestDefaultIndexConfigs_HasEntries(t *testing.T) {
	if len(DefaultIndexConfigs) == 0 {
		t.Fatal("empty")
	}
	for k, v := range DefaultIndexConfigs {
		if v.Name == "" {
			t.Errorf("%s: empty name", k)
		}
	}
}

func TestDefaultIndexConfigs_Contact(t *testing.T) {
	cfg := DefaultIndexConfigs[TypeContact]
	if cfg.Name != "contacts" {
		t.Errorf("name: %s", cfg.Name)
	}
	if len(cfg.SearchableAttrs) < 3 {
		t.Error("searchable attrs too few")
	}
}

func TestSynonymsDictionary_HasEntries(t *testing.T) {
	if len(SynonymsDictionary) == 0 {
		t.Fatal("empty")
	}
	for k, v := range SynonymsDictionary {
		if k == "" {
			t.Error("empty key")
		}
		if len(v) == 0 {
			t.Errorf("%s: empty values", k)
		}
	}
}

func TestSynonymsDictionary_Vietnamese(t *testing.T) {
	if _, ok := SynonymsDictionary["khách hàng"]; !ok {
		t.Error("missing khách hàng")
	}
}

func TestDocument_Fields(t *testing.T) {
	d := Document{
		ID:       "1",
		TenantID: "t1",
		Type:     TypeLead,
		Title:    "Test",
		Body:     "Body",
	}
	if d.ID != "1" {
		t.Errorf("id: %s", d.ID)
	}
	if d.Type != TypeLead {
		t.Errorf("type: %s", d.Type)
	}
}

func TestSearchQuery_Defaults(t *testing.T) {
	q := SearchQuery{}
	if q.Query != "" {
		t.Error("expected empty Query")
	}
	if q.Page != 0 {
		t.Error("expected Page 0")
	}
	if q.HitsPerPage != 0 {
		t.Error("expected HitsPerPage 0")
	}
	if q.Highlight {
		t.Error("expected Highlight false")
	}
}

func TestSearchResult_Fields(t *testing.T) {
	r := SearchResult{
		Query:       "q",
		Total:       5,
		Page:        1,
		HitsPerPage: 10,
	}
	if r.Query != "q" {
		t.Errorf("query: %s", r.Query)
	}
	if r.Total != 5 {
		t.Errorf("total: %d", r.Total)
	}
}

func TestSearchHit_Highlight(t *testing.T) {
	h := SearchHit{
		Highlight: map[string]string{"title": "<em>foo</em>"},
		Score:     0.95,
	}
	if h.Highlight["title"] != "<em>foo</em>" {
		t.Errorf("highlight: %v", h.Highlight)
	}
	if h.Score != 0.95 {
		t.Errorf("score: %f", h.Score)
	}
}

func TestFacet_Fields(t *testing.T) {
	f := Facet{
		Field: "tag",
		Values: []FacetValue{{Value: "x", Count: 1}},
	}
	if f.Field != "tag" {
		t.Errorf("field: %s", f.Field)
	}
	if len(f.Values) != 1 {
		t.Errorf("values: %d", len(f.Values))
	}
}

func TestIndexConfig_AllFields(t *testing.T) {
	cfg := IndexConfig{
		Name:            "test",
		SearchableAttrs: []string{"a", "b"},
		FilterableAttrs: []string{"c"},
		SortableAttrs:   []string{"d"},
		RankingRules:    []string{"e"},
		StopWords:       []string{"the"},
		Synonyms:        map[string][]string{"x": {"y"}},
	}
	if cfg.Name != "test" {
		t.Error("name")
	}
	if len(cfg.SearchableAttrs) != 2 {
		t.Error("searchable")
	}
	if len(cfg.Synonyms) != 1 {
		t.Error("synonyms")
	}
}
