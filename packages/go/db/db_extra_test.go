// Tests for db package (db.go).
package db

import (
	"testing"
	"time"
)

func TestExtra_Config_DSN(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		Database: "mydb",
		SSLMode:  "disable",
	}
	
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
	if dsn == "localhost" {
		t.Error("DSN should include all parts")
	}
}

func TestExtra_Config_DSN_WithPort(t *testing.T) {
	cfg := Config{
		Host:     "db.example.com",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "testdb",
		SSLMode:  "require",
	}
	
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestExtra_Config_DSN_WithoutPort(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		User:     "user",
		Password: "pass",
		Database: "db",
		SSLMode:  "disable",
	}
	
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestExtra_Config_DSN_WithAppName(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		User:            "user",
		Password:        "pass",
		Database:        "db",
		SSLMode:         "disable",
		ApplicationName: "test-app",
	}
	
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestExtra_Config_DSN_WithCacheCapacity(t *testing.T) {
	cfg := Config{
		Host:                    "localhost",
		User:                   "user",
		Password:               "pass",
		Database:               "db",
		SSLMode:                "disable",
		StatementCacheCapacity:  100,
	}
	
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestExtra_Config_WithDefaults(t *testing.T) {
	cfg := Config{}
	got := cfg.WithDefaults()
	
	if got.SSLMode != "disable" {
		t.Errorf("SSLMode: got %s", got.SSLMode)
	}
	if got.MaxConns != 25 {
		t.Errorf("MaxConns: got %d", got.MaxConns)
	}
	if got.MinConns != 2 {
		t.Errorf("MinConns: got %d", got.MinConns)
	}
	if got.MaxConnLifetime != 30*time.Minute {
		t.Errorf("MaxConnLifetime: got %v", got.MaxConnLifetime)
	}
	if got.MaxConnIdleTime != 5*time.Minute {
		t.Errorf("MaxConnIdleTime: got %v", got.MaxConnIdleTime)
	}
	if got.HealthCheckPeriod != 30*time.Second {
		t.Errorf("HealthCheckPeriod: got %v", got.HealthCheckPeriod)
	}
}

func TestExtra_Config_WithDefaults_PreservesSet(t *testing.T) {
	cfg := Config{
		SSLMode:      "require",
		MaxConns:     50,
		MaxConnLifetime: 1 * time.Hour,
	}
	got := cfg.WithDefaults()
	
	if got.SSLMode != "require" {
		t.Error("SSLMode should be preserved")
	}
	if got.MaxConns != 50 {
		t.Error("MaxConns should be preserved")
	}
	if got.MaxConnLifetime != 1*time.Hour {
		t.Error("MaxConnLifetime should be preserved")
	}
}

func TestExtra_Config_Fields(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            5432,
		User:            "user",
		Password:        "pass",
		Database:        "db",
		SSLMode:        "disable",
		MaxConns:       10,
		MinConns:       1,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 10 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
		ApplicationName: "app",
		StatementCacheCapacity: 50,
	}
	
	if cfg.Host != "localhost" {
		t.Error("Host")
	}
	if cfg.Port != 5432 {
		t.Error("Port")
	}
}

func TestExtra_PoolStats_Fields(t *testing.T) {
	stats := PoolStats{
		Active: 5,
		Idle:   3,
		Total:  10,
		Max:    20,
	}
	
	if stats.Active != 5 {
		t.Error("Active")
	}
	if stats.Idle != 3 {
		t.Error("Idle")
	}
	if stats.Total != 10 {
		t.Error("Total")
	}
	if stats.Max != 20 {
		t.Error("Max")
	}
}

func TestExtra_ctxKey_Constants(t *testing.T) {
	if TenantIDKey == "" {
		t.Error("TenantIDKey should not be empty")
	}
	if UserIDKey == "" {
		t.Error("UserIDKey should not be empty")
	}
	if IsAdminKey == "" {
		t.Error("IsAdminKey should not be empty")
	}
	if IsAuthKey == "" {
		t.Error("IsAuthKey should not be empty")
	}
	if RequestIDKey == "" {
		t.Error("RequestIDKey should not be empty")
	}
	if TraceIDKey == "" {
		t.Error("TraceIDKey should not be empty")
	}
	if BypassRLSKey == "" {
		t.Error("BypassRLSKey should not be empty")
	}
}

func TestExtra_ctxKey_Unique(t *testing.T) {
	keys := []ctxKey{TenantIDKey, UserIDKey, IsAdminKey, IsAuthKey, RequestIDKey, TraceIDKey, BypassRLSKey}
	seen := make(map[ctxKey]bool)
	for _, k := range keys {
		if seen[k] {
			t.Errorf("duplicate key: %s", k)
		}
		seen[k] = true
	}
}
