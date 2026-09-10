// Tests for observability-service cmd helpers.
package main

import "testing"

func TestServiceName(t *testing.T) {
	if serviceName != "observability-service" {
		t.Errorf("serviceName: got %q", serviceName)
	}
	if version != "1.0.0" {
		t.Errorf("version: got %q", version)
	}
}
