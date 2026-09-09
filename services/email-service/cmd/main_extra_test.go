// Extra tests for email-service cmd/main.go helpers.
package main

import (
	"encoding/json"
	"testing"
)

func TestNullString_Empty(t *testing.T) {
	if got := nullString(""); got != nil {
		t.Errorf("empty: got %v, want nil", got)
	}
}

func TestNullString_NonEmpty(t *testing.T) {
	got := nullString("hello")
	if got != "hello" {
		t.Errorf("non-empty: got %v, want hello", got)
	}
}

func TestMustJSON_Struct(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
	}
	got := mustJSON(sample{Name: "Alice"})
	if string(got) != `{"name":"Alice"}` {
		t.Errorf("got %s", got)
	}
}

func TestMustJSON_Map(t *testing.T) {
	got := mustJSON(map[string]int{"a": 1})
	var m map[string]int
	if err := json.Unmarshal(got, &m); err != nil {
		t.Errorf("unmarshal: %v", err)
	}
	if m["a"] != 1 {
		t.Errorf("got %v", m)
	}
}

func TestMustJSON_Nil(t *testing.T) {
	got := mustJSON(nil)
	if string(got) != "null" {
		t.Errorf("nil: got %s", got)
	}
}

func TestNullString_TypeCheck(t *testing.T) {
	// nullString returns interface{} - we want nil for empty and string for non-empty
	v := nullString("")
	s, ok := v.(*string)
	if !ok && v != nil {
		t.Errorf("empty: want nil or *string, got %T", v)
	}
	if ok {
		t.Errorf("empty: shouldn't return *string, got %v", *s)
	}

	v = nullString("hello")
	s2, ok := v.(string)
	if !ok {
		t.Errorf("non-empty: want string, got %T", v)
	}
	if s2 != "hello" {
		t.Errorf("got %v", s2)
	}
}
