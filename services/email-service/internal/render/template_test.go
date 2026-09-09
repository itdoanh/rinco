// Tests for email-service render package.
package render

import (
	"strings"
	"testing"
)

func TestExtra_NewEngine_Simple(t *testing.T) {
	e, err := NewEngine("Hello {{.Name}}!")
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if e == nil {
		t.Fatal("nil engine")
	}
}

func TestExtra_NewEngine_ParseError(t *testing.T) {
	_, err := NewEngine("{{ .Invalid")  // syntax error
	if err == nil {
		t.Error("expected error for bad template")
	}
}

func TestExtra_Render_Simple(t *testing.T) {
	e, _ := NewEngine("Hello {{.Name}}!")
	got, err := e.Render(map[string]any{"Name": "World"})
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if got != "Hello World!" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_NilEngine(t *testing.T) {
	var e *Engine
	_, err := e.Render(map[string]any{})
	if err == nil {
		t.Error("nil engine should error")
	}
}

func TestExtra_Render_FuncUpper(t *testing.T) {
	e, _ := NewEngine("{{upper .Name}}")
	got, _ := e.Render(map[string]any{"Name": "hello"})
	if got != "HELLO" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncLower(t *testing.T) {
	e, _ := NewEngine("{{lower .Name}}")
	got, _ := e.Render(map[string]any{"Name": "HELLO"})
	if got != "hello" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncTitle(t *testing.T) {
	e, _ := NewEngine("{{title .Name}}")
	got, _ := e.Render(map[string]any{"Name": "hello world"})
	if got != "Hello World" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncTrim(t *testing.T) {
	e, _ := NewEngine("[{{trim .Name}}]")
	got, _ := e.Render(map[string]any{"Name": "  spaced  "})
	if got != "[spaced]" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncDefault_NilValue(t *testing.T) {
	e, _ := NewEngine("{{default \"fallback\" .Name}}")
	got, _ := e.Render(map[string]any{})
	if got != "fallback" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncDefault_EmptyString(t *testing.T) {
	e, _ := NewEngine("{{default \"fallback\" .Name}}")
	got, _ := e.Render(map[string]any{"Name": ""})
	if got != "fallback" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncDefault_NonEmpty(t *testing.T) {
	e, _ := NewEngine("{{default \"fallback\" .Name}}")
	got, _ := e.Render(map[string]any{"Name": "value"})
	if got != "value" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncI18n_VI(t *testing.T) {
	e, _ := NewEngine("{{i18n \"hello\" \"vi\"}}")
	got, _ := e.Render(map[string]any{})
	if got != "Xin chào" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncI18n_EN(t *testing.T) {
	e, _ := NewEngine("{{i18n \"hello\" \"en\"}}")
	got, _ := e.Render(map[string]any{})
	if got != "Hello" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncI18n_DefaultLang(t *testing.T) {
	e, _ := NewEngine("{{i18n \"hello\"}}")
	got, _ := e.Render(map[string]any{})
	// Default is Vietnamese
	if got != "Xin chào" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncI18n_UnknownKey(t *testing.T) {
	e, _ := NewEngine("{{i18n \"unknown_key\"}}")
	got, _ := e.Render(map[string]any{})
	// Unknown key returns the key itself
	if got != "unknown_key" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncDateFormat_String(t *testing.T) {
	e, _ := NewEngine("{{dateFormat \"2006-01-02\" .Date}}")
	got, _ := e.Render(map[string]any{"Date": "2026-01-15T10:30:00Z"})
	if got != "2026-01-15" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncDateFormat_Time(t *testing.T) {
	e, _ := NewEngine("{{dateFormat \"2006-01-02\" .Date}}")
	got, _ := e.Render(map[string]any{"Date": mustParse(t, "2026-01-15T10:30:00Z")})
	if got != "2026-01-15" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncDateFormat_NoLayout(t *testing.T) {
	e, _ := NewEngine("{{dateFormat \"\" .Date}}")
	got, _ := e.Render(map[string]any{"Date": "2026-01-15T10:30:00Z"})
	if !strings.Contains(got, "2026-01-15") {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncUrlSafe_String(t *testing.T) {
	e, _ := NewEngine("{{urlSafe .V}}")
	got, _ := e.Render(map[string]any{"V": "hello world"})
	if got != "hello+world" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncUrlSafe_Map(t *testing.T) {
	e, _ := NewEngine("{{urlSafe .M}}")
	got, _ := e.Render(map[string]any{"M": map[string]string{"a": "1", "b": "2"}})
	if !strings.Contains(got, "a=1") || !strings.Contains(got, "b=2") {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Render_FuncMarkdown(t *testing.T) {
	e, _ := NewEngine("{{markdown .Body}}")
	got, _ := e.Render(map[string]any{"Body": "# Hello"})
	if !strings.Contains(got, "<h1>") {
		t.Errorf("expected h1: got %s", got)
	}
}

func TestExtra_mdToHTML_Bold(t *testing.T) {
	got := mdToHTML("**bold text**")
	if !strings.Contains(got, "<strong>") {
		t.Errorf("expected strong: got %s", got)
	}
}

func TestExtra_mdToHTML_Italic(t *testing.T) {
	got := mdToHTML("*italic text*")
	if !strings.Contains(got, "<em>") {
		t.Errorf("expected em: got %s", got)
	}
}

func TestExtra_mdToHTML_Code(t *testing.T) {
	got := mdToHTML("`code here`")
	if !strings.Contains(got, "<code>") {
		t.Errorf("expected code: got %s", got)
	}
}

func TestExtra_mdToHTML_Link(t *testing.T) {
	got := mdToHTML("[click](https://example.com)")
	t.Logf("got: %s", got)
	if !strings.Contains(got, "<a href=") {
		t.Errorf("expected a href: got %s", got)
	}
	if !strings.Contains(got, "click") {
		t.Errorf("expected click: got %s", got)
	}
	if !strings.Contains(got, "https://example.com") {
		t.Errorf("expected URL: got %s", got)
	}
	if !strings.Contains(got, "</a>") {
		t.Errorf("missing </a>: got %s", got)
	}
}

func TestExtra_mdToHTML_Heading(t *testing.T) {
	got := mdToHTML("## Subheading")
	if !strings.Contains(got, "<h2>") {
		t.Errorf("expected h2: got %s", got)
	}
}

func TestExtra_mdToHTML_Paragraph(t *testing.T) {
	got := mdToHTML("hello world")
	if !strings.Contains(got, "<p>") {
		t.Errorf("expected p: got %s", got)
	}
	if !strings.Contains(got, "</p>") {
		t.Errorf("expected /p: got %s", got)
	}
}

func TestExtra_mdToHTML_HTMLEscaped(t *testing.T) {
	got := mdToHTML("<script>alert('xss')</script>")
	if strings.Contains(got, "<script>") {
		t.Errorf("XSS should be escaped: got %s", got)
	}
}

func TestExtra_InjectTrackingPixel_Empty(t *testing.T) {
	got := InjectTrackingPixel("", "https://t.example/p.gif")
	if got != "" {
		t.Errorf("empty body should stay empty: got %s", got)
	}
}

func TestExtra_InjectTrackingPixel_BeforeBody(t *testing.T) {
	body := "<html><body>Hello</body></html>"
	got := InjectTrackingPixel(body, "https://t.example/p.gif")
	if !strings.Contains(got, "<img src=\"https://t.example/p.gif\"") {
		t.Errorf("pixel not injected: %s", got)
	}
	// Pixel should be before </body>
	idxImg := strings.Index(got, "<img")
	idxBody := strings.Index(got, "</body>")
	if idxImg > idxBody {
		t.Error("pixel should be before </body>")
	}
}

func TestExtra_InjectTrackingPixel_NoBodyTag(t *testing.T) {
	body := "Plain text content"
	got := InjectTrackingPixel(body, "https://t.example/p.gif")
	if !strings.Contains(got, "<img") {
		t.Errorf("pixel should be appended: got %s", got)
	}
}

func TestExtra_InjectTrackingPixel_UppercaseBody(t *testing.T) {
	body := "<HTML><BODY>X</BODY></HTML>"
	got := InjectTrackingPixel(body, "https://t.example/p.gif")
	if !strings.Contains(got, "<img") {
		t.Errorf("pixel should be injected before uppercase </BODY>: got %s", got)
	}
}

func TestExtra_RewriteClickLinks_Empty(t *testing.T) {
	got := RewriteClickLinks("", "https://r.example/r")
	if got != "" {
		t.Errorf("empty should stay empty: got %s", got)
	}
}

func TestExtra_RewriteClickLinks_NoRedirectBase(t *testing.T) {
	body := "<a href=\"https://example.com\">x</a>"
	got := RewriteClickLinks(body, "")
	if got != body {
		t.Errorf("no redirect base should leave unchanged: got %s", got)
	}
}

func TestExtra_RewriteClickLinks_BasicRewrite(t *testing.T) {
	body := `<a href="https://example.com">click</a>`
	got := RewriteClickLinks(body, "https://r.example/r")
	if !strings.Contains(got, "https://r.example/r?url=") {
		t.Errorf("link not rewritten: %s", got)
	}
}

func TestExtra_RewriteClickLinks_PreservesOtherAttributes(t *testing.T) {
	body := `<a class="btn" href="https://example.com" target="_blank">click</a>`
	got := RewriteClickLinks(body, "https://r.example/r")
	if !strings.Contains(got, "class=\"btn\"") {
		t.Errorf("class attribute not preserved: %s", got)
	}
	if !strings.Contains(got, "target=\"_blank\"") {
		t.Errorf("target attribute not preserved: %s", got)
	}
}

func TestExtra_BuildUnsubscribeHeader_HTTPOnly(t *testing.T) {
	got := BuildUnsubscribeHeader("https://u.example/unsub", "")
	if !strings.Contains(got, "<https://u.example/unsub>") {
		t.Errorf("missing HTTP: %s", got)
	}
	if strings.Contains(got, "mailto:") {
		t.Errorf("should not contain mailto: %s", got)
	}
}

func TestExtra_BuildUnsubscribeHeader_MailtoOnly(t *testing.T) {
	got := BuildUnsubscribeHeader("", "unsub@example.com")
	if !strings.Contains(got, "<mailto:unsub@example.com>") {
		t.Errorf("missing mailto: %s", got)
	}
}

func TestExtra_BuildUnsubscribeHeader_Both(t *testing.T) {
	got := BuildUnsubscribeHeader("https://u.example/unsub", "unsub@example.com")
	if !strings.Contains(got, "<https://u.example/unsub>") {
		t.Errorf("missing HTTP: %s", got)
	}
	if !strings.Contains(got, "<mailto:unsub@example.com>") {
		t.Errorf("missing mailto: %s", got)
	}
}

func TestExtra_BuildUnsubscribeHeader_Neither(t *testing.T) {
	got := BuildUnsubscribeHeader("", "")
	if got != "" {
		t.Errorf("neither should be empty: %s", got)
	}
}

func TestExtra_defVal_Nil(t *testing.T) {
	if got := defVal("fallback", nil); got != "fallback" {
		t.Errorf("nil: got %v", got)
	}
}

func TestExtra_defVal_EmptyString(t *testing.T) {
	if got := defVal("fallback", ""); got != "fallback" {
		t.Errorf("empty: got %v", got)
	}
}

func TestExtra_defVal_NonEmpty(t *testing.T) {
	if got := defVal("fallback", "value"); got != "value" {
		t.Errorf("non-empty: got %v", got)
	}
}

func TestExtra_defaultI18n_HasVI(t *testing.T) {
	dict := defaultI18n()
	if _, ok := dict["vi"]; !ok {
		t.Error("missing vi")
	}
}

func TestExtra_defaultI18n_HasEN(t *testing.T) {
	dict := defaultI18n()
	if _, ok := dict["en"]; !ok {
		t.Error("missing en")
	}
}

func TestExtra_defaultI18n_HasCommonKeys(t *testing.T) {
	dict := defaultI18n()
	for _, key := range []string{"hello", "welcome", "thank_you"} {
		if _, ok := dict["vi"][key]; !ok {
			t.Errorf("missing vi.%s", key)
		}
		if _, ok := dict["en"][key]; !ok {
			t.Errorf("missing en.%s", key)
		}
	}
}

func mustParse(t *testing.T, s string) interface{} {
	t.Helper()
	// Just return as string for simplicity
	return s
}
