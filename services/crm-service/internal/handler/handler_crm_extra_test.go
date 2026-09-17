// Package handler — pure-Go unit tests for pieces that don't require a
// database connection. Tests the workflow condition DSL and utility helpers.
package handler

import (
	"database/sql"
	"testing"
)

func TestEvaluateCondition_AllPredicates(t *testing.T) {
	cond := struct {
		All []map[string]any `json:"all"`
		Any []map[string]any `json:"any"`
	}{
		All: []map[string]any{
			{"field": "score", "op": ">", "value": 50},
			{"field": "source", "op": "==", "value": "facebook"},
		},
	}
	inputs := map[string]any{"score": 75.0, "source": "facebook"}
	if !evaluateCondition(cond, inputs) {
		t.Fatalf("expected all-match to pass")
	}
	inputs2 := map[string]any{"score": 30.0, "source": "facebook"}
	if evaluateCondition(cond, inputs2) {
		t.Fatalf("expected score<50 to fail")
	}
}

func TestEvaluateCondition_AnyPredicate(t *testing.T) {
	cond := struct {
		All []map[string]any `json:"all"`
		Any []map[string]any `json:"any"`
	}{
		Any: []map[string]any{
			{"field": "status", "op": "==", "value": "WON"},
			{"field": "status", "op": "==", "value": "LOST"},
		},
	}
	if !evaluateCondition(cond, map[string]any{"status": "WON"}) {
		t.Fatalf("WON should pass any")
	}
	if !evaluateCondition(cond, map[string]any{"status": "LOST"}) {
		t.Fatalf("LOST should pass any")
	}
	if evaluateCondition(cond, map[string]any{"status": "NEW"}) {
		t.Fatalf("NEW should fail any")
	}
}

func TestEvaluateCondition_Empty(t *testing.T) {
	cond := struct {
		All []map[string]any `json:"all"`
		Any []map[string]any `json:"any"`
	}{}
	if !evaluateCondition(cond, nil) {
		t.Fatalf("empty condition should pass")
	}
}

func TestEvaluateCondition_Contains(t *testing.T) {
	cond := struct {
		All []map[string]any `json:"all"`
		Any []map[string]any `json:"any"`
	}{
		All: []map[string]any{
			{"field": "utm_source", "op": "contains", "value": "ads"},
		},
	}
	if !evaluateCondition(cond, map[string]any{"utm_source": "facebook_ads"}) {
		t.Fatalf("contains should pass")
	}
	if evaluateCondition(cond, map[string]any{"utm_source": "google"}) {
		t.Fatalf("contains should fail")
	}
}

func TestCoalesceString(t *testing.T) {
	def := "X"
	if got := coalesceString(nil, def); got != def {
		t.Fatalf("nil -> def: got %s", got)
	}
	empty := ""
	if got := coalesceString(&empty, def); got != def {
		t.Fatalf("empty -> def: got %s", got)
	}
	val := "hi"
	if got := coalesceString(&val, def); got != "hi" {
		t.Fatalf("val -> val: got %s", got)
	}
}

func TestNullableString(t *testing.T) {
	if v := nullableString(sql.NullString{String: "a", Valid: true}); v != "a" {
		t.Fatalf("valid -> 'a': got %v", v)
	}
	if v := nullableString(sql.NullString{String: "", Valid: false}); v != nil {
		t.Fatalf("invalid -> nil: got %v", v)
	}
}
