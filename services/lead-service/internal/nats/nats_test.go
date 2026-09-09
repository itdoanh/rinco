// Tests for lead-service NATS wrapper.
package nats

import (
	"testing"
)

func TestSubjects(t *testing.T) {
	expected := map[string]string{
		"SubjectLeadCreated":   "lead.created",
		"SubjectLeadScored":    "lead.scored",
		"SubjectLeadAssigned":  "lead.assigned",
		"SubjectLeadConverted": "lead.converted",
		"SubjectLeadUpdated":   "lead.updated",
		"SubjectUserCreated":   "user.created",
	}
	got := map[string]string{
		"SubjectLeadCreated":   SubjectLeadCreated,
		"SubjectLeadScored":    SubjectLeadScored,
		"SubjectLeadAssigned":  SubjectLeadAssigned,
		"SubjectLeadConverted": SubjectLeadConverted,
		"SubjectLeadUpdated":   SubjectLeadUpdated,
		"SubjectUserCreated":   SubjectUserCreated,
	}
	for k, v := range expected {
		if got[k] != v {
			t.Errorf("%s: got %s want %s", k, got[k], v)
		}
	}
}

func TestNewClient_EmptyURL(t *testing.T) {
	c, err := NewClient("")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if c == nil {
		t.Fatal("nil")
	}
	if c.conn != nil {
		t.Error("expected nil conn")
	}
}

func TestIsConnected_NoConn(t *testing.T) {
	c := &Client{}
	if c.IsConnected() {
		t.Error("expected false")
	}
}

func TestClose_NoConn(t *testing.T) {
	c := &Client{}
	c.Close() // should not panic
}
