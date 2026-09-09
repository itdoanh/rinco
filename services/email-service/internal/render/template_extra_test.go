// Extra tests for email-service render package.
package render

import (
	"strings"
	"testing"
)

func TestExtraMdToHTML_Heading(t *testing.T) {
	got := mdToHTML("# Heading 1")
	if !strings.Contains(got, "<h1>") {
		t.Errorf("expected h1, got %s", got)
	}
}

func TestExtraMdToHTML_H2(t *testing.T) {
	got := mdToHTML("## H2")
	if !strings.Contains(got, "<h2>") {
		t.Errorf("expected h2, got %s", got)
	}
}

func TestExtraMdToHTML_H3(t *testing.T) {
	got := mdToHTML("### H3")
	if !strings.Contains(got, "<h3>") {
		t.Errorf("expected h3, got %s", got)
	}
}

func TestExtraMdToHTML_Bold(t *testing.T) {
	got := mdToHTML("**bold**")
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Errorf("expected bold, got %s", got)
	}
}

func TestExtraMdToHTML_Italic(t *testing.T) {
	got := mdToHTML("*italic*")
	if !strings.Contains(got, "<em>italic</em>") {
		t.Errorf("expected italic, got %s", got)
	}
}

func TestExtraMdToHTML_Link(t *testing.T) {
	got := mdToHTML("[text](https://example.com)")
	if !strings.Contains(got, `<a href="https://example.com">text</a>`) {
		t.Errorf("expected link, got %s", got)
	}
}

func TestExtraMdToHTML_Code(t *testing.T) {
	got := mdToHTML("`code`")
	if !strings.Contains(got, "<code>code</code>") {
		t.Errorf("expected code, got %s", got)
	}
}

func TestExtraMdToHTML_ParagraphWrap(t *testing.T) {
	got := mdToHTML("hello")
	if !strings.HasPrefix(got, "<p>") || !strings.HasSuffix(got, "</p>") {
		t.Errorf("expected <p>...</p>: %s", got)
	}
}

func TestExtraMdToHTML_NoParagraphOnHeading(t *testing.T) {
	got := mdToHTML("# H")
	if !strings.HasPrefix(got, "<h1>") {
		t.Errorf("heading should not be wrapped: %s", got)
	}
}

func TestExtraMdToHTML_LineBreaks(t *testing.T) {
	got := mdToHTML("a\nb")
	if !strings.Contains(got, "<br/>") {
		t.Errorf("expected br: %s", got)
	}
}

func TestExtraMdToHTML_ParagraphBreaks(t *testing.T) {
	got := mdToHTML("a\n\nb")
	if !strings.Contains(got, "</p><p>") {
		t.Errorf("expected para break: %s", got)
	}
}

func TestExtraMdToHTML_Escape(t *testing.T) {
	got := mdToHTML("<script>")
	if strings.Contains(got, "<script>") {
		t.Errorf("should escape: %s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped: %s", got)
	}
}

func TestExtraDefVal_Nil(t *testing.T) {
	if got := defVal("fallback", nil); got != "fallback" {
		t.Errorf("got %v", got)
	}
}

func TestExtraDefVal_Empty(t *testing.T) {
	if got := defVal("fallback", ""); got != "fallback" {
		t.Errorf("got %v", got)
	}
}

func TestExtraDefVal_NonEmpty(t *testing.T) {
	if got := defVal("fallback", "value"); got != "value" {
		t.Errorf("got %v", got)
	}
}

func TestExtraUrlSafe_String(t *testing.T) {
	got := urlSafe("hello world")
	if !strings.Contains(got, "hello") {
		t.Errorf("got %s", got)
	}
}

func TestExtraUrlSafe_Map(t *testing.T) {
	m := map[string]string{"key": "value"}
	got := urlSafe(m)
	if !strings.Contains(got, "key=value") {
		t.Errorf("got %s", got)
	}
}

func TestExtraUrlSafe_Default(t *testing.T) {
	got := urlSafe(42)
	if !strings.Contains(got, "42") {
		t.Errorf("got %s", got)
	}
}

func TestExtraDateFormat_Time(t *testing.T) {
	got := dateFormat("2006-01-02", "2024-01-15T00:00:00Z")
	if got != "2024-01-15" {
		t.Errorf("got %s", got)
	}
}

func TestExtraDateFormat_String(t *testing.T) {
	got := dateFormat("2006-01-02", "2024-01-15T00:00:00Z")
	if got != "2024-01-15" {
		t.Errorf("got %s", got)
	}
}

func TestExtraInjectTrackingPixel_Empty(t *testing.T) {
	got := InjectTrackingPixel("", "http://x.com/px")
	if got != "" {
		t.Errorf("got %s", got)
	}
}

func TestExtraInjectTrackingPixel_BeforeBody(t *testing.T) {
	body := "<html><body>Hi</body></html>"
	got := InjectTrackingPixel(body, "http://x.com/px")
	if !strings.Contains(got, "http://x.com/px") {
		t.Errorf("missing pixel URL")
	}
	if !strings.Contains(got, "</body>") {
		t.Errorf("body close missing")
	}
}

func TestExtraInjectTrackingPixel_NoBody(t *testing.T) {
	body := "<h1>Hi</h1>"
	got := InjectTrackingPixel(body, "http://x.com/px")
	if !strings.HasSuffix(got, "/>") {
		t.Errorf("pixel not appended: %s", got)
	}
}

func TestExtraRewriteClickLinks_Empty(t *testing.T) {
	if got := RewriteClickLinks("", "http://track"); got != "" {
		t.Errorf("got %s", got)
	}
	if got := RewriteClickLinks("body", ""); got != "body" {
		t.Errorf("got %s", got)
	}
}

func TestExtraRewriteClickLinks_Basic(t *testing.T) {
	body := `<a href="https://example.com/page">link</a>`
	got := RewriteClickLinks(body, "https://track.example.com")
	if strings.Contains(got, "https://example.com/page") {
		t.Errorf("URL not rewritten: %s", got)
	}
	if !strings.Contains(got, "track.example.com") {
		t.Errorf("redirect base missing: %s", got)
	}
}

func TestExtraRewriteClickLinks_PreservesAttrs(t *testing.T) {
	body := `<a class="btn" href="https://example.com/x" rel="noopener">click</a>`
	got := RewriteClickLinks(body, "https://track.example.com")
	if !strings.Contains(got, `class="btn"`) {
		t.Errorf("class missing: %s", got)
	}
}

func TestExtraBuildUnsubscribeHeader_HTTP(t *testing.T) {
	got := BuildUnsubscribeHeader("https://unsub.example.com", "")
	if !strings.Contains(got, "https://unsub.example.com") {
		t.Errorf("URL missing: %s", got)
	}
}

func TestExtraBuildUnsubscribeHeader_Mailto(t *testing.T) {
	got := BuildUnsubscribeHeader("", "unsub@example.com")
	if !strings.Contains(got, "mailto:unsub@example.com") {
		t.Errorf("mailto missing: %s", got)
	}
}

func TestExtraBuildUnsubscribeHeader_Both(t *testing.T) {
	got := BuildUnsubscribeHeader("https://unsub", "unsub@example.com")
	if !strings.Contains(got, ", ") {
		t.Errorf("both should be joined: %s", got)
	}
}

func TestExtraRender_NilEngine(t *testing.T) {
	var e *Engine
	if _, err := e.Render(nil); err == nil {
		t.Error("expected error from nil engine")
	}
}
