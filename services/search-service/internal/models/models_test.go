package models_test

import (
	"testing"
	"time"

	"github.com/itdoanh/rinco/services/search-service/internal/models"
)

func TestDefaultIndexConfigs(t *testing.T) {
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

func TestSynonymsDictionary(t *testing.T) {
	if len(models.SynonymsDictionary) == 0 {
		t.Error("synonyms dictionary should not be empty")
	}
	if _, ok := models.SynonymsDictionary["công ty"]; !ok {
		t.Error("Vietnamese synonym 'công ty' missing")
	}
}

func TestDocumentCreation(t *testing.T) {
	now := time.Now()
	doc := models.Document{
		ID:        "doc-1",
		TenantID:  "tenant-1",
		Type:      models.TypeContact,
		Title:     "Nguyen Van A",
		Body:      "nguyen@example.com",
		Tags:      []string{"vip", "b2b"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if doc.Type != models.TypeContact {
		t.Errorf("expected contact, got %s", doc.Type)
	}
	if len(doc.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(doc.Tags))
	}
}

func TestSearchQueryDefaults(t *testing.T) {
	q := models.SearchQuery{Query: "test"}
	if q.Page != 0 {
		t.Errorf("default page should be 0, got %d", q.Page)
	}
	if q.HitsPerPage != 0 {
		t.Errorf("default hits_per_page should be 0, got %d", q.HitsPerPage)
	}
}

func TestSearchResultSum(t *testing.T) {
	r := &models.SearchResult{
		Query:       "test",
		Total:       100,
		Page:        1,
		HitsPerPage: 20,
		Hits:        make([]models.SearchHit, 20),
	}
	if len(r.Hits) != 20 {
		t.Errorf("expected 20 hits, got %d", len(r.Hits))
	}
}

func TestSearchableTypeConstants(t *testing.T) {
	types := []models.SearchableType{
		models.TypeContact, models.TypeCompany, models.TypeLead,
		models.TypeDeal, models.TypeActivity, models.TypeUser, models.TypeDocument,
	}
	for _, typ := range types {
		if typ == "" {
			t.Error("searchable type should not be empty")
		}
	}
}
