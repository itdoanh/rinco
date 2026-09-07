package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

var (
	emailRe   = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
	phoneVNRe = regexp.MustCompile(`^(\+84|0)(3|5|7|8|9)\d{8,9}$`)
	taxCodeRe = regexp.MustCompile(`^\d{10}(-\d{3})?$`)
	cccdRe    = regexp.MustCompile(`^\d{12}$`)
	colorRe   = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)
)

type Error struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Field struct {
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Required        bool                   `json:"required"`
	Default         map[string]interface{} `json:"default,omitempty"`
	ValidationRules map[string]interface{} `json:"validation_rules,omitempty"`
}

func Validate(fields []Field, data map[string]interface{}) ([]Error, error) {
	errs := make([]Error, 0)
	for _, f := range fields {
		val, present := data[f.Name]
		if !present || val == nil {
			if f.Required {
				errs = append(errs, Error{Field: f.Name, Message: "required"})
			}
			continue
		}
		if msg := validateField(f, val); msg != "" {
			errs = append(errs, Error{Field: f.Name, Message: msg})
		}
	}
	return errs, nil
}

func validateField(f Field, value interface{}) string {
	switch f.Type {
	case "string", "text":
		s, ok := value.(string)
		if !ok { return "must be string" }
		if min, ok := numeric(f.ValidationRules["min_length"]); ok && float64(len(s)) < min { return "min_length violated" }
		if max, ok := numeric(f.ValidationRules["max_length"]); ok && float64(len(s)) > max { return "max_length violated" }
		if p, ok := f.ValidationRules["pattern"].(string); ok && p != "" {
			re, err := regexp.Compile(p)
			if err == nil && !re.MatchString(s) { return "pattern violated" }
		}
		if v, ok := f.ValidationRules["validator"].(string); ok {
			if msg, bad := runCustomValidator(v, s); bad { return msg }
		}
	case "number":
		f1, ok := toFloat(value); if !ok { return "must be number" }
		if min, ok := numeric(f.ValidationRules["min"]); ok && f1 < min { return "below min" }
		if max, ok := numeric(f.ValidationRules["max"]); ok && f1 > max { return "above max" }
	case "integer":
		f1, ok := toFloat(value); if !ok || f1 != float64(int64(f1)) { return "must be integer" }
	case "boolean":
		if _, ok := value.(bool); !ok { return "must be boolean" }
	case "date", "datetime", "time":
		s, ok := value.(string); if !ok { return "must be date string" }
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			if _, err2 := time.Parse("2006-01-02", s); err2 != nil { return "invalid date format" }
		}
	case "enum":
		opts, _ := f.ValidationRules["options"].([]interface{})
		s, ok := value.(string); if !ok { return "must be string from enum" }
		for _, o := range opts { if o == s { return "" } }
		return "not in enum"
	case "email":
		s, ok := value.(string); if !ok { return "must be string" }
		if !emailRe.MatchString(s) { return "invalid email" }
	case "phone":
		s, ok := value.(string); if !ok { return "must be string" }
		if !phoneVNRe.MatchString(s) { return "invalid VN phone" }
	case "url":
		s, ok := value.(string); if !ok { return "must be string" }
		if !startsWith(s, "http://") && !startsWith(s, "https://") { return "must be http(s) URL" }
	case "color":
		s, ok := value.(string); if !ok { return "must be string" }
		if !colorRe.MatchString(s) { return "invalid color hex" }
	case "array":
		if _, ok := value.([]interface{}); !ok { return "must be array" }
	case "object", "json":
		if _, ok := value.(map[string]interface{}); !ok { return "must be object" }
	}
	if b, ok := f.ValidationRules["json_schema"].(map[string]interface{}); ok {
		if js, err := json.Marshal(b); err == nil {
			loader := gojsonschema.NewBytesLoader(js)
			data, _ := json.Marshal(value)
			if res, err := gojsonschema.Validate(loader, gojsonschema.NewBytesLoader(data)); err == nil && !res.Valid() {
				return "schema: " + res.Errors()[0].Description()
			}
		}
	}
	return ""
}

func runCustomValidator(name, s string) (string, bool) {
	switch name {
	case "email":
		if emailRe.MatchString(s) { return "", false }
	case "phone-vn":
		if phoneVNRe.MatchString(s) { return "", false }
	case "tax-code":
		if taxCodeRe.MatchString(s) { return "", false }
	case "cccd":
		if cccdRe.MatchString(s) { return "", false }
	}
	return "invalid " + name, true
}

func numeric(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64: return x, true
	case int: return float64(x), true
	case int64: return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64); return f, err == nil
	}
	return 0, false
}

func toFloat(v interface{}) (float64, bool) { return numeric(v) }

func startsWith(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }

func JSONSchemaForType(fieldType string, rules map[string]interface{}) map[string]interface{} {
	prop := map[string]interface{}{}
	switch fieldType {
	case "string", "text", "email", "phone", "url", "color":
		prop["type"] = "string"
	case "number": prop["type"] = "number"
	case "integer": prop["type"] = "integer"
	case "boolean": prop["type"] = "boolean"
	case "date", "datetime", "time": prop["type"] = "string"; prop["format"] = "date-time"
	case "enum": prop["type"] = "string"
	case "array": prop["type"] = "array"
	case "object", "json": prop["type"] = "object"
	case "relation", "ref", "file": prop["type"] = "string"
	}
	if rules == nil { return prop }
	if min, ok := numeric(rules["min_length"]); ok { prop["minLength"] = int(min) }
	if max, ok := numeric(rules["max_length"]); ok { prop["maxLength"] = int(max) }
	if min, ok := numeric(rules["min"]); ok { prop["minimum"] = min }
	if max, ok := numeric(rules["max"]); ok { prop["maximum"] = max }
	if p, ok := rules["pattern"].(string); ok { prop["pattern"] = p }
	if opts, ok := rules["options"].([]interface{}); ok {
		strs := make([]string, 0, len(opts))
		for _, o := range opts { if s, ok := o.(string); ok { strs = append(strs, s) } }
		prop["enum"] = strs
	}
	return prop
}

func String() string { return fmt.Sprint(time.Now().Unix()) }
