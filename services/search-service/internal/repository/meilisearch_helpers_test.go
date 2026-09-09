// Extra tests for search-service meilisearch repository helpers.
package repository

import (
	"strings"
	"testing"
)

func TestQString_ToFilterString_Empty(t *testing.T) {
	var q qString = ""
	got := q.toFilterString()
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestQString_ToFilterString_NonEmpty(t *testing.T) {
	var q qString = "tenant-123"
	got := q.toFilterString()
	if !strings.Contains(got, "tenant_id") {
		t.Error("missing tenant_id")
	}
	if !strings.Contains(got, "tenant-123") {
		t.Error("missing tenant value")
	}
}

func TestSearchFilterBuilder_Add(t *testing.T) {
	b := &SearchFilterBuilder{}
	b.Add("field1", "value1")
	if len(b.parts) != 1 {
		t.Errorf("parts len: %d", len(b.parts))
	}
}

func TestSearchFilterBuilder_Add_Chained(t *testing.T) {
	b := &SearchFilterBuilder{}
	result := b.Add("a", "1").Add("b", "2")
	if result != b {
		t.Error("Add should return same builder")
	}
	if len(b.parts) != 2 {
		t.Errorf("parts: %d", len(b.parts))
	}
}

func TestSearchFilterBuilder_String_Empty(t *testing.T) {
	b := &SearchFilterBuilder{}
	if b.String() != "" {
		t.Error("empty builder should return empty string")
	}
}

func TestSearchFilterBuilder_String_Multiple(t *testing.T) {
	b := &SearchFilterBuilder{}
	b.Add("a", "1").Add("b", "2")
	got := b.String()
	if !strings.Contains(got, " AND ") {
		t.Error("missing AND separator")
	}
	if !strings.Contains(got, "a = '1'") {
		t.Error("missing first part")
	}
	if !strings.Contains(got, "b = '2'") {
		t.Error("missing second part")
	}
}

func TestSearchFilterBuilder_Build_Empty(t *testing.T) {
	b := &SearchFilterBuilder{}
	if b.Build() != "" {
		t.Error("Build should return empty")
	}
}

func TestEscapeFilter(t *testing.T) {
	got := escapeFilter("hello world")
	if got == "" {
		t.Error("should produce output")
	}
	if !strings.Contains(got, "hello") {
		t.Error("missing 'hello'")
	}
}

func TestEscapeFilter_SpecialChars(t *testing.T) {
	got := escapeFilter("a&b=c")
	if got == "" {
		t.Error("should produce output")
	}
}

func TestErrors(t *testing.T) {
	if ErrMeilisearchUnreachable == nil {
		t.Error("nil error")
	}
	if ErrInvalidQuery == nil {
		t.Error("nil error")
	}
}

func TestNewMeilisearch_TrimsSlash(t *testing.T) {
	s := NewMeilisearch("http://example.com/", "key")
	if s.host != "http://example.com" {
		t.Errorf("host: %s", s.host)
	}
}

func TestNewMeilisearch_NoSlash(t *testing.T) {
	s := NewMeilisearch("http://example.com", "key")
	if s.host != "http://example.com" {
		t.Errorf("host: %s", s.host)
	}
}

func TestNewMeilisearch_ClientTimeout(t *testing.T) {
	s := NewMeilisearch("http://example.com", "")
	if s.client == nil {
		t.Fatal("nil client")
	}
	if s.client.Timeout == 0 {
		t.Error("timeout should be set")
	}
}
