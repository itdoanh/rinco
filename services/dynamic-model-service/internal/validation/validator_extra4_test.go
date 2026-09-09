// Extra tests for dynamic-model-service validator helpers (only NEW tests, with Ex suffix).
package validation

import "testing"

func TestJSONSchemaForType_String_Ex4(t *testing.T) {
	schema := JSONSchemaForType("string", nil)
	if schema["type"] != "string" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_Number_Ex4(t *testing.T) {
	schema := JSONSchemaForType("number", nil)
	if schema["type"] != "number" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_Boolean_Ex4(t *testing.T) {
	schema := JSONSchemaForType("boolean", nil)
	if schema["type"] != "boolean" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_Email_Ex4(t *testing.T) {
	schema := JSONSchemaForType("email", nil)
	if schema["type"] != "string" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_Phone_Ex4(t *testing.T) {
	schema := JSONSchemaForType("phone", nil)
	if schema["type"] != "string" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_URL_Ex4(t *testing.T) {
	schema := JSONSchemaForType("url", nil)
	if schema["type"] != "string" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_Color_Ex4(t *testing.T) {
	schema := JSONSchemaForType("color", nil)
	if schema["type"] != "string" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_JSON_Ex4(t *testing.T) {
	schema := JSONSchemaForType("json", nil)
	if schema["type"] != "object" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_Ref_Ex4(t *testing.T) {
	schema := JSONSchemaForType("ref", nil)
	if schema["type"] != "string" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_File_Ex4(t *testing.T) {
	schema := JSONSchemaForType("file", nil)
	if schema["type"] != "string" {
		t.Errorf("type: %v", schema["type"])
	}
}

func TestJSONSchemaForType_EmptyRules_Ex4(t *testing.T) {
	schema := JSONSchemaForType("string", nil)
	if schema == nil {
		t.Fatal("nil schema")
	}
}

func TestNumeric_Float64Extra(t *testing.T) {
	if v, ok := numeric(float64(3.14)); !ok || v != 3.14 {
		t.Error("float64")
	}
}

func TestNumeric_IntExtra(t *testing.T) {
	if v, ok := numeric(int(42)); !ok || v != 42 {
		t.Error("int")
	}
}

func TestNumeric_StringExtra(t *testing.T) {
	if v, ok := numeric("1.5"); !ok || v != 1.5 {
		t.Error("string")
	}
}

func TestNumeric_NilExtra(t *testing.T) {
	if _, ok := numeric(nil); ok {
		t.Error("nil should fail")
	}
}

func TestNumeric_BoolExtra(t *testing.T) {
	if _, ok := numeric(true); ok {
		t.Error("bool should fail")
	}
}

func TestNumeric_InvalidStringExtra(t *testing.T) {
	if _, ok := numeric("not-a-number"); ok {
		t.Error("invalid string should fail")
	}
}
