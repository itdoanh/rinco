// Extra tests for email-service render helpers (mdToHTML, defVal, dateFormat, urlSafe, etc).
package render

import (
	"strings"
	"testing"
	"time"
)

func TestMdToHTML_Heading(t *testing.T) {
	got := mdToHTML("# Hello")
	if !strings.Contains(got, "<h1>Hello</h1>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_H2(t *testing.T) {
	got := mdToHTML("## Subhead")
	if !strings.Contains(got, "<h2>Subhead</h2>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_H6(t *testing.T) {
	got := mdToHTML("###### Six")
	if !strings.Contains(got, "<h6>Six</h6>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_Bold(t *testing.T) {
	got := mdToHTML("**bold**")
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_Italic(t *testing.T) {
	got := mdToHTML("*italic*")
	if !strings.Contains(got, "<em>italic</em>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_Code(t *testing.T) {
	got := mdToHTML("`code`")
	if !strings.Contains(got, "<code>code</code>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_Link(t *testing.T) {
	got := mdToHTML("[text](https://example.com)")
	if !strings.Contains(got, `<a href="https://example.com">text</a>`) {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_Paragraph(t *testing.T) {
	got := mdToHTML("hello world")
	if !strings.Contains(got, "<p>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_NewlineBreaks(t *testing.T) {
	got := mdToHTML("a\nb")
	if !strings.Contains(got, "<br/>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_ParagraphBreaks(t *testing.T) {
	got := mdToHTML("a\n\nb")
	if !strings.Contains(got, "</p><p>") {
		t.Errorf("got %q", got)
	}
}

func TestMdToHTML_EscapesHTML(t *testing.T) {
	got := mdToHTML("<script>")
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("got %q", got)
	}
}

func TestDefVal_Nil(t *testing.T) {
	if defVal("fallback", nil) != "fallback" {
		t.Error("expected fallback for nil")
	}
}

func TestDefVal_EmptyString(t *testing.T) {
	if defVal("fallback", "") != "fallback" {
		t.Error("expected fallback for empty string")
	}
}

func TestDefVal_Valid(t *testing.T) {
	if defVal("fallback", "value") != "value" {
		t.Error("expected value")
	}
}

func TestDateFormat_Time(t *testing.T) {
	t1 := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	got := dateFormat("2006-01-02", t1)
	if got != "2025-01-15" {
		t.Errorf("got %s", got)
	}
}

func TestDateFormat_String(t *testing.T) {
	got := dateFormat("2006-01-02", "2025-01-15T10:30:00Z")
	if got != "2025-01-15" {
		t.Errorf("got %s", got)
	}
}

func TestDateFormat_BadString(t *testing.T) {
	got := dateFormat("2006-01-02", "not-a-date")
	if got != "not-a-date" {
		t.Errorf("got %s", got)
	}
}

func TestDateFormat_EmptyLayout(t *testing.T) {
	got := dateFormat("", "2025-01-15T10:30:00Z")
	if got != "2025-01-15T10:30:00Z" {
		t.Errorf("got %s", got)
	}
}

func TestDateFormat_Default(t *testing.T) {
	got := dateFormat("2006-01-02", 123)
	if got != "123" {
		t.Errorf("got %s", got)
	}
}

func TestUrlSafe_String(t *testing.T) {
	got := urlSafe("hello world")
	if got != "hello+world" {
		t.Errorf("got %s", got)
	}
}

func TestUrlSafe_Map(t *testing.T) {
	got := urlSafe(map[string]string{"a": "1", "b": "2"})
	if !strings.Contains(got, "a=1") || !strings.Contains(got, "b=2") {
		t.Errorf("got %s", got)
	}
}

func TestUrlSafe_Other(t *testing.T) {
	got := urlSafe(123)
	if got == "" {
		t.Error("expected non-empty")
	}
}

func TestInjectTrackingPixel_BeforeBody(t *testing.T) {
	body := "<p>hi</p></body>"
	got := InjectTrackingPixel(body, "https://x.com/p.gif")
	if !strings.Contains(got, "x.com/p.gif") {
		t.Errorf("got %s", got)
	}
	if strings.Index(got, "p.gif") > strings.Index(got, "</body>") {
		t.Error("pixel should be before </body>")
	}
}

func TestInjectTrackingPixel_NoBody(t *testing.T) {
	body := "<p>hi</p>"
	got := InjectTrackingPixel(body, "https://x.com/p.gif")
	if !strings.Contains(got, "x.com/p.gif") {
		t.Errorf("got %s", got)
	}
}

func TestInjectTrackingPixel_Empty(t *testing.T) {
	if got := InjectTrackingPixel("", "x"); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestRewriteClickLinks_Empty(t *testing.T) {
	if got := RewriteClickLinks("", "base"); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestRewriteClickLinks_EmptyBase(t *testing.T) {
	body := `<a href="https://x.com">x</a>`
	if got := RewriteClickLinks(body, ""); got != body {
		t.Errorf("got %q", got)
	}
}

func TestRewriteClickLinks_HTTP(t *testing.T) {
	body := `<a href="https://example.com">link</a>`
	got := RewriteClickLinks(body, "https://track.com")
	if !strings.Contains(got, "track.com") {
		t.Errorf("got %q", got)
	}
}

func TestBuildUnsubscribeHeader_OneClick(t *testing.T) {
	got := BuildUnsubscribeHeader("https://unsub.com", "")
	if got != "<https://unsub.com>" {
		t.Errorf("got %s", got)
	}
}

func TestBuildUnsubscribeHeader_Mailto(t *testing.T) {
	got := BuildUnsubscribeHeader("", "unsub@x.com")
	if got != "<mailto:unsub@x.com>" {
		t.Errorf("got %s", got)
	}
}

func TestBuildUnsubscribeHeader_Both(t *testing.T) {
	got := BuildUnsubscribeHeader("https://unsub.com", "unsub@x.com")
	if !strings.Contains(got, "https://unsub.com") {
		t.Errorf("got %s", got)
	}
	if !strings.Contains(got, "mailto:unsub@x.com") {
		t.Errorf("got %s", got)
	}
}

func TestBuildUnsubscribeHeader_Empty(t *testing.T) {
	if got := BuildUnsubscribeHeader("", ""); got != "" {
		t.Errorf("got %s", got)
	}
}

func TestDefaultI18n_Vietnamese(t *testing.T) {
	dict := defaultI18n()
	if dict["vi"]["hello"] != "Xin chào" {
		t.Errorf("got %s", dict["vi"]["hello"])
	}
}

func TestDefaultI18n_English(t *testing.T) {
	dict := defaultI18n()
	if dict["en"]["hello"] != "Hello" {
		t.Errorf("got %s", dict["en"]["hello"])
	}
}

func TestDefaultI18n_HasKeys(t *testing.T) {
	dict := defaultI18n()
	for _, lang := range []string{"vi", "en"} {
		for _, key := range []string{"hello", "welcome", "thank_you", "unsubscribe", "view_in_browser", "your_order", "reset_password", "verification_code"} {
			if _, ok := dict[lang][key]; !ok {
				t.Errorf("missing %s.%s", lang, key)
			}
		}
	}
}

func TestTfuncI18n_Vietnamese(t *testing.T) {
	e := &Engine{i18n: defaultI18n()}
	if got := e.tfuncI18n("hello"); got != "Xin chào" {
		t.Errorf("got %s", got)
	}
}

func TestTfuncI18n_English(t *testing.T) {
	e := &Engine{i18n: defaultI18n()}
	if got := e.tfuncI18n("hello", "en"); got != "Hello" {
		t.Errorf("got %s", got)
	}
}

func TestTfuncI18n_UnknownKey(t *testing.T) {
	e := &Engine{i18n: defaultI18n()}
	if got := e.tfuncI18n("nonexistent"); got != "nonexistent" {
		t.Errorf("got %s", got)
	}
}

func TestTfuncI18n_UnknownLang(t *testing.T) {
	e := &Engine{i18n: defaultI18n()}
	if got := e.tfuncI18n("hello", "fr"); got != "Hello" {
		t.Errorf("got %s", got)
	}
}

func TestNewEngine_ParseError(t *testing.T) {
	_, err := NewEngine("{{.invalid syntax")
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestNewEngine_OK(t *testing.T) {
	e, err := NewEngine("Hello {{.Name}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Render(map[string]any{"Name": "World"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Hello World" {
		t.Errorf("got %q", got)
	}
}

func TestEngine_Render_Nil(t *testing.T) {
	var e *Engine
	_, err := e.Render(nil)
	if err == nil {
		t.Error("expected error for nil engine")
	}
}

func TestEngine_Render_NoTemplate(t *testing.T) {
	e := &Engine{}
	_, err := e.Render(nil)
	if err == nil {
		t.Error("expected error for missing template")
	}
}
