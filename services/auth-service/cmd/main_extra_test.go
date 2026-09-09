// Extra tests for auth-service cmd/main helpers.
package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtraHashPwd_Format(t *testing.T) {
	h, err := hashPwd("hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Errorf("not argon2id: %s", h)
	}
}

func TestExtraHashPwd_Unique(t *testing.T) {
	h1, _ := hashPwd("same")
	h2, _ := hashPwd("same")
	if h1 == h2 {
		t.Error("salts should differ")
	}
}

func TestExtraVerifyPwd_RoundTrip(t *testing.T) {
	h, _ := hashPwd("password123")
	ok, err := verifyPwd("password123", h)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Error("expected match")
	}
}

func TestExtraVerifyPwd_Wrong(t *testing.T) {
	h, _ := hashPwd("password123")
	ok, err := verifyPwd("password_wrong", h)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Error("expected mismatch")
	}
}

func TestExtraVerifyPwd_BadEncoding(t *testing.T) {
	ok, _ := verifyPwd("x", "$bcrypt$v=1$x$y")
	if ok {
		t.Error("expected fail on non-argon2id")
	}
}

func TestExtraVerifyPwd_BadFormat(t *testing.T) {
	ok, _ := verifyPwd("x", "$argon2id$bad$bad$bad")
	if ok {
		t.Error("expected fail on bad format")
	}
}

func TestExtraVerifyPwd_BadSalt(t *testing.T) {
	// Valid format but invalid base64 salt
	hash := "$argon2id$v=19$m=65536,t=3,p=2$!!!not_base64$$"
	ok, _ := verifyPwd("x", hash)
	if ok {
		t.Error("expected fail on bad salt")
	}
}

func TestExtraSha256Hex_Length(t *testing.T) {
	got := sha256Hex("hello")
	if len(got) != 64 {
		t.Errorf("expected 64 hex chars: got %d", len(got))
	}
}

func TestExtraSha256Hex_Deterministic(t *testing.T) {
	a := sha256Hex("test")
	b := sha256Hex("test")
	if a != b {
		t.Error("should match")
	}
}

func TestExtraSha256Hex_Different(t *testing.T) {
	a := sha256Hex("a")
	b := sha256Hex("b")
	if a == b {
		t.Error("different should hash differently")
	}
}

func TestExtraLast4_Short(t *testing.T) {
	if got := last4("ab"); got != "ab" {
		t.Errorf("expected 'ab': got %s", got)
	}
}

func TestExtraLast4_Long(t *testing.T) {
	if got := last4("abcdefgh"); got != "efgh" {
		t.Errorf("expected 'efgh': got %s", got)
	}
}

func TestExtraIsUniqueViolation_True(t *testing.T) {
	err := errWrap("error: duplicate key value violates unique constraint")
	if !isUniqueViolation(err) {
		t.Error("should be true")
	}
}

func TestExtraIsUniqueViolation_False(t *testing.T) {
	err := errWrap("error: other issue")
	if isUniqueViolation(err) {
		t.Error("should be false")
	}
}

func TestExtraIsUniqueViolation_Nil(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Error("nil should be false")
	}
}

type errWrap string

func (e errWrap) Error() string { return string(e) }

func TestExtraFirstNonEmpty_Picks(t *testing.T) {
	if got := firstNonEmpty("", "", "x", "y"); got != "x" {
		t.Errorf("got %s", got)
	}
}

func TestExtraFirstNonEmpty_AllEmpty(t *testing.T) {
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %s", got)
	}
}

func TestExtraNewID_Unique(t *testing.T) {
	a := newID()
	b := newID()
	if a == b {
		t.Error("IDs should be unique")
	}
	if len(a) < 32 {
		t.Errorf("ID too short: %s", a)
	}
}

func TestExtraNullUUID_Empty(t *testing.T) {
	if got := nullUUID(""); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestExtraNullUUID_NonEmpty(t *testing.T) {
	if got := nullUUID("123"); got != "123" {
		t.Errorf("got %v", got)
	}
}

func TestExtraAsString_String(t *testing.T) {
	if got := asString("hello"); got != "hello" {
		t.Errorf("got %s", got)
	}
}

func TestExtraAsString_Other(t *testing.T) {
	if got := asString(42); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtraAsBool_True(t *testing.T) {
	if !asBool(true) {
		t.Error("true should pass")
	}
}

func TestExtraAsBool_False(t *testing.T) {
	if asBool(false) {
		t.Error("false should fail")
	}
}

func TestExtraAsBool_String(t *testing.T) {
	if asBool("hello") {
		t.Error("non-true should fail")
	}
}

func TestExtraGenState_Length(t *testing.T) {
	s, err := genState(24)
	if err != nil {
		t.Fatal(err)
	}
	// 24 bytes -> 48 hex chars
	if len(s) != 48 {
		t.Errorf("expected 48: got %d", len(s))
	}
}

func TestExtraGenPKCE_VerifierLength(t *testing.T) {
	v, _, err := genPKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 64 { // 32 bytes -> 64 hex
		t.Errorf("expected 64: got %d", len(v))
	}
}

func TestExtraGenPKCE_ChallengeURLSafe(t *testing.T) {
	_, c, err := genPKCE()
	if err != nil {
		t.Fatal(err)
	}
	// Should be URL-safe base64 (no '+', '/', '=' padding chars)
	if strings.ContainsAny(c, "+/=") {
		t.Errorf("not URL-safe: %s", c)
	}
	// Should be base64 decodable
	if _, err := base64.RawURLEncoding.DecodeString(c); err != nil {
		t.Errorf("decoding failed: %v", err)
	}
}

func TestExtraJsonErr(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := jsonErr(c, 400, "TEST", "test msg")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

// helper: ad-hoc error provider
func TestExtraReadRand(t *testing.T) {
	b := make([]byte, 16)
	n, err := readRand(b)
	if err != nil {
		t.Fatalf("readRand: %v", err)
	}
	if n != 16 {
		t.Errorf("expected 16: got %d", n)
	}
}

func TestExtraPKCE_URLSafe(t *testing.T) {
	v := "abc~123_test"
	u := url.QueryEscape(v)
	// url.escape doesn't encode alphanum or tested chars. Test that PathEscape works.
	u2 := url.PathEscape(v)
	_ = u2
	// just verify call doesn't panic
	_ = u
}
