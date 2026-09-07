package capi

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// Event represents a tracking event for CAPI.
type Event struct {
	// Event identification
	EventID   string `json:"event_id,omitempty"`   // UUID for dedup
	EventName string `json:"event_name"`
	EventTime int64  `json:"event_time"`           // Unix timestamp (seconds)

	// Action source
	ActionSource string `json:"action_source"`

	// User data
	UserData *UserData `json:"user_data,omitempty"`

	// Custom data
	CustomData *CustomData `json:"custom_data,omitempty"`

	// Context
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// UserData contains PII and matching data for Meta.
type UserData struct {
	// Emails (should be SHA256 hashed before setting)
	Email   string `json:"em,omitempty"`
	EmailH  string `json:"e,omitempty"` // Already hashed

	// Phone numbers (should be SHA256 hashed before setting)
	Phone   string `json:"ph,omitempty"`
	PhoneH  string `json:"p,omitempty"` // Already hashed

	// Name (should be SHA256 hashed)
	FirstName   string `json:"fn,omitempty"`
	FirstNameH  string `json:"fnb,omitempty"`
	LastName    string `json:"ln,omitempty"`
	LastNameH   string `json:"lnb,omitempty"`

	// Date of birth (format: YYYYMMDD, hashed)
	DateOfBirth string `json:"dob,omitempty"`
	DateOfBirthH string `json:"dobm,omitempty"`

	// Gender (hashed)
	Gender string `json:"ge,omitempty"`

	// Location (hashed)
	City    string `json:"ct,omitempty"`
	State   string `json:"st,omitempty"`
	ZipCode string `json:"zp,omitempty"`
	Country string `json:"country,omitempty"`

	// External identifiers
	ExternalID string `json:"external_id,omitempty"`

	// Client context
	ClientIPAddress   string `json:"client_ip_address,omitempty"`
	ClientUserAgent   string `json:"client_user_agent,omitempty"`

	// Meta identifiers
	FBCookieID string `json:"fbc,omitempty"` // fbclid from URL
	FBPCookieID string `json:"fbp,omitempty"` // _fbp cookie
}

// CustomData contains event-specific attributes.
type CustomData struct {
	// E-commerce
	Value          float64        `json:"value,omitempty"`
	Currency       string         `json:"currency,omitempty"`
	ContentName    string         `json:"content_name,omitempty"`
	ContentCategory string        `json:"content_category,omitempty"`
	ContentIDs     []string       `json:"content_ids,omitempty"`
	ContentType    string         `json:"content_type,omitempty"`
	Contents       []ContentItem  `json:"contents,omitempty"`
	NumItems       int            `json:"num_items,omitempty"`
	OrderID        string         `json:"order_id,omitempty"`

	// Search
	SearchString string `json:"search_string,omitempty"`

	// Custom properties (flexible key-value)
	CustomProps map[string]any `json:"-"`
}

// ContentItem represents a product in an event.
type ContentItem struct {
	ID          string  `json:"id,omitempty"`
	Quantity    int     `json:"quantity,omitempty"`
	ItemPrice   float64 `json:"item_price,omitempty"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Brand       string  `json:"brand,omitempty"`
	Category    string  `json:"category,omitempty"`
}

// HashEmail hashes an email using SHA256 and returns lowercase hex.
func HashEmail(email string) string {
	return HashString(strings.ToLower(strings.TrimSpace(email)))
}

// HashPhone hashes a phone number using SHA256.
// Remove all non-digit characters before hashing.
func HashPhone(phone string) string {
	digits := make([]byte, 0, len(phone))
	for i := range phone {
		if phone[i] >= '0' && phone[i] <= '9' {
			digits = append(digits, phone[i])
		}
	}
	return HashString(string(digits))
}

// HashString hashes a string using SHA256 and returns lowercase hex.
func HashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// NormalizePhone removes non-digit characters from phone number.
func NormalizePhone(phone string) string {
	digits := make([]byte, 0, len(phone))
	for i := range phone {
		if phone[i] >= '0' && phone[i] <= '9' {
			digits = append(digits, phone[i])
		}
	}
	return string(digits)
}

// NewUserData creates a UserData with hashed email and phone.
func NewUserData(email, phone string) *UserData {
	ud := &UserData{}

	if email != "" {
		ud.Email = HashEmail(email)
	}

	if phone != "" {
		phone = NormalizePhone(phone)
		ud.Phone = HashPhone(phone)
	}

	return ud
}

// NewServerEvent creates a new ServerEvent with common fields set.
func NewServerEvent(eventName, actionSource string) *ServerEvent {
	return &ServerEvent{
		EventName:    eventName,
		EventTime:    time.Now().Unix(),
		ActionSource: actionSource,
	}
}

// WithUserData sets user data on an event.
func (e *ServerEvent) WithUserData(ud *UserData) *ServerEvent {
	e.UserData = ud
	return e
}

// WithCustomData sets custom data on an event.
func (e *ServerEvent) WithCustomData(cd *CustomData) *ServerEvent {
	e.CustomData = cd
	return e
}

// WithContext sets IP address and user agent.
func (e *ServerEvent) WithContext(ip, ua string) *ServerEvent {
	e.IPAddress = ip
	e.UserAgent = ua
	if e.UserData != nil {
		e.UserData.ClientIPAddress = ip
		e.UserData.ClientUserAgent = ua
	}
	return e
}

// WithEventID sets a custom event ID for deduplication.
func (e *ServerEvent) WithEventID(id string) *ServerEvent {
	e.EventID = id
	return e
}

// NewLeadEvent creates a Lead event for a form submission.
func NewLeadEvent(email, phone, ip, ua string, value float64, currency string) *ServerEvent {
	event := NewServerEvent(EventLead, ActionSourceWebsite)
	event.UserData = NewUserData(email, phone)
	event.CustomData = &CustomData{
		Value:    value,
		Currency: currency,
	}
	event.WithContext(ip, ua)
	return event
}

// NewCompleteRegistrationEvent creates a CompleteRegistration event.
func NewCompleteRegistrationEvent(email, firstName, lastName, ip, ua string) *ServerEvent {
	event := NewServerEvent(EventCompleteRegistration, ActionSourceWebsite)
	event.UserData = NewUserData(email, "")
	event.UserData.FirstName = HashString(strings.ToLower(firstName))
	event.UserData.LastName = HashString(strings.ToLower(lastName))
	event.WithContext(ip, ua)
	return event
}

// NewPageViewEvent creates a PageView event.
func NewPageViewEvent(ip, ua string) *ServerEvent {
	event := NewServerEvent(EventPageView, ActionSourceWebsite)
	event.WithContext(ip, ua)
	return event
}

// NewPurchaseEvent creates a Purchase event.
func NewPurchaseEvent(email, ip, ua string, value float64, currency string, orderID string, contents []ContentItem) *ServerEvent {
	event := NewServerEvent(EventPurchase, ActionSourceWebsite)
	event.UserData = NewUserData(email, "")
	event.CustomData = &CustomData{
		Value:     value,
		Currency:  currency,
		OrderID:   orderID,
		Contents:  contents,
		NumItems:  len(contents),
	}
	event.WithContext(ip, ua)
	return event
}

// NewContentViewEvent creates a ViewContent event.
func NewContentViewEvent(contentName, contentType string, contentIDs []string, ip, ua string) *ServerEvent {
	event := NewServerEvent(EventViewContent, ActionSourceWebsite)
	event.CustomData = &CustomData{
		ContentName:  contentName,
		ContentType: contentType,
		ContentIDs:  contentIDs,
	}
	event.WithContext(ip, ua)
	return event
}

// GenerateEventID creates a unique event ID for deduplication.
// Format: tenantID + formSlug + identifier + timestamp
func GenerateEventID(tenantID, formSlug, identifier string, timestamp int64) string {
	data := tenantID + "|" + formSlug + "|" + identifier + "|" + strconv.FormatInt(timestamp, 10)
	return HashString(data)
}
