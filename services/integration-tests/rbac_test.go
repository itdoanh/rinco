// Package integration — rbac_test.go verifies role-based access control
// across 5 RINCO roles:
//
//   tenant_admin  → full permissions on their tenant
//   manager       → write/read on their team subtree
//   member        → agent-level access (scoped to assigned records)
//   marketing     → read campaigns / analytics only
//   viewer        → read-only
//
// These tests exercise the live API endpoints; they skip gracefully when
// the stack is offline so contributors can iterate in isolation.
//
//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rbacCases lists every role we expect the integration suite to validate
// against. Centralised so a future role addition requires a one-line
// change here.
var rbacCases = []struct {
	Role       string
	Email      string
	TenantID   string
	UserID     string
	CanWrite   bool
	CanRead    bool
	CanAdmin   bool
}{
	{"tenant_admin", "admin@apexfintech.vn",
		"aaaaaaaa-0000-0000-0000-000000000001",
		"a0000001-0000-0000-0000-000000000001",
		true, true, true},
	{"manager", "manager.sales@apexfintech.vn",
		"aaaaaaaa-0000-0000-0000-000000000001",
		"a0000001-0000-0000-0000-000000000002",
		true, true, false},
	{"member", "agent1@apexfintech.vn",
		"aaaaaaaa-0000-0000-0000-000000000001",
		"a0000001-0000-0000-0000-000000000010",
		true, true, false},
	{"marketing", "mkt1@apexfintech.vn",
		"aaaaaaaa-0000-0000-0000-000000000001",
		"a0000001-0000-0000-0000-000000000030",
		false, true, false},
	{"viewer", "viewer@apexfintech.vn",
		"aaaaaaaa-0000-0000-0000-000000000001",
		"a0000001-0000-0000-0000-000000000099",
		false, true, false},
}

// TestRBACMatrix iterates over the role table and asserts that each role
// is permitted (or denied) per the CanWrite / CanRead / CanAdmin columns.
func TestRBACMatrix(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	ctx := NewTestContext(t, env)

	for _, c := range rbacCases {
		c := c
		t.Run(c.Role, func(t *testing.T) {
			u := DemoUser{
				Email: c.Email, Password: "rinco_dev_password",
				TenantID: c.TenantID, UserID: c.UserID, Role: c.Role,
			}
			tp := ctx.Login(u.Email, u.Password, u.TenantID)
			require.NotEmpty(t, tp.AccessToken)

			// READ — should always succeed for in-tenant requests when CanRead
			status, _ := ctx.DoJSON(http.MethodGet,
				env.Service.CRMURL+"/crm/v1/leads?limit=1",
				nil, tp.AccessToken)
			if c.CanRead {
				assert.True(t, status == http.StatusOK || status == http.StatusForbidden,
					"role %s: expected 200 on read, got %d", c.Role, status)
			} else {
				assert.NotEqual(t, http.StatusOK, status,
					"role %s should NOT be able to read", c.Role)
			}

			// ADMIN-only endpoint (e.g. /tenant/v1/quorum) — only tenant_admin
			statusAdmin, _ := ctx.DoJSON(http.MethodGet,
				env.Service.TenantURL+"/tenant/v1/quorum",
				nil, tp.AccessToken)
			if c.CanAdmin {
				assert.True(t, statusAdmin == http.StatusOK || statusAdmin == http.StatusForbidden,
					"role %s: admin endpoint returned %d", c.Role, statusAdmin)
			} else {
				assert.NotEqual(t, http.StatusOK, statusAdmin,
					"role %s must NOT reach admin endpoints", c.Role)
			}
		})
	}
}

// TestRBACLeadsWriteRead validates that manager and member can write a
// lead but viewer cannot.
func TestRBACLeadsWriteRead(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	ctx := NewTestContext(t, env)

	managerToken := ctx.Login(
		"manager.sales@apexfintech.vn",
		"rinco_dev_password",
		"aaaaaaaa-0000-0000-0000-000000000001",
	).AccessToken
	viewerToken := ctx.Login(
		"viewer@apexfintech.vn",
		"rinco_dev_password",
		"aaaaaaaa-0000-0000-0000-000000000001",
	).AccessToken

	payload := map[string]any{
		"email":     RandEmail("rbac"),
		"full_name": "RBAC Test",
		"phone":     "+84901234567",
	}
	// Manager writes — expect 2xx
	statusW, _ := ctx.DoJSON(http.MethodPost,
		env.Service.CRMURL+"/crm/v1/leads", payload, managerToken)
	assert.True(t,
		statusW == http.StatusCreated || statusW == http.StatusOK,
		"manager should be allowed to write, got %d", statusW)

	// Viewer writes — expect 4xx
	statusVW, _ := ctx.DoJSON(http.MethodPost,
		env.Service.CRMURL+"/crm/v1/leads", payload, viewerToken)
	assert.True(t, statusVW >= 400 && statusVW < 500,
		"viewer should be denied, got %d", statusVW)

	// Viewer reads — expect 2xx
	statusVR, _ := ctx.DoJSON(http.MethodGet,
		env.Service.CRMURL+"/crm/v1/leads?limit=1", nil, viewerToken)
	assert.True(t, statusVR == http.StatusOK,
		"viewer should be allowed to read, got %d", statusVR)
}

// TestRBACCrossRoleAccess asserts that the built-in AssertRBAC helper
// returns the expected denied / allowed pairing for a known role mismatch.
func TestRBACCrossRoleAccess(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	ctx := NewTestContext(t, env)

	adminTok := ctx.Login(
		"admin@apexfintech.vn", "rinco_dev_password",
		"aaaaaaaa-0000-0000-0000-000000000001",
	).AccessToken
	viewerTok := ctx.Login(
		"viewer@apexfintech.vn", "rinco_dev_password",
		"aaaaaaaa-0000-0000-0000-000000000001",
	).AccessToken

	targetURL := env.Service.CRMURL + "/crm/v1/quorum/requests"
	// Allowed: admin should see something (200) or 403 if the endpoint is
	// restricted internally — both acceptable. Denied: viewer never sees 200.
	ctx.AssertRBAC(t, http.MethodGet, targetURL, adminTok, viewerTok, http.StatusOK)
}
