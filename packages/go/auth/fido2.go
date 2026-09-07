// Package auth - FIDO2/WebAuthn helpers (challenge store + ceremony wiring).
package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// WebAuthnConfig cấu hình WebAuthn cho một tenant.
// Mỗi tenant nên có config riêng để RP_ID khớp với domain.
type WebAuthnConfig struct {
	RPDisplayName          string // tên hiển thị (e.g. "RINCO")
	RPID                   string // relying party ID (e.g. "login.rinco.app")
	RPOrigins              []string
	UserVerification       string
	AttestationPreference  string
	AuthenticatorAttachment string
	Timeout                time.Duration
}

// ToWebAuthnConfig convert sang struct của go-webauthn.
func (c WebAuthnConfig) ToWebAuthnConfig() *webauthn.Config {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	uv := protocol.UserVerificationRequirement(c.UserVerification)
	if uv == "" {
		uv = protocol.VerificationPreferred
	}
	attestation := protocol.ConveyancePreference(c.AttestationPreference)
	if attestation == "" {
		attestation = protocol.PreferNoAttestation
	}
	return &webauthn.Config{
		RPDisplayName: c.RPDisplayName,
		RPID:          c.RPID,
		RPOrigins:     c.RPOrigins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification:        uv,
			AuthenticatorAttachment: protocol.AuthenticatorAttachment(c.AuthenticatorAttachment),
		},
		AttestationPreference: attestation,
		Timeout:               int(timeout / time.Millisecond),
	}
}

// WebAuthnUser implements webauthn.User.
type WebAuthnUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte          { return u.ID }
func (u *WebAuthnUser) WebAuthnName() string        { return u.Name }
func (u *WebAuthnUser) WebAuthnDisplayName() string { return u.DisplayName }
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}
func (u *WebAuthnUser) WebAuthnIcon() string { return "" }

// AddCredential thêm credential mới vào user.
func (u *WebAuthnUser) AddCredential(c webauthn.Credential) {
	u.Credentials = append(u.Credentials, c)
}

// ===== Challenge store =====

var (
	ErrChallengeNotFound = errors.New("webauthn: challenge not found")
	ErrChallengeExpired  = errors.New("webauthn: challenge expired")
)

// ChallengeData là session data lưu giữa Begin và Finish.
type ChallengeData struct {
	Flow         string
	UserID       string
	TenantID     string
	Challenge    string
	AllowedCreds [][]byte
	ExpiresAt    time.Time
	Extra        map[string]string
}

// ChallengeStore interface để swap backend (memory/redis/valkey).
type ChallengeStore interface {
	Save(ctx context.Context, key string, data ChallengeData, ttl time.Duration) error
	Get(ctx context.Context, key string) (*ChallengeData, error)
	Delete(ctx context.Context, key string) error
}

// MemoryChallengeStore là in-memory store dùng cho dev/test.
type MemoryChallengeStore struct {
	mu   sync.RWMutex
	data map[string]ChallengeData
}

// NewMemoryChallengeStore tạo in-memory store.
func NewMemoryChallengeStore() *MemoryChallengeStore {
	return &MemoryChallengeStore{data: make(map[string]ChallengeData)}
}

func (s *MemoryChallengeStore) Save(_ context.Context, key string, data ChallengeData, ttl time.Duration) error {
	data.ExpiresAt = time.Now().Add(ttl)
	s.mu.Lock()
	s.data[key] = data
	s.mu.Unlock()
	return nil
}

func (s *MemoryChallengeStore) Get(_ context.Context, key string) (*ChallengeData, error) {
	s.mu.RLock()
	d, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrChallengeNotFound
	}
	if time.Now().After(d.ExpiresAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return nil, ErrChallengeExpired
	}
	return &d, nil
}

func (s *MemoryChallengeStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
	return nil
}

// ===== Ceremony helpers =====

// BeginRegistration sinh options cho registration ceremony.
func BeginRegistration(ctx context.Context, w *webauthn.WebAuthn, store ChallengeStore, user *WebAuthnUser, tenantID string, ttl time.Duration) (string, *protocol.CredentialCreation, error) {
	options, sessionData, err := w.BeginRegistration(user)
	if err != nil {
		return "", nil, err
	}
	key := "webauthn:reg:" + newV7String()
	err = store.Save(ctx, key, ChallengeData{
		Flow:      "registration",
		UserID:    user.WebAuthnName(),
		TenantID:  tenantID,
		Challenge: sessionData.Challenge,
		ExpiresAt: sessionData.Expires,
	}, ttl)
	if err != nil {
		return "", nil, err
	}
	return key, options, nil
}

// FinishRegistration verify client response và update credential list.
func FinishRegistration(ctx context.Context, w *webauthn.WebAuthn, store ChallengeStore, user *WebAuthnUser, key string, responseName, responseValue string) (*webauthn.Credential, error) {
	challenge, err := store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(strBody(responseName, responseValue)))
	if err != nil {
		return nil, err
	}
	sessionData := webauthn.SessionData{
		Challenge: challenge.Challenge,
		UserID:    []byte(challenge.UserID),
		Expires:   challenge.ExpiresAt,
	}
	cred, err := w.CreateCredential(user, sessionData, parsed)
	if err != nil {
		return nil, err
	}
	user.AddCredential(*cred)
	_ = store.Delete(ctx, key)
	return cred, nil
}

// BeginLogin sinh options cho authentication ceremony.
func BeginLogin(ctx context.Context, w *webauthn.WebAuthn, store ChallengeStore, user *WebAuthnUser, tenantID string, ttl time.Duration) (string, *protocol.CredentialAssertion, error) {
	options, sessionData, err := w.BeginLogin(user)
	if err != nil {
		return "", nil, err
	}
	key := "webauthn:login:" + newV7String()
	allowed := make([][]byte, 0, len(user.Credentials))
	for _, c := range user.Credentials {
		allowed = append(allowed, c.ID)
	}
	err = store.Save(ctx, key, ChallengeData{
		Flow:         "login",
		UserID:       user.WebAuthnName(),
		TenantID:     tenantID,
		Challenge:    sessionData.Challenge,
		AllowedCreds: allowed,
		ExpiresAt:    sessionData.Expires,
	}, ttl)
	if err != nil {
		return "", nil, err
	}
	return key, options, nil
}

// FinishLogin verify assertion và trả về credential ID.
func FinishLogin(ctx context.Context, w *webauthn.WebAuthn, store ChallengeStore, user *WebAuthnUser, key, responseName, responseValue string) ([]byte, error) {
	challenge, err := store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(strBody(responseName, responseValue)))
	if err != nil {
		return nil, err
	}
	sessionData := webauthn.SessionData{
		Challenge:            challenge.Challenge,
		UserID:               []byte(challenge.UserID),
		Expires:              challenge.ExpiresAt,
		AllowedCredentialIDs: challenge.AllowedCreds,
	}
	cred, err := w.ValidateLogin(user, sessionData, parsed)
	if err != nil {
		return nil, err
	}
	_ = store.Delete(ctx, key)
	return cred.ID, nil
}

// ===== internal helpers =====

func newV7String() string {
	u, err := uuid.NewV7()
	if err != nil {
		return uuid.New().String()
	}
	return u.String()
}

// strBody gom 2 giá trị JSON về một string để parser nhận.
func strBody(name, value string) string {
	body, _ := json.Marshal(map[string]string{name: value})
	// Ensure we wrap into a reader-compatible shape via base64-friendly raw JSON.
	_ = bytes.NewReader(body)
	return string(body)
}

// Ensure base64 is referenced (kept for future API where Challenge is []byte).
var _ = base64.RawURLEncoding.EncodeToString