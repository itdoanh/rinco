// Package render provides the Go text/template engine used by the email
// service.  Templates can use the following custom funcs:
//   - upper / lower / title / trim
//   - default "<fallback>"
//   - dateFormat "<layout>" "<RFC3339 or layout>"
//   - i18n "<key>" -- returns the localised string from the in-memory
//     dictionary (Vietnamese / English)
//   - markdown "<raw>" -- converts the input Markdown to HTML using a
//     minimal built-in renderer (no third-party dependency).
//   - urlSafe "<value>" -- percent-encodes the value so it can be embedded
//     inside HTML attributes or query strings.
package render

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Engine wraps a template plus the i18n dictionaries.
type Engine struct {
	tmpl       *template.Template
	i18n       map[string]map[string]string
}

// NewEngine creates a new engine with the given template body. The body
// may declare any number of named templates via the {{define ...}} syntax.
func NewEngine(body string) (*Engine, error) {
	e := &Engine{i18n: defaultI18n()}
	t, err := template.New("email").Funcs(template.FuncMap{
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      strings.Title,
		"trim":       strings.TrimSpace,
		"default":    defVal,
		"dateFormat": dateFormat,
		"i18n":       e.tfuncI18n,
		"markdown":   mdToHTML,
		"urlSafe":    urlSafe,
	}).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	e.tmpl = t
	return e, nil
}

// Render executes the template (name="") with the supplied data and returns
// the resulting string.
func (e *Engine) Render(data map[string]any) (string, error) {
	if e == nil || e.tmpl == nil {
		return "", fmt.Errorf("template engine is not configured")
	}
	var buf bytes.Buffer
	if err := e.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}
	return buf.String(), nil
}

// tfuncI18n resolves a key against the in-memory dictionary for the
// language stored in data["lang"] (defaults to "vi").
func (e *Engine) tfuncI18n(key string, args ...any) string {
	lang := "vi"
	if len(args) > 0 {
		if s, ok := args[0].(string); ok && s != "" {
			lang = s
		}
	}
	dict, ok := e.i18n[lang]
	if !ok {
		dict = e.i18n["en"]
	}
	if v, ok := dict[key]; ok {
		return v
	}
	return key
}

func defVal(def, v any) any {
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok && s == "" {
		return def
	}
	return v
}

// dateFormat returns v formatted with the given layout.  If v is a string
// it is first parsed with time.RFC3339; if parsing fails it is returned
// unchanged.  If v is a time.Time the layout is applied directly.
func dateFormat(layout string, v any) string {
	switch t := v.(type) {
	case time.Time:
		return t.Format(layout)
	case string:
		if layout == "" {
			layout = time.RFC3339
		}
		if ts, err := time.Parse(time.RFC3339, t); err == nil {
			return ts.Format(layout)
		}
		return t
	default:
		return fmt.Sprintf("%v", v)
	}
}

// urlSafe percent-encodes the supplied value. If a second argument is
// provided it is treated as the path part of a URL whose query will be
// built from the first argument (a map[string]string).
func urlSafe(v any, args ...any) string {
	switch val := v.(type) {
	case string:
		return url.QueryEscape(val)
	case map[string]string:
		q := url.Values{}
		for k, vv := range val {
			q.Set(k, vv)
		}
		return q.Encode()
	}
	return url.QueryEscape(fmt.Sprintf("%v", v))
}

// =============================================================================
// Markdown → HTML (tiny built-in renderer)
// =============================================================================

var mdHeading = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
var mdBold = regexp.MustCompile(`\*\*([^*]+)\*\*`)
var mdItalic = regexp.MustCompile(`\*([^*]+)\*`)
var mdLink = regexp.MustCompile(`\[([^\]]+)\)\(([^)]+)\)`)
var mdCode = regexp.MustCompile("`([^`]+)`")

// mdToHTML applies a tiny subset of Markdown that covers the common email
// needs (headings, bold, italic, links, inline code).  Anything more
// elaborate should be replaced with a real renderer.
func mdToHTML(input string) string {
	out := html.EscapeString(input)
	out = mdHeading.ReplaceAllStringFunc(out, func(match string) string {
		parts := mdHeading.FindStringSubmatch(match)
		level := len(parts[1])
		return fmt.Sprintf("<h%d>%s</h%d>", level, parts[2], level)
	})
	out = mdBold.ReplaceAllString(out, "<strong>$1</strong>")
	out = mdItalic.ReplaceAllString(out, "<em>$1</em>")
	out = mdCode.ReplaceAllString(out, "<code>$1</code>")
	out = mdLink.ReplaceAllStringFunc(out, func(match string) string {
		parts := mdLink.FindStringSubmatch(match)
		return fmt.Sprintf(`<a href="%s">%s</a`, parts[2], parts[1])
	})
	out = strings.ReplaceAll(out, "\n\n", "</p><p>")
	out = strings.ReplaceAll(out, "\n", "<br/>")
	if !strings.HasPrefix(out, "<h") && !strings.HasPrefix(out, "<p>") {
		out = "<p>" + out + "</p>"
	}
	return out
}

// =============================================================================
// Default i18n dictionary
// =============================================================================

func defaultI18n() map[string]map[string]string {
	return map[string]map[string]string{
		"vi": {
			"hello":             "Xin chào",
			"welcome":           "Chào mừng bạn đến với RINCO",
			"thank_you":         "Cảm ơn quý khách",
			"unsubscribe":       "Hủy đăng ký",
			"view_in_browser":   "Xem trong trình duyệt",
			"your_order":        "Đơn hàng của bạn",
			"reset_password":    "Đặt lại mật khẩu",
			"verification_code": "Mã xác minh của bạn là",
		},
		"en": {
			"hello":             "Hello",
			"welcome":           "Welcome to RINCO",
			"thank_you":         "Thank you",
			"unsubscribe":       "Unsubscribe",
			"view_in_browser":   "View in browser",
			"your_order":        "Your order",
			"reset_password":    "Reset your password",
			"verification_code": "Your verification code is",
		},
	}
}

// =============================================================================
// Tracking helpers
// =============================================================================

// InjectTrackingPixel inserts a 1x1 tracking pixel right before </body>.
func InjectTrackingPixel(htmlBody, pixelURL string) string {
	if htmlBody == "" {
		return htmlBody
	}
	pixel := fmt.Sprintf(`<img src="%s" width="1" height="1" alt="" style="display:none;border:0;"/>`, html.EscapeString(pixelURL))
	if idx := strings.LastIndex(strings.ToLower(htmlBody), "</body>"); idx >= 0 {
		return htmlBody[:idx] + pixel + htmlBody[idx:]
	}
	return htmlBody + pixel
}

var linkRegex = regexp.MustCompile(`<a\s+([^>]*?)href="(https?://[^"]+)"([^>]*)>`)

// RewriteClickLinks rewrites every http(s) link in the supplied HTML so it
// goes through the provided tracking redirector.  Existing attributes are
// preserved.
func RewriteClickLinks(htmlBody, redirectBase string) string {
	if htmlBody == "" || redirectBase == "" {
		return htmlBody
	}
	return linkRegex.ReplaceAllStringFunc(htmlBody, func(match string) string {
		parts := linkRegex.FindStringSubmatch(match)
		target := parts[2]
		encoded := url.QueryEscape(target)
		rewritten := fmt.Sprintf(`<a %shref="%s?url=%s"%s>`, parts[1], redirectBase, encoded, parts[3])
		return rewritten
	})
}

// BuildUnsubscribeHeader builds a List-Unsubscribe value pointing at the
// supplied one-click URL.
func BuildUnsubscribeHeader(oneClickURL, mailto string) string {
	parts := []string{}
	if oneClickURL != "" {
		parts = append(parts, "<"+oneClickURL+">")
	}
	if mailto != "" {
		parts = append(parts, "<mailto:"+mailto+">")
	}
	return strings.Join(parts, ", ")
}