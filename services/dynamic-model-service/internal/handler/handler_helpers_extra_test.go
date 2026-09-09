// Extra tests for dynamic-model-service handler helpers (pure helpers only).
package handler

import (
	"testing"

	"github.com/rinco/services/dynamic-model-service/internal/validation"
)

func TestZeroUUID(t *testing.T) {
	if got := zeroUUID(); got != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("got %s", got)
	}
}

func TestOrEmpty_Nil(t *testing.T) {
	got := orEmpty(nil)
	if got == nil {
		t.Error("expected empty map, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map: %v", got)
	}
}

func TestOrEmpty_Present(t *testing.T) {
	m := map[string]interface{}{"a": 1}
	got := orEmpty(m)
	if got["a"] != 1 {
		t.Errorf("got %v", got)
	}
}

func TestLower(t *testing.T) {
	in := []string{"Foo", " BAR ", "Baz"}
	out := lower(in)
	if out[0] != "foo" || out[1] != "bar" || out[2] != "baz" {
		t.Errorf("got %v", out)
	}
}

func TestLower_Empty(t *testing.T) {
	in := []string{}
	out := lower(in)
	if len(out) != 0 {
		t.Errorf("got %v", out)
	}
}

func TestToCSV_String(t *testing.T) {
	if toCSV("hello") != "hello" {
		t.Error("string should pass through")
	}
}

func TestToCSV_Float(t *testing.T) {
	if toCSV(1.5) != "1.5" {
		t.Errorf("got %s", toCSV(1.5))
	}
}

func TestToCSV_Bool(t *testing.T) {
	if toCSV(true) != "true" {
		t.Errorf("got %s", toCSV(true))
	}
}

func TestToCSV_Nil(t *testing.T) {
	if toCSV(nil) != "" {
		t.Errorf("got %s", toCSV(nil))
	}
}

func TestToCSV_Map(t *testing.T) {
	if toCSV(map[string]interface{}{"a": "b"}) != `{"a":"b"}` {
		t.Errorf("got %s", toCSV(map[string]interface{}{"a": "b"}))
	}
}

func TestToCSV_Slice(t *testing.T) {
	if toCSV([]interface{}{1, 2}) != "[1,2]" {
		t.Errorf("got %s", toCSV([]interface{}{1, 2}))
	}
}

func TestToCSV_Int(t *testing.T) {
	if toCSV(123) != "123" {
		t.Errorf("got %s", toCSV(123))
	}
}

func TestJoinErrs_Empty(t *testing.T) {
	if joinErrs(nil) != "" {
		t.Errorf("got %s", joinErrs(nil))
	}
}

func TestJoinErrs_One(t *testing.T) {
	errs := []validation.Error{{Field: "name", Message: "required"}}
	got := joinErrs(errs)
	if got != "name: required" {
		t.Errorf("got %s", got)
	}
}

func TestJoinErrs_Multiple(t *testing.T) {
	errs := []validation.Error{{Field: "name", Message: "required"}, {Field: "email", Message: "invalid"}}
	got := joinErrs(errs)
	if got != "name: required; email: invalid" {
		t.Errorf("got %s", got)
	}
}

func TestAttachJSON_Empty(t *testing.T) {
	got := attachJSON(map[string]interface{}{})
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestAttachJSON_One(t *testing.T) {
	got := attachJSON(map[string]interface{}{}, "data", []byte(`{"x":1}`))
	if got["data"] == nil {
		t.Error("expected data")
	}
}

func TestAttachJSON_Multiple(t *testing.T) {
	got := attachJSON(map[string]interface{}{}, "a", []byte(`{"x":1}`), "b", []byte(`{"y":2}`))
	if got["a"] == nil || got["b"] == nil {
		t.Errorf("got %v", got)
	}
}

func TestAttachJSON_InvalidJSON(t *testing.T) {
	got := attachJSON(map[string]interface{}{}, "data", []byte(`not-json`))
	// implementation sets target[pairs[i].(string)] = decoded (empty map for invalid json)
	v, ok := got["data"]
	if !ok {
		t.Errorf("key should exist: %v", got)
	}
	// either nil or empty map is acceptable
	if v != nil {
		m, isMap := v.(map[string]interface{})
		if !isMap || len(m) != 0 {
			t.Errorf("expected nil or empty map for invalid json, got %v", v)
		}
	}
}

func TestValidFieldType_String(t *testing.T) {
	if !validFieldType("string") {
		t.Error("string should be valid")
	}
}

func TestValidFieldType_Number(t *testing.T) {
	if !validFieldType("number") {
		t.Error("number should be valid")
	}
}

func TestValidFieldType_Integer(t *testing.T) {
	if !validFieldType("integer") {
		t.Error("integer should be valid")
	}
}

func TestValidFieldType_Boolean(t *testing.T) {
	if !validFieldType("boolean") {
		t.Error("boolean should be valid")
	}
}

func TestValidFieldType_Email(t *testing.T) {
	if !validFieldType("email") {
		t.Error("email should be valid")
	}
}

func TestValidFieldType_Phone(t *testing.T) {
	if !validFieldType("phone") {
		t.Error("phone should be valid")
	}
}

func TestValidFieldType_URL(t *testing.T) {
	if !validFieldType("url") {
		t.Error("url should be valid")
	}
}

func TestValidFieldType_Color(t *testing.T) {
	if !validFieldType("color") {
		t.Error("color should be valid")
	}
}

func TestValidFieldType_Relation(t *testing.T) {
	if !validFieldType("relation") {
		t.Error("relation should be valid")
	}
}

func TestValidFieldType_Invalid(t *testing.T) {
	if validFieldType("garbage") {
		t.Error("garbage should be invalid")
	}
}

func TestValidFieldType_Empty(t *testing.T) {
	if validFieldType("") {
		t.Error("empty should be invalid")
	}
}

func TestSlugRe_Match(t *testing.T) {
	if !slugRe.MatchString("hello-world_1") {
		t.Error("should match")
	}
}

func TestSlugRe_NoMatch(t *testing.T) {
	if slugRe.MatchString("Hello") {
		t.Error("uppercase should not match")
	}
}

func TestSlugRe_Spaces(t *testing.T) {
	if slugRe.MatchString("hello world") {
		t.Error("spaces should not match")
	}
}

func TestIdentRe_Match(t *testing.T) {
	if !identRe.MatchString("hello_world1") {
		t.Error("should match")
	}
}

func TestIdentRe_NoMatch(t *testing.T) {
	if identRe.MatchString("1hello") {
		t.Error("digit start should not match")
	}
}

func TestIdentRe_NoMatch_Special(t *testing.T) {
	if identRe.MatchString("hello-world") {
		t.Error("hyphen should not match")
	}
}

func TestModelSchema(t *testing.T) {
	if modelSchema != "model" {
		t.Errorf("got %s", modelSchema)
	}
}
