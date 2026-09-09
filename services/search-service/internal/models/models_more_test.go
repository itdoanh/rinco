package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSearchableTypeConstants(t *testing.T) {
	if TypeContact != "contact" {
		t.Errorf("TypeContact = %q, want 'contact'", TypeContact)
	}
	if TypeCompany != "company" {
		t.Errorf("TypeCompany = %q, want 'company'", TypeCompany)
	}
	if TypeLead != "lead" {
		t.Errorf("TypeLead = %q, want 'lead'", TypeLead)
	}
	if TypeDeal != "deal" {
		t.Errorf("TypeDeal = %q, want 'deal'", TypeDeal)
	}
	if TypeActivity != "activity" {
		t.Errorf("TypeActivity = %q, want 'activity'", TypeActivity)
	}
	if TypeUser != "user" {
		t.Errorf("TypeUser = %q, want 'user'", TypeUser)
	}
	if TypeDocument != "document" {
		t.Errorf("TypeDocument = %q, want 'document'", TypeDocument)
	}
}

func TestBuildFilterEmptyTenant(t *testing.T) {
	q := SearchQuery{}
	if got := q.BuildFilter(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildFilterWithTenant(t *testing.T) {
	q := SearchQuery{TenantID: "tenant-abc"}
	got := q.BuildFilter()
	if !strings.Contains(got, "tenant-abc") {
		t.Errorf("BuildFilter missing tenant: %q", got)
	}
	if !strings.Contains(got, "tenant_id") {
		t.Errorf("BuildFilter missing field: %q", got)
	}
}

func TestBuildFilterSpecialChars(t *testing.T) {
	q := SearchQuery{TenantID: "tenant-x'y"}
	got := q.BuildFilter()
	if !strings.Contains(got, "tenant-x'y") {
		t.Errorf("expected tenant in filter: %q", got)
	}
}

func TestDocumentJSONMarshal(t *testing.T) {
	now := time.Now()
	d := Document{
		ID:        "d1",
		TenantID:  "t1",
		Type:      TypeLead,
		Title:     "Test Lead",
		Body:      "Hello",
		Metadata:  map[string]interface{}{"status": "open"},
		Tags:      []string{"vip"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"id":"d1"`) {
		t.Errorf("missing id: %s", data)
	}
	if !strings.Contains(string(data), `"type":"lead"`) {
		t.Errorf("missing type: %s", data)
	}
}

func TestDocumentJSONUnmarshal(t *testing.T) {
	data := `{"id":"d2","tenant_id":"t2","type":"contact","title":"X","body":"Y"}`
	var d Document
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.ID != "d2" {
		t.Errorf("ID = %q, want d2", d.ID)
	}
	if d.Type != TypeContact {
		t.Errorf("Type = %q", d.Type)
	}
}

func TestSearchHitIncludesDocument(t *testing.T) {
	hit := SearchHit{
		Document: Document{ID: "h1", Title: "Hi"},
		Score:    0.95,
	}
	if hit.ID != "h1" {
		t.Error("embedded Document fields not accessible")
	}
	if hit.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", hit.Score)
	}
}

func TestFacetValues(t *testing.T) {
	f := Facet{
		Field: "tags",
		Values: []FacetValue{
			{Value: "vip", Count: 5},
			{Value: "premium", Count: 2},
		},
	}
	if len(f.Values) != 2 {
		t.Errorf("Values len = %d, want 2", len(f.Values))
	}
}

func TestDefaultIndexConfigsAllTypes(t *testing.T) {
	types := []SearchableType{
		TypeContact, TypeLead, TypeDeal, TypeDocument,
	}
	for _, t2 := range types {
		cfg, ok := DefaultIndexConfigs[t2]
		if !ok {
			t.Errorf("missing config for %s", t2)
			continue
		}
		if cfg.Name == "" {
			t.Errorf("%s: Name empty", t2)
		}
	}
}

func TestDefaultIndexConfigsFilterableIncludeTenant(t *testing.T) {
	for typ, cfg := range DefaultIndexConfigs {
		found := false
		for _, attr := range cfg.FilterableAttrs {
			if attr == "tenant_id" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("type %s: filterable attrs missing tenant_id", typ)
		}
	}
}

func TestSynonymsDictionaryVietnamese(t *testing.T) {
	if SynonymsDictionary["công ty"] == nil {
		t.Error("missing 'công ty' synonyms")
	}
	if SynonymsDictionary["khách hàng"] == nil {
		t.Error("missing 'khách hàng' synonyms")
	}
}

func TestSearchQueryDefaults(t *testing.T) {
	q := SearchQuery{Query: "test"}
	if q.Page != 0 {
		t.Errorf("Page = %d, default 0", q.Page)
	}
	if q.HitsPerPage != 0 {
		t.Errorf("HitsPerPage = %d, default 0", q.HitsPerPage)
	}
	if q.Highlight {
		t.Errorf("Highlight default false")
	}
}

func TestSearchResultHitCount(t *testing.T) {
	r := SearchResult{
		Query:       "test",
		Total:       10,
		Page:        1,
		HitsPerPage: 5,
		Hits: []SearchHit{
			{Document: Document{ID: "1"}},
			{Document: Document{ID: "2"}},
		},
	}
	if r.Total != 10 {
		t.Errorf("Total = %d, want 10", r.Total)
	}
	if len(r.Hits) != 2 {
		t.Errorf("len(Hits) = %d, want 2", len(r.Hits))
	}
}

func TestIndexConfigJSONMarshal(t *testing.T) {
	cfg := IndexConfig{
		Name:            "test",
		SearchableAttrs: []string{"title"},
		FilterableAttrs: []string{"tenant_id"},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"Name":"test"`) {
		t.Errorf("missing name: %s", data)
	}
}
