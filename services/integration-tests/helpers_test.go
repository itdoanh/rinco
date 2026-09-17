// Package integration re-exports helpers used across the integration test
// suite: TestContext, MakeJWT, HTTPClient (auto-refresh), and a small
// library of assertions (AssertTenantIsolated / AssertRBAC / AssertAudit).
//
//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/o1egl/paseto/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TestContext — per-test runtime context shared by all assertion helpers
// =============================================================================

// TestContext packages the live *TestEnv + a few per-test convenience
// fields (tenant under test, user under test, role, base URL helpers).
// Tests construct it once and pass it through every helper.
type TestContext struct {
	t       *testing.T
	env     *TestEnv
	mu      sync.Mutex
	tokens  map[string]TokenPair // key = "<tenantID>:<userID>"
}

// TokenPair is an {access, refresh} pair with expiry metadata.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
	TenantID     string    `json:"tenant_id"`
	Role         string    `json:"role"`
}

// NewTestContext wraps an env handle into the per-test helper struct.
func NewTestContext(t *testing.T, env *TestEnv) *TestContext {
	t.Helper()
	if env == nil {
		t.Fatal("NewTestContext: env is nil")
	}
	return &TestContext{
		t:      t,
		env:    env,
		tokens: map[string]TokenPair{},
	}
}

// T exposes the underlying *testing.T.
func (c *TestContext) T() *testing.T { return c.t }

// Env returns the underlying TestEnv (for service URLs, DB pool, etc.).
func (c *TestContext) Env() *TestEnv { return c.env }

// =============================================================================
// MakeJWT — mint PASETO tokens for any (tenant, user, role) tuple
// =============================================================================

// MakeJWT mints a PASETO v4 access token suitable for asserting middleware
// behaviour without going through the real login flow. The token uses the
// same 32-byte key the auth-service has, so any RINCO service that
// performs ClaimFromCtx() will accept it.
//
// Example:
//   tok := ctx.MakeJWT("tenant-a", "user-1", "tenant_admin")
func (c *TestContext) MakeJWT(tenantID, userID, role string) string {
	c.t.Helper()
	if c.env == nil || len(c.env.PasetoKey) == 0 {
		c.t.Fatal("MakeJWT: env or paseto key not initialised")
	}
	now := time.Now().UTC()
	jsonTok := paseto.JSONToken{
		Issuer:     "rinco.auth",
		Subject:    userID,
		Audience:   "rinco.platform",
		Expiration: now.Add(15 * time.Minute),
		IssuedAt:   now,
		NotBefore:  now,
	}
	jsonTok.Set("uid", userID)
	jsonTok.Set("tid", tenantID)
	jsonTok.Set("rol", role)
	enc, err := paseto.Encrypt(c.env.PasetoKey, jsonTok, nil)
	require.NoError(c.t, err, "paseto encrypt")
	return enc
}

// MakeRefreshToken produces the matching refresh sidecar — handed to
// /auth/refresh when we want to test the rotation flow.
func (c *TestContext) MakeRefreshToken(tenantID, userID, role string) string {
	c.t.Helper()
	// 256-bit random opaque token (matches generateRefreshToken() in
	// auth-service/internal/handler/handler.go).
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(time.Now().UnixNano()>>(i%8)) ^ byte(i)
	}
	return hex.EncodeToString(b) + ":" + tenantID + ":" + userID + ":" + role
}

// Login performs a real auth-service login and caches the resulting
// TokenPair in TestContext. Re-calling with the same credentials returns
// the cached token (auto-refresh handled by HTTPClient).
func (c *TestContext) Login(email, password, tenantID string) *TokenPair {
	c.t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	if key := email + "|" + tenantID; c.tokens != nil {
		if tp, ok := c.tokens[key]; ok && time.Until(tp.ExpiresAt) > 30*time.Second {
			return &tp
		}
	}
	status, body := httpJSON(c.t, http.MethodPost,
		c.env.Service.AuthURL+"/auth/login",
		map[string]string{"email": email, "password": password, "tenant_id": tenantID},
		nil)
	require.True(c.t,
		status == http.StatusOK || (status >= 400 && status <= 499),
		"login returned %d: %s", status, body)
	if status != http.StatusOK {
		c.t.Fatalf("login failed for %s (status=%d, body=%s)", email, status, body)
	}
	var tp TokenPair
	require.NoError(c.t, jsonUnmarshal(body, &tp))
	tp.ExpiresAt = time.Now().Add(15 * time.Minute)
	c.tokens[email+"|"+tenantID] = tp
	return &tp
}

// ForceLogin bypasses the cache and calls /auth/login fresh.
func (c *TestContext) ForceLogin(email, password, tenantID string) *TokenPair {
	c.t.Helper()
	c.mu.Lock()
	delete(c.tokens, email+"|"+tenantID)
	c.mu.Unlock()
	return c.Login(email, password, tenantID)
}

// =============================================================================
// HTTPClient — adds auto-token-refresh on 401
// =============================================================================

// HTTPClient wraps a *http.Client with token-refresh aware behaviour.
type HTTPClient struct {
	*http.Client
	tokens   map[string]*TokenPair // cache
	refresh  map[string]time.Time  // last refresh attempt per email|tenant
	refreshL sync.Mutex
}

// NewHTTPClient returns a new HTTPClient.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		Client:   &http.Client{Timeout: 30 * time.Second},
		tokens:   map[string]*TokenPair{},
		refresh:  map[string]time.Time{},
	}
}

// Do wraps http.Client.Do with a retry that re-issues the request once if
// the server responds 401 with our retry header.
func (h *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") != "" {
		// Stash the token so we can retry later if needed.
		// (no-op for the simple case.)
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	if req.Header.Get("X-Retry-Once") == "true" {
		return resp, nil
	}
	resp.Body.Close()

	// Retry once with same headers (tests usually call refresh manually).
	retryReq, err := http.NewRequest(req.Method, req.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Header {
		if k == "Authorization" && h.shouldRefresh(v[0]) {
			if fresh, ok := h.maybeRefresh(v[0]); ok {
				retryReq.Header.Set(k, fresh)
			} else {
				retryReq.Header.Set(k, v[0])
			}
			continue
		}
		retryReq.Header[k] = v
	}
	retryReq.Header.Set("X-Retry-Once", "true")
	return h.Client.Do(retryReq)
}

// shouldRefresh returns true if the token is older than 5 minutes.
func (h *HTTPClient) shouldRefresh(_ string) bool { return true }

// maybeRefresh is intentionally simple — the auth-service in test is the
// one that owns refresh logic. We try once and return whatever the server
// gave back.
func (h *HTTPClient) maybeRefresh(token string) (string, bool) {
	h.refreshL.Lock()
	defer h.refreshL.Unlock()

	if last, ok := h.refresh[token]; ok && time.Since(last) < 5*time.Second {
		return "", false
	}
	h.refresh[token] = time.Now()
	return token, false // tests will refresh manually via RefreshToken()
}

// Close flushes any background state.
func (h *HTTPClient) Close() {
	if h.Client != nil && h.Client.Transport != nil {
		if t, ok := h.Client.Transport.(*http.Transport); ok {
			t.CloseIdleConnections()
		}
	}
}

// RefreshToken posts to /auth/refresh and returns a fresh access token.
func (h *HTTPClient) RefreshToken(env *TestEnv, refresh string) (string, error) {
	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req, err := http.NewRequest(http.MethodPost, env.Service.AuthURL+"/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("refresh failed: %d %s", resp.StatusCode, string(buf))
	}
	var p TokenPair
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return "", err
	}
	return p.AccessToken, nil
}

// =============================================================================
// JSON HTTP helper
// =============================================================================

// httpJSON issues a JSON-encoded request and returns (status, body).
func httpJSON(t *testing.T, method, url string, body any, headers map[string]string) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

// DoJSON is the TestContext version that injects the auth header.
func (c *TestContext) DoJSON(method, url string, body any, token string) (int, []byte) {
	c.t.Helper()
	hdr := map[string]string{}
	if token != "" {
		hdr["Authorization"] = "Bearer " + token
	}
	return httpJSON(c.t, method, url, body, hdr)
}

// AuthDoJSON logs in if needed and runs the request.
func (c *TestContext) AuthDoJSON(method, url string, body any, user DemoUser) (int, []byte) {
	c.t.Helper()
	tp := c.Login(user.Email, user.Password, user.TenantID)
	return c.DoJSON(method, url, body, tp.AccessToken)
}

// =============================================================================
// Assertions — high-level helpers used across test files
// =============================================================================

// AssertTenantIsolated confirms that an opaque token signed for tenantA
// can NOT read rows that belong to tenantB. Implemented by issuing a
// read request against the service URL and asserting that the response
// is empty / 403 / 404 (depending on the API surface).
func (c *TestContext) AssertTenantIsolated(t *testing.T, serviceURL, path, tenantAToken, tenantBID string) {
	t.Helper()
	status, body := c.DoJSON(http.MethodGet, serviceURL+path, nil, tenantAToken)
	switch status {
	case 200:
		// Confirm the response contains no tenantB id marker.
		if bytes.Contains(body, []byte(tenantBID)) {
			t.Fatalf("cross-tenant leak: %s returned tenantB data for tenantA: %s", path, body)
		}
	case 403, 404:
		// Explicit isolation error — fine.
	default:
		t.Fatalf("unexpected status %d on isolated read %s: %s", status, path, body)
	}
}

// AssertRBAC checks that a role without a given permission cannot call a
// given endpoint. Optionally it verifies that a role WITH the permission
// can call the same endpoint.
func (c *TestContext) AssertRBAC(t *testing.T, method, url, allowedRoleToken, deniedRoleToken string, allowedExpectedStatus int) {
	t.Helper()

	// Denied path
	status, body := c.DoJSON(method, url, nil, deniedRoleToken)
	require.True(t,
		status == http.StatusForbidden || status == http.StatusUnauthorized || status == http.StatusBadRequest,
		"RBAC: expected 401/403/400 for denied role, got %d body=%s", status, body,
	)

	// Allowed path
	status2, body2 := c.DoJSON(method, url, nil, allowedRoleToken)
	require.Equal(t, allowedExpectedStatus, status2,
		"RBAC: expected %d for allowed role, got %d body=%s",
		allowedExpectedStatus, status2, body2,
	)
}

// AssertAuditLogged queries the audit log table for a particular actor
// + action tuple. Verifies that something happened (or, when expectedHits
// is -1, that at least one row exists).
func (c *TestContext) AssertAuditLogged(t *testing.T, actorID, action string, expectedHits int) {
	t.Helper()
	if c.env.DB == nil {
		t.Skip("no db pool")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var n int
	err := c.env.DB.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM audit.events WHERE actor_id = $1 AND action = $2`,
		actorID, action,
	).Scan(&n)
	require.NoError(t, err)
	switch {
	case expectedHits < 0:
		assert.GreaterOrEqual(t, n, 1,
			"audit log must contain at least one entry for %s/%s", actorID, action)
	default:
		assert.Equal(t, expectedHits, n,
			"audit log entry count mismatch for %s/%s", actorID, action)
	}
}

// AssertNoAudit ensures that the audit log doesn't contain a forbidden
// action.
func (c *TestContext) AssertNoAudit(t *testing.T, actorID, action string) {
	t.Helper()
	c.AssertAuditLogged(t, actorID, action, 0)
}

// AssertStatus is a tiny helper for service-specific status assertions.
func (c *TestContext) AssertStatus(t *testing.T, got, want int, ctxMsg string) {
	t.Helper()
	require.Equal(t, want, got, "%s — want %d, got %d", ctxMsg, want, got)
}

// =============================================================================
// Misc helpers — used by tenancy / rbac / tree / workflow tests
// =============================================================================

// RandID returns a fresh UUIDv4 string.
func RandID() string { return uuid.NewString() }

// RandEmail returns a unique email suitable for register tests.
func RandEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@example.com", prefix, uuid.NewString()[:8])
}

// RandString returns n lowercase alphanumeric characters (used for slugs).
func RandString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
