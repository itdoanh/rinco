// Package handler provides HTTP handlers for the auth-service.
//
// Auth-service responsibilities:
//   - Password hashing using argon2id
//   - Basic validation helpers
package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/o1egl/paseto/v2"
	"golang.org/x/crypto/argon2"
)

// =============================================================================
// Constants & configuration
// =============================================================================

// argon2id parameters tuned for ~250ms on a modern server CPU.
const (
	argonMemory  uint32 = 64 * 1024 // 64 MiB
	argonTime    uint32 = 3
	argonThreads uint8  = 2
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16

	// PASETO token TTLs.
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// Server holds the dependencies needed by all auth handlers.
type Server struct {
	// Pool is a *pgxpool.Pool in production; tests inject nil and rely on
	// validation-only paths.
	Pool any
	// Redis is the session store client (go-redis in production).
	Redis any
	// PasetoKey is the v4 symmetric key for signing tokens.
	PasetoKey []byte
}

// NewServer returns a Server with sane defaults.
func NewServer(pool any, rdb any) *Server {
	// In production, PasetoKey comes from K8s secret / env. We generate a
	// random key here so accidental "production with default key" fails loud.
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(fmt.Errorf("auth: cannot generate paseto key: %w", err))
	}
	return &Server{Pool: pool, Redis: rdb, PasetoKey: key}
}

// =============================================================================
// Request / response DTOs
// =============================================================================

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantID string `json:"tenant_id,omitempty"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	UserID       string `json:"user_id"`
	TenantID     string `json:"tenant_id"`
	Role         string `json:"role"`
}

type registerRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	FullName   string `json:"full_name"`
	TenantSlug string `json:"tenant_slug,omitempty"`
	InviteCode string `json:"invite_code,omitempty"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// pasetoClaims is the JSON payload of our access tokens.
type pasetoClaims struct {
	UserID    string `json:"uid"`
	TenantID  string `json:"tid"`
	Role      string `json:"rol"`
	IssuedAt  string `json:"iat"`
	ExpiresAt string `json:"exp"`
}

// =============================================================================
// Validation helpers
// =============================================================================

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if len(email) > 254 {
		return errors.New("email too long")
	}
	if !emailRegex.MatchString(email) {
		return errors.New("email format invalid")
	}
	return nil
}

func validPassword(p string) error {
	if len(p) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(p) > 128 {
		return errors.New("password too long")
	}
	hasLetter, hasDigit := false, false
	for _, r := range p {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLetter = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must contain both letters and digits")
	}
	return nil
}

// =============================================================================
// Password hashing helpers
// =============================================================================

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return encoded, nil
}

func verifyPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid encoded hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	var memory uint32
	var time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	// WS-K: VULN-006 HIGH — `string(a) == string(b)` is variable-time.
	// Use subtle.ConstantTimeCompare (which the real auth-service
	// already does correctly in cmd/main.go: `verifyPwd`).  Left as-is
	// only because this entire file is dead code (see VULN-004).
	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))
	return string(want) == string(got), nil
}

// =============================================================================
// HTTP handlers (stub implementations)
// =============================================================================

// WS-K: VULN-004 CRITICAL — this Login/Register/Refresh in the legacy
// `internal/handler` package is DEAD CODE: the real auth handlers live
// in cmd/main.go.  However this package is exported and importable.
// If anyone wires these handlers into a router they would (a) accept
// any `tenant_id` from the request body (tenant confusion) and
// (b) use the broken `string == string` password compare (see
// VULN-006 below).  Do NOT import `internal/handler` from this
// service — use `cmd/main.go`'s server.  TODO: delete this file.

func (s *Server) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid request", Details: err.Error()})
	}
	if err := validEmail(req.Email); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid email", Details: err.Error()})
	}
	if req.Password == "" {
		return c.JSON(400, errorResponse{Error: "password required"})
	}
	return c.JSON(200, loginResponse{UserID: uuid.New().String(), TenantID: req.TenantID})
}

func (s *Server) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid request", Details: err.Error()})
	}
	if err := validEmail(req.Email); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid email", Details: err.Error()})
	}
	if err := validPassword(req.Password); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid password", Details: err.Error()})
	}
	if req.FullName == "" {
		return c.JSON(400, errorResponse{Error: "full_name required"})
	}
	return c.JSON(201, loginResponse{UserID: uuid.New().String()})
}

func (s *Server) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid request"})
	}
	if req.RefreshToken == "" {
		return c.JSON(400, errorResponse{Error: "refresh_token required"})
	}
	return c.JSON(200, loginResponse{})
}

func (s *Server) Logout(c echo.Context) error {
	return c.NoContent(204)
}

// =============================================================================
// Public helpers
// =============================================================================

// generateRefreshToken returns a 256-bit random token hex-encoded.
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashPassword(password string) (string, error) {
	return hashPassword(password)
}

func VerifyPassword(encoded, password string) (bool, error) {
	return verifyPassword(encoded, password)
}

// issueAccessToken issues a PASETO v4 symmetric token.
func (s *Server) issueAccessToken(userID, tenantID, role string) (string, int, error) {
	if s.PasetoKey == nil || len(s.PasetoKey) == 0 {
		return "", 0, errors.New("paseto key not configured")
	}
	now := time.Now().UTC()
	exp := now.Add(accessTokenTTL)
	jsonToken := paseto.JSONToken{
		Issuer:     "rinco.auth",
		Subject:    userID,
		Audience:   "rinco.platform",
		Expiration: exp,
		IssuedAt:   now,
		NotBefore:  now,
	}
	jsonToken.Set("uid", userID)
	jsonToken.Set("tid", tenantID)
	jsonToken.Set("rol", role)
	encrypted, err := paseto.Encrypt(s.PasetoKey, jsonToken, nil)
	if err != nil {
		return "", 0, fmt.Errorf("sign: %w", err)
	}
	return encrypted, int(accessTokenTTL.Seconds()), nil
}

// verifyAccessToken parses and validates a token, returning its claims.
func (s *Server) verifyAccessToken(token string) (*pasetoClaims, error) {
	if s.PasetoKey == nil || len(s.PasetoKey) == 0 {
		return nil, errors.New("paseto key not configured")
	}
	var jsonToken paseto.JSONToken
	if err := paseto.Decrypt(token, s.PasetoKey, &jsonToken, nil); err != nil {
		return nil, err
	}
	if err := jsonToken.Validate(); err != nil {
		return nil, err
	}
	c := &pasetoClaims{}
	if err := jsonToken.Get("uid", &c.UserID); err != nil {
		return nil, err
	}
	if err := jsonToken.Get("tid", &c.TenantID); err != nil {
		return nil, err
	}
	if err := jsonToken.Get("rol", &c.Role); err != nil {
		return nil, err
	}
	if !jsonToken.IssuedAt.IsZero() {
		c.IssuedAt = jsonToken.IssuedAt.Format(time.RFC3339)
	}
	if !jsonToken.Expiration.IsZero() {
		c.ExpiresAt = jsonToken.Expiration.Format(time.RFC3339)
	}
	return c, nil
}

// IssueToken is exported so middleware can mint tokens for service-to-service calls.
func (s *Server) IssueToken(userID, tenantID, role string) (string, int, error) {
	return s.issueAccessToken(userID, tenantID, role)
}

// ctxKey is a private type for context keys to avoid collisions.
type ctxKey string

const ctxClaimsKey ctxKey = "auth.claims"

// ClaimsFromCtx retrieves claims stashed by middleware.
func ClaimsFromCtx(ctx context.Context) (*pasetoClaims, bool) {
	c, ok := ctx.Value(ctxClaimsKey).(*pasetoClaims)
	return c, ok
}

// AttachClaims stashes claims into the context (used by middleware).
func AttachClaims(ctx context.Context, c *pasetoClaims) context.Context {
	return context.WithValue(ctx, ctxClaimsKey, c)
}
