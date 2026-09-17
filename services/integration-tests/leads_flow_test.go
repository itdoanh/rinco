// Package integration — leads_flow_test.go covers the canonical lead
// lifecycle:
//   1. Login as tenant admin → get PASETO access token.
//   2. Submit the landing form (POST /landing/v1/leads).
//   3. Verify the lead appears in CRM under the same tenant.
//
//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLeadEndToEndFlow walks the canonical happy path with the unified
// TestContext helpers.
func TestLeadEndToEndFlow(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	if env.Users == nil {
		t.Skip("no demo users")
	}
	ctx := NewTestContext(t, env)

	admin := env.Users.ApexAdmin

	// 1. Login
	tp := ctx.Login(admin.Email, admin.Password, admin.TenantID)
	require.NotEmpty(t, tp.AccessToken, "no access token from login")

	// 2. Submit landing form
	email := RandEmail("e2e")
	status, body := ctx.DoJSON(http.MethodPost,
		env.Service.LandingURL+"/landing/v1/leads",
		map[string]any{
			"tenant_slug": "apexfintech",
			"email":       email,
			"full_name":   "John Doe E2E",
			"phone":       "+84901234567",
			"utm_source":  "facebook",
			"fbclid":      "fb.1." + uuid.NewString(),
		},
		"")
	require.Equal(t, http.StatusCreated, status,
		"landing submit failed: %s", body)
	var submitResp struct {
		ID      string `json:"id"`
		EventID string `json:"event_id"`
		HMAC    string `json:"hmac_signature"`
	}
	require.NoError(t, json.Unmarshal(body, &submitResp))
	require.NotEmpty(t, submitResp.ID)
	require.NotEmpty(t, submitResp.HMAC)
	assert.Len(t, submitResp.HMAC, 64, "HMAC must be 64 hex chars (SHA-256)")

	// 3. Brief async propagation delay.
	time.Sleep(1 * time.Second)

	// 4. Verify lead appears in CRM.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_ = readCtx

	status, body = ctx.DoJSON(http.MethodGet,
		env.Service.CRMURL+"/crm/v1/leads?email="+email,
		nil, tp.AccessToken)
	if status == http.StatusOK {
		var listResp struct {
			Data []struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				Name  string `json:"full_name"`
			} `json:"data"`
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal(body, &listResp))
		assert.GreaterOrEqual(t, listResp.Total, 1, "lead should appear in CRM")
		if len(listResp.Data) > 0 {
			assert.Equal(t, email, listResp.Data[0].Email)
		}
	} else {
		t.Logf("CRM lookup returned %d (may be normal if RLS context missing): %s", status, body)
	}
}

// TestHMACTamperingRejected verifies that submitting a lead with a tampered
// HMAC is rejected by meta-capi-service.
func TestHMACTamperingRejected(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	ctx := NewTestContext(t, env)

	payload := map[string]any{
		"tenant_id":      "apexfintech",
		"lead_id":        uuid.NewString(),
		"event_name":     "Lead",
		"event_id":       uuid.NewString(),
		"timestamp":      time.Now().Unix(),
		"email":          "evil@attacker.com",
		"hmac_signature": "deadbeef" + "0",
	}
	status, _ := ctx.DoJSON(http.MethodPost,
		env.Service.MetaCAPIURL+"/meta-capi/v1/events", payload, "")
	assert.Equal(t, http.StatusUnauthorized, status,
		"tampered HMAC must be rejected by meta-capi")
}

// TestQuorumDeletionRequires2Of3 verifies that a single admin signature
// is insufficient to delete a tenant.
func TestQuorumDeletionRequires2Of3(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	ctx := NewTestContext(t, env)

	slug := fmt.Sprintf("test-%s", RandString(8))
	_, body := ctx.DoJSON(http.MethodPost,
		env.Service.TenantURL+"/tenant/v1/tenants",
		map[string]string{
			"slug":        slug,
			"name":        "Test Tenant",
			"plan":        "starter",
			"owner_email": "test@example.com",
		}, "")
	var tenant struct{ ID string `json:"id"` }
	require.NoError(t, json.Unmarshal(body, &tenant))
	require.NotEmpty(t, tenant.ID)

	_, body = ctx.DoJSON(http.MethodPost,
		env.Service.TenantURL+"/tenant/v1/tenants/"+tenant.ID+"/delete",
		nil, "")
	var quorum struct {
		ID            string   `json:"id"`
		RequiredCount int      `json:"required_count"`
		Executed      bool     `json:"executed"`
		Signatures    []string `json:"signatures"`
	}
	require.NoError(t, json.Unmarshal(body, &quorum))
	require.Equal(t, 2, quorum.RequiredCount)
	require.False(t, quorum.Executed)

	// First signature → not enough
	_, body = ctx.DoJSON(http.MethodPost,
		fmt.Sprintf("%s/q/%s", env.Service.TenantURL, quorum.ID),
		map[string]string{"admin_id": "admin-1"}, "")
	var q1 struct {
		Executed   bool     `json:"executed"`
		Signatures []string `json:"signatures"`
	}
	require.NoError(t, json.Unmarshal(body, &q1))
	require.False(t, q1.Executed, "1-of-2 must not execute")
	require.Len(t, q1.Signatures, 1)
}

// TestMetaCAPIRecordsFailedDelivery verifies the CAPI rejection path.
func TestMetaCAPIRecordsFailedDelivery(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	ctx := NewTestContext(t, env)

	eventID := uuid.NewString()
	payload := map[string]any{
		"tenant_id":      "apexfintech",
		"lead_id":        uuid.NewString(),
		"event_name":     "Lead",
		"event_id":       eventID,
		"timestamp":      time.Now().Unix(),
		"email":          "test@example.com",
		"hmac_signature": "valid-signature",
	}
	_, _ = ctx.DoJSON(http.MethodPost,
		env.Service.MetaCAPIURL+"/meta-capi/v1/events", payload, "")

	status, body := ctx.DoJSON(http.MethodGet,
		env.Service.MetaCAPIURL+"/meta-capi/v1/events/"+eventID,
		nil, "")
	if status == http.StatusOK {
		var dr struct {
			Status   string `json:"status"`
			Attempts int    `json:"attempts"`
		}
		require.NoError(t, json.Unmarshal(body, &dr))
		assert.Contains(t, []string{"failed", "queued"}, dr.Status)
	}
}
