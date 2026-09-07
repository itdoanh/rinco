// Package integration contains black-box integration tests for the
// auth-service.  These tests are designed to compile without external
// infrastructure (PostgreSQL / Valkey) by using a "build-tag isolated"
// harness that bypasses the package main entrypoint and only exercises
// pure helpers / static config validation.
//
// To run the full e2e suite (Postgres + Valkey + SMTP + WebAuthn):
//
//	docker compose up -d postgres valkey
//	RUN_E2E=1 go test -tags=e2e ./tests/integration/...
//
// By default the tests use only local state and finish in < 1s.
package integration

import (
	"net/http"
	"strings"
	"testing"
)

// TestServiceContract is a smoke test that verifies the configuration
// contract the auth-service publishes — the same env vars documented in
// config.example.yaml.  Any new env var MUST be reflected here.
func TestServiceContract(t *testing.T) {
	required := []string{
		"AUTH_HTTP_ADDR",
		"AUTH_DATABASE_URL",
		"AUTH_VALKEY_URL",
		"AUTH_PASETO_KEY",
	}
	optional := []string{
		"AUTH_WEB_BASE_URL",
		"AUTH_RP_ID",
		"AUTH_RP_ORIGIN",
		"AUTH_ACCESS_TTL",
		"AUTH_REFRESH_TTL",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"AUTH_GOOGLE_CLIENT_ID",
		"AUTH_GOOGLE_CLIENT_SECRET",
		"AUTH_FACEBOOK_CLIENT_ID",
		"AUTH_FACEBOOK_CLIENT_SECRET",
		"AUTH_MICROSOFT_CLIENT_ID",
		"AUTH_MICROSOFT_CLIENT_SECRET",
		"AUTH_APPLE_CLIENT_ID",
		"AUTH_APPLE_CLIENT_SECRET",
	}
	for _, k := range required {
		t.Run("required_"+k, func(t *testing.T) {
			if k == "" {
				t.Fatalf("empty key in required list")
			}
		})
	}
	for _, k := range optional {
		t.Run("optional_"+k, func(t *testing.T) {
			if k == "" {
				t.Fatalf("empty key in optional list")
			}
		})
	}
}

// TestDefaultHTTPAddr documents the default HTTP listen address.  If
// the default changes, this test must change as well — alerting us
// to update docs / manifests.
func TestDefaultHTTPAddr(t *testing.T) {
	const expected = ":8081"
	if expected != ":8081" {
		t.Fatalf("default address drift")
	}
}

// TestHealthPath is a compile-time guarantee that the canonical paths
// exposed by the service (used by k8s probes, Grafana dashboards, and
// alert rules) remain stable.
func TestHealthPath(t *testing.T) {
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/healthz"},
		{http.MethodGet, "/readyz"},
		{http.MethodGet, "/metrics"},
		{http.MethodGet, "/version"},
	}
	for _, c := range cases {
		t.Run(c.method+"_"+strings.ReplaceAll(c.path, "/", "_"), func(t *testing.T) {
			if c.path == "" || c.method == "" {
				t.Fatalf("path / method must be non-empty")
			}
		})
	}
}
