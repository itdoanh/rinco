// Package auth_integration covers auth-service flow tests.
//
//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterLoginRefreshLogout walks the full auth lifecycle.
func TestRegisterLoginRefreshLogout(t *testing.T) {
	cfg := loadConfig(t)
	skipIfNoStack(t, cfg)

	email := "e2e-" + randString(8) + "@example.com"
	password := "StrongPassword1"

	// 1. Register
	status, body := httpJSON(t, http.MethodPost, cfg.AuthURL+"/auth/register", map[string]string{
		"email":     email,
		"password":  password,
		"full_name": "E2E User",
	}, nil)
	require.True(t, status == http.StatusCreated || status == http.StatusConflict,
		"register returned %d: %s", status, body)

	// 2. Login
	status, body = httpJSON(t, http.MethodPost, cfg.AuthURL+"/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, nil)
	require.Equal(t, http.StatusOK, status, "login failed: %s", body)
	var login struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, jsonUnmarshal(body, &login))
	require.NotEmpty(t, login.AccessToken)

	// 3. Refresh
	status, body = httpJSON(t, http.MethodPost, cfg.AuthURL+"/auth/refresh",
		map[string]string{"refresh_token": login.RefreshToken}, nil)
	require.Equal(t, http.StatusOK, status, "refresh failed: %s", body)
	var refreshed struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, jsonUnmarshal(body, &refreshed))
	assert.NotEqual(t, login.AccessToken, refreshed.AccessToken,
		"refresh must rotate access token")

	// 4. Logout
	status, _ = httpJSON(t, http.MethodPost, cfg.AuthURL+"/auth/logout", nil,
		map[string]string{"Authorization": "Bearer " + refreshed.AccessToken})
	assert.Equal(t, http.StatusNoContent, status)
}

// TestRejectWeakPassword verifies that the register endpoint enforces
// minimum password complexity.
func TestRejectWeakPassword(t *testing.T) {
	cfg := loadConfig(t)
	skipIfNoStack(t, cfg)

	bad := []string{"short", "nodigits", "12345678"}
	for _, p := range bad {
		status, _ := httpJSON(t, http.MethodPost, cfg.AuthURL+"/auth/register",
			map[string]string{"email": randString(8) + "@x.co", "password": p, "full_name": "X"}, nil)
		assert.Equal(t, http.StatusBadRequest, status, "weak password %q must be rejected", p)
	}
}

// TestRejectWrongLoginCredentials verifies a wrong password returns 401.
func TestRejectWrongLoginCredentials(t *testing.T) {
	cfg := loadConfig(t)
	skipIfNoStack(t, cfg)

	status, _ := httpJSON(t, http.MethodPost, cfg.AuthURL+"/auth/login",
		map[string]string{"email": "ghost@nowhere.vn", "password": "Whatever1"}, nil)
	assert.True(t, status == http.StatusUnauthorized || status == http.StatusBadRequest,
		"wrong creds should yield 401/400, got %d", status)
}

// ============================================================================
// helpers
// ============================================================================

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
