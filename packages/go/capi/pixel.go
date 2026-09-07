package capi

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// PixelConfig holds Meta Pixel configuration.
type PixelConfig struct {
	PixelID    string
	Version    string // e.g., "18.0"
	AccessToken string

	// Optional configuration
	AutoConfig bool // Auto-configure based on events
	DebugMode  bool

	// Conversions API integration
	EnableCAPI bool
	Agent      string // User agent string for CAPI
}

// InitScript returns the JavaScript code to initialize Meta Pixel.
func (p *PixelConfig) InitScript() string {
	var buf bytes.Buffer

	// Modern Pixel initialization (fbq function)
	// Note: This outputs the JavaScript that should be embedded in <head>
	// The actual script tag: <script> with src to Pixel loader

	tmpl := template.Must(template.New("pixel").Parse(pixelInitTemplate))
	tmpl.Execute(&buf, map[string]interface{}{
		"PixelID":    p.PixelID,
		"Version":    p.Version,
		"DebugMode":  p.DebugMode,
		"EnableCAPI": p.EnableCAPI,
	})

	return buf.String()
}

// pixelInitTemplate is the JavaScript template for Pixel initialization.
const pixelInitTemplate = `!function(f,b,e,v,n,t,s)
{if(f.fbq)return;n=f.fbq=function(){n.callMethod?
n.callMethod.apply(n,arguments):n.queue.push(arguments)};
if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='{{.Version}}';
n.queue=[];t=b.createElement(e);t.async=!0;
t.src=v;s=b.getElementsByTagName(e)[0];
s.parentNode.insertBefore(t,s)}(window, document,'script',
'https://connect.facebook.net/en_US/fbevents.js');
{{if .DebugMode}}fbq('set', 'autoConfig', false);{{end}}
fbq('init', '{{.PixelID}}');
fbq('track', 'PageView');
{{if .EnableCAPI}}
fbq('track', 'PageView', {}, {eventID: generateEventID()});
{{end}}
`

// GenerateEventScript generates JavaScript for event ID generation.
// This should be included in a <script> tag before fbq calls.
func GenerateEventScript() string {
	return `
window.generateEventID = function() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        var r = Math.random() * 16 | 0;
        var v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
};

// Helper to track events with deduplication
window.trackCAPI = function(eventName, params) {
    var eventID = window.generateEventID ? window.generateEventID() : '';
    if (window.fbq) {
        fbq('track', eventName, params, {eventID: eventID});
    }
    // Also send to server-side CAPI
    fetch('/v1/capi/track', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
            event_name: eventName,
            event_id: eventID,
            event_time: Math.floor(Date.now() / 1000),
            action_source: 'website',
            ...params
        })
    }).catch(function() {});
};
`
}

// TrackPageView generates the fbq call for PageView.
func TrackPageView() string {
	return `fbq('track', 'PageView');`
}

// TrackLead generates the fbq call for Lead event.
func TrackLead(params map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString(`fbq('track', 'Lead'`)
	if len(params) > 0 {
		buf.WriteString(`, `)
		formatMap(&buf, params)
	}
	buf.WriteString(`);`)
	return buf.String()
}

// TrackCompleteRegistration generates the fbq call for CompleteRegistration.
func TrackCompleteRegistration(params map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString(`fbq('track', 'CompleteRegistration'`)
	if len(params) > 0 {
		buf.WriteString(`, `)
		formatMap(&buf, params)
	}
	buf.WriteString(`);`)
	return buf.String()
}

// TrackViewContent generates the fbq call for ViewContent.
func TrackViewContent(params map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString(`fbq('track', 'ViewContent'`)
	if len(params) > 0 {
		buf.WriteString(`, `)
		formatMap(&buf, params)
	}
	buf.WriteString(`);`)
	return buf.String()
}

// TrackCustomEvent generates the fbq call for a custom event.
func TrackCustomEvent(eventName string, params map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString(`fbq('trackCustom', '`)
	buf.WriteString(eventName)
	buf.WriteString(`'`)
	if len(params) > 0 {
		buf.WriteString(`, `)
		formatMap(&buf, params)
	}
	buf.WriteString(`);`)
	return buf.String()
}

func formatMap(buf *bytes.Buffer, m map[string]interface{}) {
	buf.WriteString(`{`)
	var first = true
	for k, v := range m {
		if !first {
			buf.WriteString(`, `)
		}
		first = false
		buf.WriteString(k)
		buf.WriteString(`: `)
		switch val := v.(type) {
		case string:
			buf.WriteString(`"`)
			buf.WriteString(escapeJS(val))
			buf.WriteString(`"`)
		case float64:
			buf.WriteString(fmt.Sprintf(`%f`, val))
		case int:
			buf.WriteString(fmt.Sprintf(`%d`, val))
		case bool:
			if val {
				buf.WriteString(`true`)
			} else {
				buf.WriteString(`false`)
			}
		default:
			buf.WriteString(`null`)
		}
	}
	buf.WriteString(`}`)
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// NoscriptPixel returns the <noscript> pixel tag for environments without JS.
func NoscriptPixel(pixelID string) string {
	return fmt.Sprintf(`<noscript><img height="1" width="1" style="display:none" src="https://www.facebook.com/tr?id=%s&ev=PageView&noscript=1"/></noscript>`, pixelID)
}
