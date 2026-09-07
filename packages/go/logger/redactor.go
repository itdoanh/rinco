// Package logger - PII/secret redactor.
//
// Tự động phát hiện và thay thế các thông tin nhạy cảm trong logs:
//   - Email addresses
//   - Phone numbers
//   - Credit card numbers
//   - API tokens / access tokens
//   - Password fields
//   - JWT tokens
//   - IPv4/IPv6 addresses
//   - MAC addresses
//   - Custom regex patterns
//
// Mặc định thay thế bằng SHA-256 prefix để có thể nhận diện nhưng không khôi phục.
//
// Usage:
//
//	redactor := logger.NewRedactor(&logger.RedactorConfig{
//	    HashPrefix: "[REDACTED:",
//	    HashSuffix: "]",
//	})
//	logger.Init(ctx, logger.Config{
//	    Redactor: redactor,
//	})
package logger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// RedactorConfig cấu hình redactor.
type RedactorConfig struct {
	// HashPrefix là phần đầu của giá trị thay thế (vd: "[REDACTED:")
	HashPrefix string

	// HashSuffix là phần cuối của giá trị thay thế (vd: "]")
	HashSuffix string

	// DisableEmail tắt detection email
	DisableEmail bool

	// DisablePhone tắt detection phone
	DisablePhone bool

	// DisableToken tắt detection token/JWT
	DisableToken bool

	// DisableCreditCard tắt detection credit card
	DisableCreditCard bool

	// DisableIPAddress tắt detection IP address
	DisableIPAddress bool

	// DisableMACAddress tắt detection MAC address
	DisableMACAddress bool

	// DisablePassword tắt detection password fields
	DisablePassword bool

	// CustomPatterns là danh sách regex tùy chỉnh để redact
	CustomPatterns []*RedactionPattern

	// FieldNames là tên các field mà nội dung cần redact hoàn toàn
	FieldNames []string

	// MaskAllValuesOfMaskFields thay thế toàn bộ value của các field sensitive
	MaskAllValuesOfMaskFields bool
}

// DefaultRedactorConfig trả về config mặc định.
func DefaultRedactorConfig() *RedactorConfig {
	return &RedactorConfig{
		HashPrefix: "[REDACTED:",
		HashSuffix: "]",
		FieldNames: []string{
			"password",
			"passwd",
			"pwd",
			"secret",
			"api_key",
			"apikey",
			"access_token",
			"refresh_token",
			"authorization",
			"auth",
			"cookie",
			"session_id",
			"csrf_token",
		},
		MaskAllValuesOfMaskFields: true,
	}
}

// RedactionPattern định nghĩa một regex pattern tùy chỉnh.
type RedactionPattern struct {
	// Name để nhận diện pattern
	Name string

	// Pattern là regex
	Pattern *regexp.Regexp

	// Replacement là giá trị thay thế (mặc định dùng hash)
	Replacement string

	// Validate hàm kiểm tra nội dung có hợp lệ trước khi redact
	Validate func(match string) bool
}

// Redactor là handler wrap slog.Handler với khả năng redact PII.
type Redactor struct {
	config        *RedactorConfig
	emailRegex    *regexp.Regexp
	phoneRegex    *regexp.Regexp
	ccRegex       *regexp.Regexp
	ipV4Regex     *regexp.Regexp
	ipV6Regex     *regexp.Regexp
	macRegex      *regexp.Regexp
	tokenRegex    *regexp.Regexp
	jwtRegex      *regexp.Regexp
	passwordRegex *regexp.Regexp
}

// NewRedactor tạo một redactor mới với config.
func NewRedactor(cfg *RedactorConfig) *Redactor {
	if cfg == nil {
		cfg = DefaultRedactorConfig()
	}
	if cfg.HashPrefix == "" {
		cfg.HashPrefix = "[REDACTED:"
	}
	if cfg.HashSuffix == "" {
		cfg.HashSuffix = "]"
	}

	r := &Redactor{
		config: cfg,
	}

	// Email regex
	if !cfg.DisableEmail {
		r.emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	}

	// Phone regex (international formats)
	if !cfg.DisablePhone {
		r.phoneRegex = regexp.MustCompile(`\b\+?[0-9]{1,4}[ \-]?\(?[0-9]{1,4}\)?[ \-]?[0-9]{3,4}[ \-]?[0-9]{3,4}\b`)
	}

	// Credit card regex (basic - Luhn check)
	if !cfg.DisableCreditCard {
		r.ccRegex = regexp.MustCompile(`\b(?:\d[ \-]?){13,19}\b`)
	}

	// IPv4
	if !cfg.DisableIPAddress {
		r.ipV4Regex = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
		r.ipV6Regex = regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`)
	}

	// MAC address
	if !cfg.DisableMACAddress {
		r.macRegex = regexp.MustCompile(`\b(?:[0-9a-fA-F]{2}[:\-]){5}[0-9a-fA-F]{2}\b`)
	}

	// Token / JWT
	if !cfg.DisableToken {
		r.tokenRegex = regexp.MustCompile(`\b[A-Za-z0-9_\-]{32,}\b`)
		r.jwtRegex = regexp.MustCompile(`\bey[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\b`)
	}

	// Password in URL or query
	if !cfg.DisablePassword {
		r.passwordRegex = regexp.MustCompile(`(?i)(password|passwd|pwd|secret)\s*[:=]\s*([^\s&;]+)`)
	}

	return r
}

// NewRedactionHandler tạo slog.Handler wrapper với redactor.
func NewRedactionHandler(inner slog.Handler, redactor *Redactor) slog.Handler {
	return &redactionHandler{
		inner:    inner,
		redactor: redactor,
	}
}

type redactionHandler struct {
	inner    slog.Handler
	redactor *Redactor
}

func (h *redactionHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *redactionHandler) Handle(ctx context.Context, record slog.Record) error {
	// Tạo record mới với attrs đã redact
	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)

	record.Attrs(func(attr slog.Attr) bool {
		// Kiểm tra field name có cần redact hoàn toàn không
		if h.shouldMaskField(attr.Key) {
			newRecord.AddAttrs(slog.String(attr.Key, h.redactor.config.HashPrefix+"field"+h.redactor.config.HashSuffix))
			return true
		}

		// Redact giá trị string
		newAttr := h.redactAttr(attr)
		newRecord.AddAttrs(newAttr)
		return true
	})

	return h.inner.Handle(ctx, newRecord)
}

func (h *redactionHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		newAttrs[i] = h.redactAttr(attr)
	}
	return &redactionHandler{
		inner:    h.inner.WithAttrs(newAttrs),
		redactor: h.redactor,
	}
}

func (h *redactionHandler) WithGroup(name string) slog.Handler {
	return &redactionHandler{
		inner:    h.inner.WithGroup(name),
		redactor: h.redactor,
	}
}

// shouldMaskField kiểm tra field có cần redact toàn bộ value không.
func (h *redactionHandler) shouldMaskField(name string) bool {
	if !h.redactor.config.MaskAllValuesOfMaskFields {
		return false
	}
	lowerName := strings.ToLower(name)
	for _, f := range h.redactor.config.FieldNames {
		if strings.Contains(lowerName, strings.ToLower(f)) {
			return true
		}
	}
	return false
}

// redactAttr redact giá trị của một attribute.
func (h *redactionHandler) redactAttr(attr slog.Attr) slog.Attr {
	// Mask hoàn toàn nếu field name match
	if h.shouldMaskField(attr.Key) {
		return slog.String(attr.Key, h.redactor.config.HashPrefix+"field"+h.redactor.config.HashSuffix)
	}

	switch v := attr.Value.Any().(type) {
	case string:
		return slog.String(attr.Key, h.redactor.RedactString(v))
	case error:
		return slog.String(attr.Key, h.redactor.RedactString(v.Error()))
	default:
		return attr
	}
}

// RedactString redact một string bằng cách thay thế PII.
func (r *Redactor) RedactString(s string) string {
	if s == "" {
		return s
	}

	result := s

	if r.emailRegex != nil {
		result = r.emailRegex.ReplaceAllStringFunc(result, func(match string) string {
			return r.hashValue("email", match)
		})
	}

	if r.phoneRegex != nil {
		result = r.phoneRegex.ReplaceAllStringFunc(result, func(match string) string {
			// Verify this is actually a phone number (not a random number sequence)
			if isLikelyPhone(match) {
				return r.hashValue("phone", match)
			}
			return match
		})
	}

	if r.ccRegex != nil {
		result = r.ccRegex.ReplaceAllStringFunc(result, func(match string) string {
			if isValidCreditCard(match) {
				return r.hashValue("cc", match)
			}
			return match
		})
	}

	if r.ipV4Regex != nil {
		result = r.ipV4Regex.ReplaceAllStringFunc(result, func(match string) string {
			return r.hashValue("ipv4", match)
		})
	}

	if r.ipV6Regex != nil {
		result = r.ipV6Regex.ReplaceAllStringFunc(result, func(match string) string {
			return r.hashValue("ipv6", match)
		})
	}

	if r.macRegex != nil {
		result = r.macRegex.ReplaceAllStringFunc(result, func(match string) string {
			return r.hashValue("mac", match)
		})
	}

	if r.jwtRegex != nil {
		result = r.jwtRegex.ReplaceAllStringFunc(result, func(match string) string {
			return r.hashValue("jwt", match)
		})
	}

	if r.tokenRegex != nil {
		result = r.tokenRegex.ReplaceAllStringFunc(result, func(match string) string {
			// Skip if already redacted (looks like hash)
			if strings.Contains(match, r.config.HashPrefix) {
				return match
			}
			// Skip if it's a UUID-like format
			if isUUIDFormat(match) {
				return match
			}
			return r.hashValue("token", match)
		})
	}

	if r.passwordRegex != nil {
		result = r.passwordRegex.ReplaceAllStringFunc(result, func(match string) string {
			// Replace password value
			parts := r.passwordRegex.FindStringSubmatch(match)
			if len(parts) >= 3 {
				return fmt.Sprintf("%s=%s", parts[1], r.hashValue("password", parts[2]))
			}
			return match
		})
	}

	// Apply custom patterns
	for _, p := range r.config.CustomPatterns {
		result = p.Pattern.ReplaceAllStringFunc(result, func(match string) string {
			if p.Validate != nil && !p.Validate(match) {
				return match
			}
			if p.Replacement != "" {
				return p.Replacement
			}
			return r.hashValue(p.Name, match)
		})
	}

	return result
}

// hashValue hash một giá trị và trả về với prefix/suffix.
func (r *Redactor) hashValue(kind, value string) string {
	h := sha256.Sum256([]byte(value))
	hexHash := hex.EncodeToString(h[:])
	// Use first 8 chars for brevity
	shortHash := hexHash[:8]
	return fmt.Sprintf("%s%s:%s%s", r.config.HashPrefix, kind, shortHash, r.config.HashSuffix)
}

// isLikelyPhone kiểm tra chuỗi có phải số điện thoại không.
func isLikelyPhone(s string) bool {
	digits := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	return digits >= 7 && digits <= 15
}

// isValidCreditCard kiểm tra bằng Luhn algorithm.
func isValidCreditCard(s string) bool {
	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "-", "")
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}

	sum := 0
	alternate := false
	for i := len(cleaned) - 1; i >= 0; i-- {
		d := int(cleaned[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if alternate {
			d *= 2
			if d > 9 {
				d = (d / 10) + (d % 10)
			}
		}
		sum += d
		alternate = !alternate
	}
	return sum%10 == 0
}

// isUUIDFormat kiểm tra có phải UUID format không.
func isUUIDFormat(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// AddCustomPattern thêm một custom pattern.
func (r *Redactor) AddCustomPattern(p *RedactionPattern) error {
	if p.Pattern == nil {
		return fmt.Errorf("pattern is nil")
	}
	r.config.CustomPatterns = append(r.config.CustomPatterns, p)
	return nil
}

// AddFieldName thêm field name cần redact.
func (r *Redactor) AddFieldName(name string) {
	r.config.FieldNames = append(r.config.FieldNames, name)
}
