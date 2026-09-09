// Extra tests for dynamic-model-service validation helpers (only JSONSchemaForType + helpers).
package validation

import (
	"testing"
)

func TestJSONSchemaForType_String(t *testing.T) {
	prop := JSONSchemaForType("string", nil)
	if prop["type"] != "string" {
		t.Errorf("expected string, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Email(t *testing.T) {
	prop := JSONSchemaForType("email", nil)
	if prop["type"] != "string" {
		t.Errorf("expected string, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Number(t *testing.T) {
	prop := JSONSchemaForType("number", nil)
	if prop["type"] != "number" {
		t.Errorf("expected number, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Integer(t *testing.T) {
	prop := JSONSchemaForType("integer", nil)
	if prop["type"] != "integer" {
		t.Errorf("expected integer, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Boolean(t *testing.T) {
	prop := JSONSchemaForType("boolean", nil)
	if prop["type"] != "boolean" {
		t.Errorf("expected boolean, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Date(t *testing.T) {
	prop := JSONSchemaForType("date", nil)
	if prop["type"] != "string" {
		t.Errorf("expected string, got %v", prop["type"])
	}
	if prop["format"] != "date-time" {
		t.Errorf("expected date-time format, got %v", prop["format"])
	}
}

func TestJSONSchemaForType_Enum(t *testing.T) {
	prop := JSONSchemaForType("enum", nil)
	if prop["type"] != "string" {
		t.Errorf("expected string, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Array(t *testing.T) {
	prop := JSONSchemaForType("array", nil)
	if prop["type"] != "array" {
		t.Errorf("expected array, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Object(t *testing.T) {
	prop := JSONSchemaForType("object", nil)
	if prop["type"] != "object" {
		t.Errorf("expected object, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Relation(t *testing.T) {
	prop := JSONSchemaForType("relation", nil)
	if prop["type"] != "string" {
		t.Errorf("expected string, got %v", prop["type"])
	}
}

func TestJSONSchemaForType_Unknown(t *testing.T) {
	prop := JSONSchemaForType("weird", nil)
	if _, ok := prop["type"]; ok {
		t.Errorf("unknown type should not have type field, got %v", prop)
	}
}

func TestJSONSchemaForType_WithMinMax(t *testing.T) {
	prop := JSONSchemaForType("string", map[string]interface{}{
		"min_length": float64(5),
		"max_length": float64(100),
	})
	if prop["minLength"] != 5 {
		t.Errorf("minLength: %v", prop["minLength"])
	}
	if prop["maxLength"] != 100 {
		t.Errorf("maxLength: %v", prop["maxLength"])
	}
}

func TestJSONSchemaForType_WithNumberMinMax(t *testing.T) {
	prop := JSONSchemaForType("number", map[string]interface{}{
		"min": float64(0),
		"max": float64(100),
	})
	if prop["minimum"] != float64(0) {
		t.Errorf("minimum: %v", prop["minimum"])
	}
	if prop["maximum"] != float64(100) {
		t.Errorf("maximum: %v", prop["maximum"])
	}
}

func TestJSONSchemaForType_WithPattern(t *testing.T) {
	prop := JSONSchemaForType("string", map[string]interface{}{
		"pattern": "^[a-z]+$",
	})
	if prop["pattern"] != "^[a-z]+$" {
		t.Errorf("pattern: %v", prop["pattern"])
	}
}

func TestJSONSchemaForType_WithEnumOptions(t *testing.T) {
	prop := JSONSchemaForType("enum", map[string]interface{}{
		"options": []interface{}{"a", "b", "c"},
	})
	opts, ok := prop["enum"].([]string)
	if !ok || len(opts) != 3 {
		t.Errorf("enum: %v", prop["enum"])
	}
}

func TestStartsWith_True(t *testing.T) {
	if !startsWith("https://example.com", "https://") {
		t.Error("should start with https://")
	}
}

func TestStartsWith_False(t *testing.T) {
	if startsWith("ftp://example.com", "https://") {
		t.Error("should not start with https://")
	}
}

func TestStartsWith_TooShort(t *testing.T) {
	if startsWith("ab", "abc") {
		t.Error("too short should return false")
	}
}

func TestNumericExtra(t *testing.T) {
	if v, ok := numeric(float64(1.5)); !ok || v != 1.5 {
		t.Error("float64")
	}
	if v, ok := numeric(int(5)); !ok || v != 5 {
		t.Error("int")
	}
	if v, ok := numeric(int64(10)); !ok || v != 10 {
		t.Error("int64")
	}
	if v, ok := numeric("3.14"); !ok || v != 3.14 {
		t.Error("string")
	}
	if _, ok := numeric(nil); ok {
		t.Error("nil should fail")
	}
}

func TestToFloatExtra(t *testing.T) {
	v, ok := toFloat(1.0)
	if !ok || v != 1.0 {
		t.Errorf("got %v, %v", v, ok)
	}
}

func TestStringExtra(t *testing.T) {
	s := String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}
