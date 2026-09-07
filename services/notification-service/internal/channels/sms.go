package channels

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// =============================================================================
// SMS channel (Twilio)
// =============================================================================

// SMSChannel delivers SMS via Twilio HTTP API.
type SMSChannel struct {
	accountSID string
	authToken string
	from      string
	client    *http.Client
}

func NewSMSChannel(accountSID, authToken, from string) *SMSChannel {
	return &SMSChannel{
		accountSID: accountSID,
		authToken:  authToken,
		from:      from,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *SMSChannel) Name() string { return "sms" }

func (c *SMSChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	if n.Phone == "" {
		return DeliveryResult{Status: "failed", Provider: "sms"}, fmt.Errorf("phone number required")
	}
	v := url.Values{}
	v.Set("To", n.Phone)
	v.Set("From", c.from)
	v.Set("Body", truncate(n.Body, 1600))
	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + c.accountSID + "/Messages.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "sms"}, err
	}
	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "sms"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return DeliveryResult{Status: "failed", Provider: "sms"}, fmt.Errorf("twilio %d: %s", resp.StatusCode, string(buf))
	}
	return DeliveryResult{Status: "sent", Provider: "sms"}, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
