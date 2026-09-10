// Tests for notification-service cmd helpers.
package main

import "testing"

func TestNotificationServiceName(t *testing.T) {
	if serviceName != "notification-service" {
		t.Errorf("serviceName: got %q", serviceName)
	}
	if version != "1.0.0" {
		t.Errorf("version: got %q", version)
	}
}
