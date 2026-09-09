// Extra tests for SMS channel helper functions.
package channels

import (
	"context"
	"testing"
)

func TestTruncate_Short(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestTruncate_Long(t *testing.T) {
	got := truncate("abcdefghij", 5)
	if len(got) != 5 {
		t.Errorf("len: %d", len(got))
	}
	if got != "abcde" {
		t.Errorf("content: %q", got)
	}
}

func TestTruncate_Empty(t *testing.T) {
	if got := truncate("", 10); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestTruncate_Exact(t *testing.T) {
	if got := truncate("hello", 5); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestNewSMSChannel(t *testing.T) {
	c := NewSMSChannel("AC123", "auth-token", "+1234567890")
	if c == nil {
		t.Fatal("nil")
	}
	if c.accountSID != "AC123" {
		t.Error("accountSID")
	}
}

func TestSMSChannel_NameEx(t *testing.T) {
	c := NewSMSChannel("AC", "token", "+1")
	if c.Name() != "sms" {
		t.Errorf("name: %s", c.Name())
	}
}

func TestSMSChannel_Send_NoPhone(t *testing.T) {
	c := NewSMSChannel("AC", "token", "+1")
	_, err := c.Send(context.Background(), Notification{Body: "test"})
	if err == nil {
		t.Error("expected error for missing phone")
	}
}

func TestSMSChannel_Client(t *testing.T) {
	c := NewSMSChannel("AC", "token", "+1")
	if c.client == nil {
		t.Error("nil client")
	}
}
