// Package auth - shared PASETO verifier used by every Go service that needs
// to validate access tokens minted by auth-service.
//
// Why this exists:
//
//   auth-service is the only writer of access tokens in RINCO.  Every other
//   Go service (crm, lead, billing, meta-capi, analytics, ...) currently
//   re-implements its own PASETO verify path or — worse — trusts whatever
//   the upstream proxy decoded.  That works in dev but breaks down as soon
//   as you (a) rotate keys, (b) want a uniform 401 contract, or (c) need
//   claims attached to structured logs.
//
//   This file is the *one* canonical implementation.  Services link it via
//   `github.com/itdoanh/rinco/packages/go/auth` (already in `packages/go/go.mod`)
//   and use it like:
//
//       v, err := auth.NewVerifierFromEnv()
//       if err != nil { ... }
//       claims, err := v.VerifyAccessToken(bearerString)
//       // claims.UserID, claims.TenantID, claims.Role, claims.ExpiresAt
//
// Configuration:
//
//   - PASETO_KEY_CURRENT  (required) 32-byte hex; same value as auth-service
//   - PASETO_KEY_PREVIOUS (optional) 32-byte hex; accepted during rotation
//   - AUTH_CLOCK_SKEW_SECONDS (optional, default 60) — tolerance for `exp`/`nbf`
//
// The verifier is safe for concurrent use; it carries no per-request state
// and reuses the underlying KeyRing (which is itself stateless).
package auth

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// AccessClaims is the trimmed claim set every service needs from a PASETO
// access token.  We deliberately do NOT expose Roles / Permissions here —
// services should query auth-service (or a cached projection) for the
// authoritative RBAC verdict.  What we *do* expose is enough to scope
// queries, write audit logs and tag metrics.
type AccessClaims struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	TokenID   string    `json:"token_id"`
	Issuer    string    `json:"issuer"`
}

// IsExpired reports whether the token has expired past the configured
// clock-skew tolerance.  Safe to call before VerifyAccessToken when you
// want to short-circuit with a friendlier error message.
//
// Semantics: token is "expired" iff `now > ExpiresAt + skew`.  When
// ExpiresAt is zero we return false (the verifier has its own opinion on
// whether a zero-exp token is acceptable, so we don't pre-judge).
func (c *AccessClaims) IsExpired(now time.Time, skew time.Duration) bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return now.After(c.ExpiresAt.Add(skew))
}

// IsSuperAdmin is a convenience predicate mirroring the role convention
// used across RINCO services.  Services that need a more granular RBAC
// verdict should call auth-service instead.
func (c *AccessClaims) IsSuperAdmin() bool {
	return c.Role == "super_admin" || c.Role == "platform_admin"
}

// Verifier validates PASETO v2 local access tokens minted by auth-service.
//
// The zero value is unusable; obtain one via NewVerifier / NewVerifierFromEnv.
// Once constructed, Verifier is immutable and safe for concurrent use.
type Verifier struct {
	keyRing   *KeyRing
	clockSkew time.Duration
	issuer    string // optional; if non-empty, claims.Issuer must match
}

// VerifierOption configures a Verifier at construction time.
type VerifierOption func(*Verifier)

// WithClockSkew overrides the default 60-second tolerance.
func WithClockSkew(d time.Duration) VerifierOption {
	return func(v *Verifier) { v.clockSkew = d }
}

// WithIssuer pins the verifier to a specific `iss` claim.  Tokens minted
// with a different issuer are rejected with ErrInvalidIssuer.
func WithIssuer(iss string) VerifierOption {
	return func(v *Verifier) { v.issuer = iss }
}

// NewVerifier builds a Verifier from explicit hex keys.
//
//   currentHex   — required, 32-byte hex (64 chars).
//   previousHex  — optional, 32-byte hex; accepts tokens signed with it.
//
// Returns ErrInvalid if either key is malformed.  We surface that as a
// plain error rather than re-using PASETO's internal sentinel so callers
// don't have to know which crypto package produced the problem.
func NewVerifier(currentHex, previousHex string, opts ...VerifierOption) (*Verifier, error) {
	if strings.TrimSpace(currentHex) == "" {
		return nil, fmt.Errorf("auth: verifier: current key is empty")
	}
	ring, err := NewKeyRing(currentHex, previousHex, "current", "previous")
	if err != nil {
		return nil, fmt.Errorf("auth: verifier: build keyring: %w", err)
	}
	v := &Verifier{
		keyRing:   ring,
		clockSkew: 60 * time.Second,
	}
	for _, o := range opts {
		o(v)
	}
	return v, nil
}

// NewVerifierFromEnv reads PASETO_KEY_CURRENT / PASETO_KEY_PREVIOUS from
// the process environment and returns a ready-to-use Verifier.
//
// It is the recommended constructor for every service main().  Services
// should call this once at boot and reuse the *Verifier for the lifetime
// of the process — building a KeyRing per-request is wasteful and makes
// hot-reload semantics harder.
func NewVerifierFromEnv(opts ...VerifierOption) (*Verifier, error) {
	current := os.Getenv("PASETO_KEY_CURRENT")
	if current == "" {
		return nil, fmt.Errorf("auth: verifier: PASETO_KEY_CURRENT not set")
	}
	previous := os.Getenv("PASETO_KEY_PREVIOUS")
	if skew := os.Getenv("AUTH_CLOCK_SKEW_SECONDS"); skew != "" {
		if n, err := strconv.Atoi(skew); err == nil && n > 0 {
			opts = append(opts, WithClockSkew(time.Duration(n)*time.Second))
		}
	}
	return NewVerifier(current, previous, opts...)
}

// VerifyAccessToken validates the raw PASETO string and returns the
// normalised AccessClaims.  The input MAY be prefixed with `Bearer `;
// the helper strips it.  An empty string yields ErrEmptyToken — callers
// usually translate that to 401 Missing Authorization.
func (v *Verifier) VerifyAccessToken(tokenString string) (*AccessClaims, error) {
	if v == nil || v.keyRing == nil {
		return nil, fmt.Errorf("auth: verifier: not initialised")
	}
	tok := strings.TrimSpace(tokenString)
	if tok == "" {
		return nil, ErrEmptyToken
	}
	// Accept both `Bearer <tok>` and the raw token.  This mirrors the
	// middleware in packages/go/middleware/auth.go so a service can choose
	// either path without coordinating headers.
	//
	// We split into <=2 parts so `Bearer ` (no token after the scheme)
	// falls into the "scheme only" branch and yields ErrEmptyToken instead
	// of trying to "decrypt" the literal string "Bearer".
	if parts := strings.SplitN(tok, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		tok = strings.TrimSpace(parts[1])
		if tok == "" {
			return nil, ErrEmptyToken
		}
	} else if strings.EqualFold(tok, "Bearer") {
		return nil, ErrEmptyToken
	}

	raw, err := v.keyRing.Decrypt(tok)
	if err != nil {
		return nil, err
	}

	claims := &AccessClaims{
		UserID:    pickFirst(raw.UserID, raw.Subject),
		TenantID:  raw.TenantID,
		ExpiresAt: raw.ExpiresAt,
		IssuedAt:  raw.IssuedAt,
		TokenID:   raw.JTI,
		Issuer:    raw.Issuer,
	}
	if len(raw.Roles) > 0 {
		claims.Role = raw.Roles[0]
	} else if raw.Scope != "" {
		// Some older tokens encode the role inside the OAuth-style `scope`
		// claim ("role:tenant_admin ...").  Fall back to that so we don't
		// regress on legacy clients.
		claims.Role = firstScopeValue(raw.Scope, "role:")
	}

	if v.issuer != "" && claims.Issuer != "" && claims.Issuer != v.issuer {
		return nil, ErrInvalidIssuer
	}
	if claims.IsExpired(time.Now(), v.clockSkew) {
		return nil, ErrExpired
	}
	return claims, nil
}

// CurrentKid exposes the active key id for diagnostics and for JWKS
// endpoints that mirror auth-service's `/auth/.well-known/jwks.json`.
func (v *Verifier) CurrentKid() string {
	if v == nil || v.keyRing == nil {
		return ""
	}
	return v.keyRing.CurrentKid()
}

// ===== helpers =====

func pickFirst(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstScopeValue(scope, prefix string) string {
	for _, p := range strings.Fields(scope) {
		if strings.HasPrefix(p, prefix) {
			return strings.TrimPrefix(p, prefix)
		}
	}
	return ""
}

// ErrEmptyToken is returned when VerifyAccessToken receives the empty
// string or only whitespace.  Distinct from ErrInvalid so middleware can
// map it to a 401 with a friendlier message ("missing token" vs
// "malformed token").
var ErrEmptyToken = errors.New("auth: verifier: empty token")

// ErrInvalidIssuer is returned when the verifier was pinned to a specific
// issuer and the token carries a different one.
var ErrInvalidIssuer = errors.New("auth: verifier: issuer mismatch")
