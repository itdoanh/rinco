// Additional tests for dynamic-model-service validator.
package validation

import (
	"testing"
)

func TestExtra_EmailRegex(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"user.name+tag@example.co.uk", true},
		{"a@b.c", true},
		{"plainstring", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := emailRe.MatchString(tt.email); got != tt.valid {
				t.Errorf("email %q: got %v, want %v", tt.email, got, tt.valid)
			}
		})
	}
}

func TestExtra_PhoneVNRegex(t *testing.T) {
	tests := []struct {
		phone string
		valid bool
	}{
		{"0901234567", true},
		{"+84901234567", true},
		{"0381234567", true},
		{"0212345678", false}, // landline
		{"12345", false},
		{"abcdefghij", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			if got := phoneVNRe.MatchString(tt.phone); got != tt.valid {
				t.Errorf("phone %q: got %v, want %v", tt.phone, got, tt.valid)
			}
		})
	}
}

func TestExtra_TaxCodeRegex(t *testing.T) {
	tests := []struct {
		code  string
		valid bool
	}{
		{"1234567890", true},
		{"1234567890-001", true},
		{"12345", false},
		{"abcdefghij", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if got := taxCodeRe.MatchString(tt.code); got != tt.valid {
				t.Errorf("tax-code %q: got %v, want %v", tt.code, got, tt.valid)
			}
		})
	}
}

func TestExtra_CCCDRegex(t *testing.T) {
	tests := []struct {
		code  string
		valid bool
	}{
		{"123456789012", true},
		{"000000000000", true},
		{"12345", false},
		{"abcdefghijkl", false},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if got := cccdRe.MatchString(tt.code); got != tt.valid {
				t.Errorf("cccd %q: got %v, want %v", tt.code, got, tt.valid)
			}
		})
	}
}

func TestExtra_ColorRegex(t *testing.T) {
	tests := []struct {
		color string
		valid bool
	}{
		{"#FF0000", true},
		{"FF0000", true},
		{"#ff00aa", true},
		{"red", false},
		{"#GG0000", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			if got := colorRe.MatchString(tt.color); got != tt.valid {
				t.Errorf("color %q: got %v, want %v", tt.color, got, tt.valid)
			}
		})
	}
}

func TestExtra_Numeric(t *testing.T) {
	tests := []struct {
		v       interface{}
		wantF   float64
		wantOk  bool
	}{
		{1, 1, true},
		{int64(2), 2, true},
		{float64(3.5), 3.5, true},
		{"4.0", 4.0, true},
		{"abc", 0, false},
		{true, 0, false},
	}
	for _, tt := range tests {
		f, ok := numeric(tt.v)
		if ok != tt.wantOk {
			t.Errorf("numeric(%v): ok got %v, want %v", tt.v, ok, tt.wantOk)
		}
		if ok && f != tt.wantF {
			t.Errorf("numeric(%v): got %v, want %v", tt.v, f, tt.wantF)
		}
	}
}

func TestExtra_ToFloat(t *testing.T) {
	// Same as numeric
	if f, ok := toFloat(42); !ok || f != 42 {
		t.Errorf("toFloat(42): got %v, %v", f, ok)
	}
}

func TestExtra_StartsWith(t *testing.T) {
	if !startsWith("https://example.com", "https://") {
		t.Error("should match https://")
	}
	if !startsWith("http://example.com", "http://") {
		t.Error("should match http://")
	}
	if startsWith("ftp://example.com", "http://") {
		t.Error("should not match ftp://")
	}
	if startsWith("hi", "hello") {
		t.Error("should not match longer prefix")
	}
}

func TestExtra_Validate_NilData(t *testing.T) {
	// Should not panic
	errs, err := Validate(nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestExtra_Validate_EmptyFields(t *testing.T) {
	errs, err := Validate([]Field{}, map[string]interface{}{"foo": "bar"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestExtra_Validate_MultipleRequired(t *testing.T) {
	fields := []Field{
		{Name: "a", Type: "string", Required: true},
		{Name: "b", Type: "string", Required: true},
	}
	errs, _ := Validate(fields, map[string]interface{}{})
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}
}

func TestExtra_Validate_NilValue(t *testing.T) {
	fields := []Field{{Name: "x", Type: "string", Required: true}}
	errs, _ := Validate(fields, map[string]interface{}{"x": nil})
	if len(errs) != 1 {
		t.Errorf("expected 1 error for nil required, got %d", len(errs))
	}
}

func TestExtra_Validate_BooleanType(t *testing.T) {
	fields := []Field{{Name: "active", Type: "boolean"}}
	errs, _ := Validate(fields, map[string]interface{}{"active": "yes"})
	if len(errs) != 1 {
		t.Errorf("expected 1 error for string boolean, got %d", len(errs))
	}
}

func TestExtra_Validate_ArrayType(t *testing.T) {
	fields := []Field{{Name: "items", Type: "array"}}
	// Valid array
	errs, _ := Validate(fields, map[string]interface{}{"items": []interface{}{"a", "b"}})
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	// Invalid - not an array
	errs, _ = Validate(fields, map[string]interface{}{"items": "not-array"})
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestExtra_Validate_ObjectType(t *testing.T) {
	fields := []Field{{Name: "meta", Type: "object"}}
	// Valid object
	errs, _ := Validate(fields, map[string]interface{}{"meta": map[string]interface{}{"k": "v"}})
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	// Invalid
	errs, _ = Validate(fields, map[string]interface{}{"meta": "not-object"})
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestExtra_JSONSchemaForType_AllTypes(t *testing.T) {
	types := []string{"string", "text", "number", "integer", "boolean", "email", "phone", "url", "color", "enum", "array", "object", "json", "date", "datetime", "time", "relation", "ref", "file"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			schema := JSONSchemaForType(typ, nil)
			if schema == nil {
				t.Errorf("schema for %s is nil", typ)
			}
			if _, ok := schema["type"]; !ok {
				t.Errorf("schema for %s missing type", typ)
			}
		})
	}
}

func TestExtra_JSONSchemaForType_NilRules(t *testing.T) {
	schema := JSONSchemaForType("string", nil)
	if schema["type"] != "string" {
		t.Errorf("type: got %v", schema["type"])
	}
}

func TestExtra_JSONSchemaForType_StringConstraints(t *testing.T) {
	rules := map[string]interface{}{
		"min_length": 5,
		"max_length": 100,
		"pattern":    "^[a-z]+$",
	}
	schema := JSONSchemaForType("string", rules)
	if schema["minLength"] != 5 {
		t.Errorf("minLength: got %v", schema["minLength"])
	}
	if schema["maxLength"] != 100 {
		t.Errorf("maxLength: got %v", schema["maxLength"])
	}
	if schema["pattern"] != "^[a-z]+$" {
		t.Errorf("pattern: got %v", schema["pattern"])
	}
}

func TestExtra_JSONSchemaForType_NumericConstraints(t *testing.T) {
	rules := map[string]interface{}{
		"min": 0.0,
		"max": 100.0,
	}
	schema := JSONSchemaForType("number", rules)
	if schema["minimum"] != 0.0 {
		t.Errorf("minimum: got %v", schema["minimum"])
	}
	if schema["maximum"] != 100.0 {
		t.Errorf("maximum: got %v", schema["maximum"])
	}
}

func TestExtra_RunCustomValidator(t *testing.T) {
	// email validator
	if _, bad := runCustomValidator("email", "user@example.com"); bad {
		t.Error("valid email should not be bad")
	}
	if _, bad := runCustomValidator("email", "invalid"); !bad {
		t.Error("invalid email should be bad")
	}

	// phone-vn validator
	if _, bad := runCustomValidator("phone-vn", "0901234567"); bad {
		t.Error("valid phone should not be bad")
	}

	// unknown validator
	if _, bad := runCustomValidator("unknown", "x"); !bad {
		t.Error("unknown validator should be bad")
	}
}

func TestExtra_String(t *testing.T) {
	s := String()
	if s == "" {
		t.Error("String() should not return empty")
	}
}
