// Extra tests for search-service models (only new ones not already covered).
package models

import (
	"encoding/json"
	"testing"
)

func TestSearchableType_ConstantsEx(t *testing.T) {
	types := []SearchableType{TypeContact, TypeCompany, TypeLead, TypeDeal, TypeActivity, TypeUser, TypeDocument}
	for _, typ := range types {
		if typ == "" {
			t.Error("empty type")
		}
	}
}

func TestDocument_MetadataEx(t *testing.T) {
	d := Document{
		Metadata: map[string]interface{}{"email": "a@b.com", "score": 0.95},
	}
	if d.Metadata["email"] != "a@b.com" {
		t.Error("metadata")
	}
}

func TestDocument_TagsEx(t *testing.T) {
	d := Document{
		Tags: []string{"vip", "newsletter"},
	}
	if len(d.Tags) != 2 {
		t.Error("tags")
	}
}

func TestSearchQuery_BuildFilter_EmptyEx(t *testing.T) {
	q := SearchQuery{}
	if q.BuildFilter() != "" {
		t.Error("empty tenant should produce empty filter")
	}
}

func TestSearchQuery_BuildFilter_WithSpecialCharEx(t *testing.T) {
	q := SearchQuery{TenantID: "t-1"}
	got := q.BuildFilter()
	if got == "" {
		t.Error("expected non-empty")
	}
}

func TestFacet_ValuesEx(t *testing.T) {
	f := Facet{
		Field:  "tag",
		Values: []FacetValue{{Value: "x", Count: 1}, {Value: "y", Count: 2}},
	}
	if len(f.Values) != 2 {
		t.Errorf("values: %d", len(f.Values))
	}
	if f.Values[1].Count != 2 {
		t.Error("count")
	}
}

func TestIndexConfig_DefaultsEx(t *testing.T) {
	cfg := IndexConfig{Name: "test"}
	if cfg.Name != "test" {
		t.Error("name")
	}
}

func TestDefaultIndexConfigs_FilterableHasTenantIDEx(t *testing.T) {
	for typ, cfg := range DefaultIndexConfigs {
		has := false
		for _, attr := range cfg.FilterableAttrs {
			if attr == "tenant_id" {
				has = true
				break
			}
		}
		if !has {
			t.Errorf("%s missing tenant_id filterable", typ)
		}
	}
}

func TestDefaultIndexConfigs_HasEnglishEx(t *testing.T) {
	if _, ok := SynonymsDictionary["khách hàng"]; !ok {
		t.Error("missing khách hàng")
	}
}

func TestSearchResult_DefaultEx(t *testing.T) {
	r := SearchResult{}
	if r.Total != 0 {
		t.Error("total should default to 0")
	}
}

func TestSearchResult_WithHitsEx(t *testing.T) {
	r := SearchResult{
		Hits: []SearchHit{{Document: Document{ID: "1"}, Score: 0.9}},
	}
	if len(r.Hits) != 1 {
		t.Errorf("hits: %d", len(r.Hits))
	}
}

func TestSearchQuery_DefaultsEx(t *testing.T) {
	q := SearchQuery{}
	if q.Page != 0 {
		t.Error("default page")
	}
	if q.HitsPerPage != 0 {
		t.Error("default hits per page")
	}
}

func TestSearchQuery_TypesFilter(t *testing.T) {
	q := SearchQuery{Types: []SearchableType{TypeContact, TypeLead}}
	if len(q.Types) != 2 {
		t.Error("types")
	}
}

func TestDocument_JSON(t *testing.T) {
	d := Document{ID: "1", TenantID: "t1", Type: TypeContact, Title: "T", Body: "B"}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty json")
	}
}

func TestSearchHit_JSON(t *testing.T) {
	h := SearchHit{
		Document:  Document{ID: "1"},
		Highlight: map[string]string{"title": "<em>T</em>"},
		Score:     0.9,
	}
	b, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty json")
	}
}

func TestSearchResult_JSON(t *testing.T) {
	r := SearchResult{
		Query:        "test",
		Total:        10,
		Page:         1,
		HitsPerPage:  20,
		ProcessingMs: 50,
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty")
	}
}

func TestFacet_JSON(t *testing.T) {
	f := Facet{
		Field:  "tag",
		Values: []FacetValue{{Value: "x", Count: 1}},
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty")
	}
}
