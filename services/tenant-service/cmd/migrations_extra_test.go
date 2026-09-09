// Tests for tenant-service migrations.go registry.
package main

import (
	"strings"
	"testing"
)

func TestMigrations_NotEmpty(t *testing.T) {
	if len(Migrations) == 0 {
		t.Fatal("empty")
	}
}

func TestMigrations_HasExpectedEntries(t *testing.T) {
	expected := []string{"0001_init", "0002_domains_settings", "0003_usage"}
	if len(Migrations) != len(expected) {
		t.Errorf("got %d want %d", len(Migrations), len(expected))
	}
	for i, name := range expected {
		if Migrations[i].Name != name {
			t.Errorf("entry %d: got %s want %s", i, Migrations[i].Name, name)
		}
		if Migrations[i].SQL == "" {
			t.Errorf("entry %d: empty SQL", i)
		}
		if !strings.Contains(Migrations[i].SQL, "CREATE") {
			t.Errorf("entry %d: SQL lacks CREATE", i)
		}
	}
}
