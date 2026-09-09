// Tests for tenant-service migrations registry.
package migrations

import (
	"strings"
	"testing"
)

func TestSQL_0001_init_NotEmpty(t *testing.T) {
	if SQL_0001_init == "" {
		t.Fatal("empty")
	}
}

func TestSQL_0001_init_CreatesSchema(t *testing.T) {
	if !strings.Contains(SQL_0001_init, "CREATE SCHEMA IF NOT EXISTS tenant") {
		t.Error("missing schema")
	}
}

func TestSQL_0001_init_CreatesTenantsTable(t *testing.T) {
	if !strings.Contains(SQL_0001_init, "CREATE TABLE IF NOT EXISTS tenant.tenants") {
		t.Error("missing tenants table")
	}
}

func TestSQL_0001_init_CreatesAuditTable(t *testing.T) {
	if !strings.Contains(SQL_0001_init, "CREATE TABLE IF NOT EXISTS tenant.tenant_audit") {
		t.Error("missing audit table")
	}
}

func TestSQL_0002_domains_CreatesDomains(t *testing.T) {
	if !strings.Contains(SQL_0002_domains_settings, "CREATE TABLE IF NOT EXISTS tenant.tenant_domains") {
		t.Error("missing tenant_domains")
	}
}

func TestSQL_0002_domains_CreatesSettings(t *testing.T) {
	if !strings.Contains(SQL_0002_domains_settings, "CREATE TABLE IF NOT EXISTS tenant.tenant_settings") {
		t.Error("missing tenant_settings")
	}
}

func TestSQL_0002_domains_CreatesBranding(t *testing.T) {
	if !strings.Contains(SQL_0002_domains_settings, "CREATE TABLE IF NOT EXISTS tenant.tenant_branding") {
		t.Error("missing tenant_branding")
	}
}

func TestSQL_0003_usage_CreatesUsage(t *testing.T) {
	if !strings.Contains(SQL_0003_usage, "CREATE TABLE IF NOT EXISTS tenant.tenant_usage") {
		t.Error("missing tenant_usage")
	}
}

func TestSQL_0003_usage_CreatesQuotas(t *testing.T) {
	if !strings.Contains(SQL_0003_usage, "CREATE TABLE IF NOT EXISTS tenant.plan_quotas") {
		t.Error("missing plan_quotas")
	}
}

func TestSQL_0003_usage_HasPlanInserts(t *testing.T) {
	for _, plan := range []string{"starter", "pro", "business", "enterprise"} {
		if !strings.Contains(SQL_0003_usage, "'"+plan+"'") {
			t.Errorf("missing plan %s", plan)
		}
	}
}
