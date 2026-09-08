// Tests for dynamic-model-service validation package.
package validation

import (
	"regexp"
	"testing"
)

func TestEmailRegex(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"user@example.com", true},
		{"name+tag@example.co.uk", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := emailRe.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("emailRe.Match(%q): want %v, got %v", tt.input, tt.valid, got)
			}
		})
	}
}

func TestPhoneVNRegex(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"0912345678", true},
		{"+84912345678", true},
		{"0381234567", true},
		{"1234567890", false},
		{"091234567", false}, // too short
		{"abcdefghij", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := phoneVNRe.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("phoneVNRe.Match(%q): want %v, got %v", tt.input, tt.valid, got)
			}
		})
	}
}

func TestTaxCodeRegex(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"1234567890", true},
		{"1234567890-001", true},
		{"123456789", false},  // too short
		{"12345678901", false}, // too long without dash
		{"abcdefghij", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := taxCodeRe.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("taxCodeRe.Match(%q): want %v, got %v", tt.input, tt.valid, got)
			}
		})
	}
}

func TestCCCDRegex(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"123456789012", true},
		{"12345678901", false},  // 11 digits
		{"1234567890123", false}, // 13 digits
		{"abcdefghijkl", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cccdRe.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("cccdRe.Match(%q): want %v, got %v", tt.input, tt.valid, got)
			}
		})
	}
}

func TestColorRegex(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"#FF00AA", true},
		{"FF00AA", true},
		{"#fff", false},  // 3 chars not 6
		{"#FF00AABB", false}, // 8 chars
		{"red", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := colorRe.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("colorRe.Match(%q): want %v, got %v", tt.input, tt.valid, got)
			}
		})
	}
}

func TestValidate_Required(t *testing.T) {
	fields := []Field{{Name: "name", Type: "string", Required: true}}
	data := map[string]interface{}{} // missing name
	errs, err := Validate(fields, data)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Field != "name" {
		t.Errorf("Field: want name, got %s", errs[0].Field)
	}
	if errs[0].Message != "required" {
		t.Errorf("Message: want required, got %s", errs[0].Message)
	}
}

func TestValidate_Required_NilValue(t *testing.T) {
	fields := []Field{{Name: "name", Type: "string", Required: true}}
	data := map[string]interface{}{"name": nil}
	errs, _ := Validate(fields, data)
	if len(errs) != 1 {
		t.Errorf("expected 1 error for nil value, got %d", len(errs))
	}
}

func TestValidate_NotRequired_Missing(t *testing.T) {
	fields := []Field{{Name: "name", Type: "string", Required: false}}
	data := map[string]interface{}{}
	errs, _ := Validate(fields, data)
	if len(errs) != 0 {
		t.Errorf("expected no errors for missing optional, got %d", len(errs))
	}
}

func TestValidate_String(t *testing.T) {
	fields := []Field{{Name: "name", Type: "string"}}
	// Wrong type
	data := map[string]interface{}{"name": 123}
	errs, _ := Validate(fields, data)
	if len(errs) != 1 {
		t.Errorf("expected 1 error for non-string, got %d", len(errs))
	}
	if errs[0].Message != "must be string" {
		t.Errorf("Message: got %s", errs[0].Message)
	}
	// Correct type
	data2 := map[string]interface{}{"name": "John"}
	errs2, _ := Validate(fields, data2)
	if len(errs2) != 0 {
		t.Errorf("expected no errors for string, got %d", len(errs2))
	}
}

func TestValidate_StringMinMaxLength(t *testing.T) {
	fields := []Field{{
		Name: "name",
		Type: "string",
		ValidationRules: map[string]interface{}{
			"min_length": 3,
			"max_length": 5,
		},
	}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"ab", 1},   // too short
		{"abc", 0},  // min
		{"abcde", 0}, // max
		{"abcdef", 1}, // too long
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"name": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_StringPattern(t *testing.T) {
	fields := []Field{{
		Name: "code",
		Type: "string",
		ValidationRules: map[string]interface{}{
			"pattern": `^[A-Z]{3}$`,
		},
	}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"ABC", 0},
		{"abc", 1},
		{"ABCD", 1},
		{"AB", 1},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"code": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Number(t *testing.T) {
	fields := []Field{{Name: "age", Type: "number"}}
	tests := []struct {
		value    interface{}
		wantErrs int
	}{
		{42, 0},
		{42.5, 0},
		{"42", 0}, // string-coerced number
		{"not-a-number", 1},
		{nil, 0}, // nil allowed (not required)
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			data := map[string]interface{}{"age": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %v: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_NumberRange(t *testing.T) {
	fields := []Field{{
		Name: "age",
		Type: "number",
		ValidationRules: map[string]interface{}{
			"min": 18.0,
			"max": 100.0,
		},
	}}
	tests := []struct {
		value    float64
		wantErrs int
	}{
		{18, 0},
		{100, 0},
		{17, 1}, // below
		{101, 1}, // above
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			data := map[string]interface{}{"age": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %v: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Integer(t *testing.T) {
	fields := []Field{{Name: "count", Type: "integer"}}
	tests := []struct {
		value    interface{}
		wantErrs int
	}{
		{42, 0},
		{42.5, 1}, // not integer
		{"42", 0}, // string-coerced integer
		{"42.5", 1},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			data := map[string]interface{}{"count": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %v: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Boolean(t *testing.T) {
	fields := []Field{{Name: "active", Type: "boolean"}}
	tests := []struct {
		value    interface{}
		wantErrs int
	}{
		{true, 0},
		{false, 0},
		{"true", 1}, // string not bool
		{1, 1},       // int not bool
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			data := map[string]interface{}{"active": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %v: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Date(t *testing.T) {
	fields := []Field{{Name: "created", Type: "date"}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"2024-01-15", 0},
		{"2024-01-15T10:30:00Z", 0},
		{"not-a-date", 1},
		{"15/01/2024", 1},
		{"", 1}, // empty string fails type check
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"created": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Enum(t *testing.T) {
	fields := []Field{{
		Name: "status",
		Type: "enum",
		ValidationRules: map[string]interface{}{
			"options": []interface{}{"active", "inactive", "pending"},
		},
	}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"active", 0},
		{"inactive", 0},
		{"pending", 0},
		{"unknown", 1},
		{"", 1}, // empty string fails type check
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"status": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Email(t *testing.T) {
	fields := []Field{{Name: "email", Type: "email"}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"user@example.com", 0},
		{"invalid", 1},
		{"@example.com", 1},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"email": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_URL(t *testing.T) {
	fields := []Field{{Name: "website", Type: "url"}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"https://example.com", 0},
		{"http://example.com", 0},
		{"example.com", 1},
		{"ftp://example.com", 1},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"website": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Color(t *testing.T) {
	fields := []Field{{Name: "theme", Type: "color"}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"#FF00AA", 0},
		{"FF00AA", 0},
		{"red", 1},
		{"#FF00", 1},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"theme": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Array(t *testing.T) {
	fields := []Field{{Name: "tags", Type: "array"}}
	tests := []struct {
		value    interface{}
		wantErrs int
	}{
		{[]interface{}{"a", "b"}, 0},
		{[]interface{}{}, 0},
		{"not-array", 1},
		{map[string]interface{}{}, 1},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			data := map[string]interface{}{"tags": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %v: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_Object(t *testing.T) {
	fields := []Field{{Name: "config", Type: "object"}}
	tests := []struct {
		value    interface{}
		wantErrs int
	}{
		{map[string]interface{}{"key": "value"}, 0},
		{"not-object", 1},
		{[]interface{}{}, 1},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			data := map[string]interface{}{"config": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %v: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_CustomValidator_Email(t *testing.T) {
	fields := []Field{{
		Name: "contact",
		Type: "string",
		ValidationRules: map[string]interface{}{
			"validator": "email",
		},
	}}
	tests := []struct {
		value    string
		wantErrs int
	}{
		{"user@example.com", 0},
		{"invalid", 1},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			data := map[string]interface{}{"contact": tt.value}
			errs, _ := Validate(fields, data)
			if len(errs) != tt.wantErrs {
				t.Errorf("value %q: expected %d errors, got %d", tt.value, tt.wantErrs, len(errs))
			}
		})
	}
}

func TestValidate_CustomValidator_PhoneVN(t *testing.T) {
	fields := []Field{{
		Name: "phone",
		Type: "string",
		ValidationRules: map[string]interface{}{
			"validator": "phone-vn",
		},
	}}
	data := map[string]interface{}{"phone": "0912345678"}
	errs, _ := Validate(fields, data)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid phone, got %d", len(errs))
	}
}

func TestValidate_CustomValidator_TaxCode(t *testing.T) {
	fields := []Field{{
		Name: "tax",
		Type: "string",
		ValidationRules: map[string]interface{}{
			"validator": "tax-code",
		},
	}}
	data := map[string]interface{}{"tax": "1234567890"}
	errs, _ := Validate(fields, data)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid tax code, got %d", len(errs))
	}
}

func TestValidate_CustomValidator_CCCD(t *testing.T) {
	fields := []Field{{
		Name: "cccd",
		Type: "string",
		ValidationRules: map[string]interface{}{
			"validator": "cccd",
		},
	}}
	data := map[string]interface{}{"cccd": "123456789012"}
	errs, _ := Validate(fields, data)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid CCCD, got %d", len(errs))
	}
}

func TestValidate_CustomValidator_Unknown(t *testing.T) {
	fields := []Field{{
		Name: "x",
		Type: "string",
		ValidationRules: map[string]interface{}{
			"validator": "unknown-validator",
		},
	}}
	data := map[string]interface{}{"x": "anything"}
	errs, _ := Validate(fields, data)
	if len(errs) != 1 {
		t.Errorf("expected 1 error for unknown validator, got %d", len(errs))
	}
}

func TestNumeric(t *testing.T) {
	tests := []struct {
		input   interface{}
		wantOk  bool
		wantVal float64
	}{
		{42.5, true, 42.5},
		{int(42), true, 42},
		{int64(42), true, 42},
		{"42.5", true, 42.5},
		{"abc", false, 0},
		{nil, false, 0},
		{[]int{1, 2}, false, 0},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			v, ok := numeric(tt.input)
			if ok != tt.wantOk {
				t.Errorf("numeric(%v) ok: want %v, got %v", tt.input, tt.wantOk, ok)
			}
			if ok && v != tt.wantVal {
				t.Errorf("numeric(%v) val: want %v, got %v", tt.input, tt.wantVal, v)
			}
		})
	}
}

func TestStartsWith(t *testing.T) {
	if !startsWith("https://example.com", "https://") {
		t.Error("expected true")
	}
	if !startsWith("http://example.com", "http://") {
		t.Error("expected true")
	}
	if startsWith("example.com", "http://") {
		t.Error("expected false")
	}
	if startsWith("", "http://") {
		t.Error("expected false for empty string")
	}
}

func TestJSONSchemaForType(t *testing.T) {
	tests := []struct {
		fieldType string
		wantKey   string
		wantVal   interface{}
	}{
		{"string", "type", "string"},
		{"text", "type", "string"},
		{"number", "type", "number"},
		{"integer", "type", "integer"},
		{"boolean", "type", "boolean"},
		{"date", "format", "date-time"},
		{"datetime", "format", "date-time"},
		{"time", "format", "date-time"},
		{"array", "type", "array"},
		{"object", "type", "object"},
		{"json", "type", "object"},
		{"unknown", "type", nil}, // unknown has no type
	}
	for _, tt := range tests {
		t.Run(tt.fieldType, func(t *testing.T) {
			schema := JSONSchemaForType(tt.fieldType, nil)
			if tt.wantVal == nil {
				_, ok := schema["type"]
				if ok {
					t.Errorf("expected no type key, got %v", schema["type"])
				}
				return
			}
			if schema[tt.wantKey] != tt.wantVal {
				t.Errorf("%s: want %v, got %v", tt.wantKey, tt.wantVal, schema[tt.wantKey])
			}
		})
	}
}

func TestJSONSchemaForType_Rules(t *testing.T) {
	rules := map[string]interface{}{
		"min_length": 3,
		"max_length": 10,
		"min":        1.0,
		"max":        100.0,
		"pattern":    "^[a-z]+$",
		"options":    []interface{}{"a", "b", "c"},
	}
	schema := JSONSchemaForType("string", rules)
	if schema["minLength"] != 3 {
		t.Errorf("minLength: got %v", schema["minLength"])
	}
	if schema["maxLength"] != 10 {
		t.Errorf("maxLength: got %v", schema["maxLength"])
	}
	if schema["pattern"] != "^[a-z]+$" {
		t.Errorf("pattern: got %v", schema["pattern"])
	}
	if schema["enum"] == nil {
		t.Error("enum should be set")
	}
}

func TestString(t *testing.T) {
	s := String()
	if len(s) == 0 {
		t.Error("expected non-empty string")
	}
	// Should be a numeric string (Unix timestamp).
	matched := regexp.MustCompile(`^\d+$`).MatchString(s)
	if !matched {
		t.Errorf("expected numeric string, got %s", s)
	}
}
