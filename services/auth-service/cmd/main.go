// Package main — RINCO auth-service entrypoint.
//
// HTTP + Connect-RPC compatible endpoints for authentication, session
// management, password reset, WebAuthn (FIDO2), OAuth (Google / Facebook /
// Microsoft / Apple), and API-key CRUD.  The service is built on Echo +
// pgx + Valkey, runs the embedded SQL migrations on start-up, and
// participates in the RINCO multi-tenant pool via RLS GUCs.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"golang.org/x/crypto/argon2"

	rincoauth "github.com/itdoanh/rinco/services/auth-service/internal/crypto"
	"github.com/itdoanh/rinco/services/auth-service/internal/platform"
)

const (
	serviceName    = "auth-service"
	version        = "1.0.0"
	accessTokenTTL = 1 * time.Hour
	refreshTTL     = 7 * 24 * time.Hour
	resetTokenTTL  = 1 * time.Hour
	stateTTL       = 10 * time.Minute
)

var argonParams = struct{ mem, iter uint32; par uint8; saltLen, keyLen uint32 }{
	mem: 64 * 1024, iter: 3, par: 2, saltLen: 16, keyLen: 32,
}

// =============================================================================
// Types
// =============================================================================

type registerReq struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	FullName   string `json:"full_name"`
	TenantSlug string `json:"tenant_slug,omitempty"`
}
type loginReq struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	TenantSlug        string `json:"tenant_slug"`
	DeviceFingerprint string `json:"device_fingerprint,omitempty"`
}
type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}
type logoutReq struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	Everywhere   bool   `json:"everywhere,omitempty"`
}
type pwChangeReq struct {
	Old string `json:"old_password"`
	New string `json:"new_password"`
}
type pwResetReq struct {
	Email      string `json:"email"`
	TenantSlug string `json:"tenant_slug"`
	WebBaseURL string `json:"web_base_url,omitempty"`
}
type pwResetConfirm struct {
	Token string `json:"token"`
	New   string `json:"new_password"`
}
type profileUpdateReq struct {
	FullName  *string `json:"full_name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Phone     *string `json:"phone,omitempty"`
}
type webauthnFinishReq struct {
	ChallengeKey string          `json:"challenge_key"`
	ResponseName string          `json:"response_name"`
	Response     json.RawMessage `json:"response"`
}
type oauthCallbackReq struct {
	State string `json:"state"`
	Code  string `json:"code"`
}
type apiKeyCreateReq struct {
	Name       string   `json:"name"`
	Scopes     []string `json:"scopes"`
	RateLimit  int      `json:"rate_limit"`
	TTLSeconds int      `json:"ttl_seconds,omitempty"`
	Env        string   `json:"environment,omitempty"`
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`
}
type userResp struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	Email         string     `json:"email"`
	FullName      string     `json:"full_name,omitempty"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	EmailVerified bool       `json:"email_verified"`
	Status        string     `json:"status"`
	MFAEnabled    bool       `json:"mfa_enabled"`
	SuperAdmin    bool       `json:"is_super_admin,omitempty"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
type apiKeyResp struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	Scopes    []string   `json:"scopes"`
	RateLimit int        `json:"rate_limit"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Key       string     `json:"key,omitempty"`
	Last4     string     `json:"last_4,omitempty"`
}

// =============================================================================
// Server
// =============================================================================

type server struct {
	pool         *pgxpool.Pool
	rdb          *redis.Client
	keyRing      *rincoauth.KeyRing
	webauthn     *webauthn.WebAuthn
	oauthMu      sync.Mutex
	oauthStates  map[string]oauthState
	oauthClients map[string]*oauthCfg
	webBase      string
	emailFrom    string
	smtpHost     string
}

type oauthCfg struct {
	provider    string
	authURL     string
	tokenURL    string
	userInfoURL string
	discovery   string
	clientID    string
}

type oauthState struct {
	Provider      string
	PKCEVerifier  string
	RedirectAfter string
	TenantID      string
}

// =============================================================================
// Server metrics
// =============================================================================

var (
	httpReqs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_service_http_requests_total", Help: "HTTP requests",
	}, []string{"method", "route", "status"})
	httpDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "auth_service_http_request_duration_seconds", Help: "Latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// =============================================================================
// main
// =============================================================================

func main() {
	cfg := loadConfig()
	logger := platform.InitLogger(serviceName, cfg.Env, version)
	logger.Info("auth-service starting", slog.String("addr", cfg.HTTPAddr))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, shutdownTracer, err := initTracer(rootCtx, cfg.OTLP, cfg.Env)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		defer func() { _ = shutdownTracer(context.Background()) }()
	}

	pool, err := platform.OpenPool(rootCtx, platform.DBConfigFromEnv("AUTH_"), serviceName)
	if err != nil {
		logger.Error("db connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	if err := runMigrations(rootCtx, pool); err != nil {
		logger.Error("migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.ValkeyAddr, Password: cfg.ValkeyPassword, DB: cfg.ValkeyDB, PoolSize: 20,
	})
	defer func() { _ = rdb.Close() }()
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		logger.Warn("valkey ping failed; cache disabled", slog.String("error", err.Error()))
	}

	ring, err := rincoauth.NewKeyRingFromHex(cfg.PasetoKey, cfg.PasetoPrevious, "v1", "v0")
	if err != nil {
		logger.Error("paseto key init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var wauthn *webauthn.WebAuthn
	wconfig := &webauthn.Config{
		RPDisplayName: "RINCO", RPID: cfg.RPID, RPOrigins: strings.Split(cfg.RPOrigin, ","),
		AuthenticatorSelection: protocol.AuthenticatorSelection{UserVerification: protocol.VerificationPreferred},
		AttestationPreference: protocol.PreferNoAttestation,
		Timeout: int((60 * time.Second) / time.Millisecond),
	}
	if w, err := webauthn.New(wconfig); err != nil {
		logger.Warn("webauthn disabled", slog.String("error", err.Error()))
	} else {
		wauthn = w
	}

	srv := &server{
		pool:         pool,
		rdb:          rdb,
		keyRing:      ring,
		webauthn:     wauthn,
		oauthStates:  make(map[string]oauthState),
		oauthClients: buildOAuthClients(cfg),
		webBase:      cfg.WebBaseURL,
		emailFrom:    cfg.EmailFrom,
		smtpHost:     cfg.SMTPHost,
	}

	e := newEcho(srv)
	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", slog.String("error", err.Error()))
		}
	}()
	<-rootCtx.Done()
	logger.Info("shutdown signal received")
	shut, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := e.Shutdown(shut); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
	}
	logger.Info("auth-service stopped")
}

// =============================================================================
// Config
// =============================================================================

type config struct {
	Env            string
	HTTPAddr       string
	PasetoKey      string
	PasetoPrevious string
	ValkeyAddr     string
	ValkeyPassword string
	ValkeyDB       int
	WebBaseURL     string
	RPID           string
	RPOrigin       string
	OTLP           string
	EmailFrom      string
	SMTPHost       string

	GoogleClientID, GoogleSecret     string
	FacebookClientID, FacebookSecret string
	MicrosoftClientID, MicrosoftSecret string
	AppleClientID, AppleSecret         string
}

func loadConfig() *config {
	c := &config{
		Env:            platform.Getenv("ENV", "development"),
		HTTPAddr:       platform.Getenv("AUTH_HTTP_ADDR", ":8081"),
		PasetoKey:      platform.Getenv("AUTH_PASETO_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"),
		PasetoPrevious: firstNonEmpty(os.Getenv("AUTH_PASETO_PUBLIC"), "0000000000000000000000000000000000000000000000000000000000000000"),
		ValkeyAddr:     platform.Getenv("AUTH_VALKEY_URL", "localhost:6379"),
		ValkeyPassword: platform.Getenv("AUTH_VALKEY_PASSWORD", "rinco_dev_password"),
		ValkeyDB:       platform.GetenvInt("AUTH_VALKEY_DB", 0),
		WebBaseURL:     platform.Getenv("AUTH_WEB_BASE_URL", "https://app.rinco.example"),
		RPID:           platform.Getenv("AUTH_RP_ID", "app.rinco.example"),
		RPOrigin:       platform.Getenv("AUTH_RP_ORIGIN", "https://app.rinco.example"),
		OTLP:           os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		EmailFrom:      platform.Getenv("AUTH_EMAIL_FROM", "no-reply@rinco.example"),
		SMTPHost:       os.Getenv("AUTH_SMTP_HOST"),

		GoogleClientID:      os.Getenv("AUTH_GOOGLE_CLIENT_ID"),
		GoogleSecret:        os.Getenv("AUTH_GOOGLE_CLIENT_SECRET"),
		FacebookClientID:    os.Getenv("AUTH_FACEBOOK_CLIENT_ID"),
		FacebookSecret:      os.Getenv("AUTH_FACEBOOK_CLIENT_SECRET"),
		MicrosoftClientID:   os.Getenv("AUTH_MICROSOFT_CLIENT_ID"),
		MicrosoftSecret:     os.Getenv("AUTH_MICROSOFT_CLIENT_SECRET"),
		AppleClientID:       os.Getenv("AUTH_APPLE_CLIENT_ID"),
		AppleSecret:         os.Getenv("AUTH_APPLE_CLIENT_SECRET"),
	}
	return c
}

// =============================================================================
// Tracing
// =============================================================================

func initTracer(ctx context.Context, endpoint, env string) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if endpoint == "" || env == "development" {
		return nil, func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	res, _ := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName), semconv.ServiceVersion(version),
		semconv.DeploymentEnvironment(env),
	))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp), sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, tp.Shutdown, nil
}

// =============================================================================
// Migrations
// =============================================================================

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("migrations: bootstrap: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("migrations: applied: %w", err)
	}
	applied := map[string]bool{}
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		applied[v] = true
	}
	rows.Close()
	for _, m := range migrationsList() {
		if applied[m.name] {
			continue
		}
		if _, err := pool.Exec(ctx, m.body); err != nil {
			return fmt.Errorf("migrations: %s: %w", m.name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, m.name); err != nil {
			return fmt.Errorf("migrations: bookkeeping %s: %w", m.name, err)
		}
		slog.Info("migration applied", slog.String("name", m.name))
	}
	return nil
}

type migrationFile struct {
	name, body string
}

func migrationsList() []migrationFile {
	return []migrationFile{
		{"0001_init", migration0001},
		{"0002_sessions", migration0002},
		{"0003_webauthn", migration0003},
		{"0004_api_keys", migration0004},
	}
}

// =============================================================================
// Echo setup
// =============================================================================

func newEcho(srv *server) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(recoveryMW())
	e.Use(traceMW())
	e.Use(loggingMW())
	e.Use(metricsMW())
	e.Use(corsMW())
	e.Use(securityHeadersMW())
	e.Use(otelecho.Middleware(serviceName))

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
	})
	e.GET("/readyz", readyz(srv))
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"service": serviceName, "version": version})
	})

	v1 := e.Group("/v1/auth")
	v1.POST("/register", srv.handleRegister)
	v1.POST("/login", srv.handleLogin)
	v1.POST("/refresh", srv.handleRefresh)
	v1.POST("/logout", srv.handleLogout, srv.requireAuth(""))
	v1.POST("/password/change", srv.handlePasswordChange, srv.requireAuth(""))
	v1.POST("/password/reset/request", srv.handlePasswordResetRequest)
	v1.POST("/password/reset/confirm", srv.handlePasswordResetConfirm)
	v1.POST("/webauthn/register/begin", srv.handleWebAuthnRegisterBegin, srv.requireAuth(""))
	v1.POST("/webauthn/register/finish", srv.handleWebAuthnRegisterFinish)
	v1.POST("/webauthn/login/begin", srv.handleWebAuthnLoginBegin)
	v1.POST("/webauthn/login/finish", srv.handleWebAuthnLoginFinish)
	v1.GET("/oauth/:provider/start", srv.handleOAuthStart)
	v1.POST("/oauth/:provider/callback", srv.handleOAuthCallback)
	v1.GET("/oauth/:provider/callback", srv.handleOAuthCallback)
	v1.GET("/me", srv.handleMe, srv.requireAuth(""))
	v1.PUT("/me", srv.handleUpdateMe, srv.requireAuth(""))
	v1.GET("/api-keys", srv.handleAPIKeysList, srv.requireAuth(""))
	v1.POST("/api-keys", srv.handleAPIKeysCreate, srv.requireAuth(""))
	v1.DELETE("/api-keys/:id", srv.handleAPIKeysRevoke, srv.requireAuth(""))

	rpc := e.Group("/internal/auth.v1.AuthService")
	rpc.POST("/Register", srv.rpcRegister)
	rpc.POST("/Login", srv.rpcLogin)
	rpc.POST("/ValidateToken", srv.rpcValidateToken)
	rpc.POST("/RevokeToken", srv.rpcRevokeToken)
	rpc.POST("/GetUser", srv.rpcGetUser)

	return e
}

func readyz(srv *server) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := srv.pool.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_unreachable", "error": err.Error()})
		}
		if srv.rdb != nil {
			if err := srv.rdb.Ping(ctx).Err(); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "valkey_unreachable", "error": err.Error()})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

// =============================================================================
// Handlers — register / login / refresh / logout
// =============================================================================

func (s *server) handleRegister(c echo.Context) error {
	var req registerReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		return jsonErr(c, 400, "INVALID_REQUEST", "email and password (min 8 chars) required")
	}
	ctx := c.Request().Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", "db error")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tenantID string
	if req.TenantSlug != "" {
		if err := tx.QueryRow(ctx,
			`SELECT id::text FROM tenant.tenants WHERE slug=$1 AND deleted_at IS NULL`,
			req.TenantSlug).Scan(&tenantID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return jsonErr(c, 404, "TENANT_NOT_FOUND", "tenant does not exist")
			}
			return jsonErr(c, 500, "DB_ERROR", err.Error())
		}
	} else {
		var count int
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM tenant.tenants WHERE deleted_at IS NULL`).Scan(&count); err != nil {
			return jsonErr(c, 500, "DB_ERROR", err.Error())
		}
		if count == 0 {
			tenantID = newID()
			if _, err := tx.Exec(ctx,
				`INSERT INTO tenant.tenants (id, slug, name, plan, status) VALUES ($1, $2, $3, 'starter', 'active')`,
				tenantID, "default", "Default Tenant"); err != nil {
				return jsonErr(c, 500, "DB_ERROR", err.Error())
			}
		} else {
			return jsonErr(c, 400, "TENANT_REQUIRED", "tenant_slug required")
		}
	}

	hash, err := hashPwd(req.Password)
	if err != nil {
		return jsonErr(c, 500, "HASH_ERROR", err.Error())
	}
	userID := newID()
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, email_verified, status)
		VALUES ($1, $2, $3, $4, $5, FALSE, 'active')
	`, userID, tenantID, req.Email, hash, req.FullName); err != nil {
		if isUniqueViolation(err) {
			return jsonErr(c, 409, "USER_EXISTS", "email already registered")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	s.audit(ctx, tenantID, userID, c.RealIP(), c.Request().UserAgent(), "user.register", "ok", "")
	return c.JSON(http.StatusCreated, userResp{
		ID: userID, TenantID: tenantID, Email: req.Email,
		FullName: req.FullName, Status: "active", CreatedAt: time.Now(),
	})
}

func (s *server) handleLogin(c echo.Context) error {
	var req loginReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	ctx := c.Request().Context()
	var (
		userID, tenantID, passwordHash, status string
		mfaEnabled, isSuperAdmin               bool
	)
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.tenant_id::text, COALESCE(u.password_hash,''), u.status,
		       u.mfa_enabled, COALESCE(u.is_super_admin, false)
		FROM auth.users u
		JOIN tenant.tenants t ON t.id = u.tenant_id
		WHERE LOWER(u.email) = $1 AND ($2 = '' OR t.slug = $2) AND u.deleted_at IS NULL
		LIMIT 1
	`, req.Email, req.TenantSlug).Scan(&userID, &tenantID, &passwordHash, &status, &mfaEnabled, &isSuperAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.audit(ctx, "", "", c.RealIP(), c.Request().UserAgent(), "user.login", "fail_email", "")
			return jsonErr(c, 401, "INVALID_CREDENTIALS", "invalid email or password")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if status == "locked" || status == "suspended" {
		return jsonErr(c, 403, "ACCOUNT_LOCKED", "account "+status)
	}
	if passwordHash == "" {
		return jsonErr(c, 401, "OAUTH_ONLY", "use OAuth")
	}
	ok, err := verifyPwd(req.Password, passwordHash)
	if err != nil || !ok {
		s.audit(ctx, tenantID, userID, c.RealIP(), c.Request().UserAgent(), "user.login", "fail_password", "")
		return jsonErr(c, 401, "INVALID_CREDENTIALS", "invalid email or password")
	}
	access, refresh, err := s.issueTokens(ctx, userID, tenantID, req.DeviceFingerprint, c.RealIP(), c.Request().UserAgent(), isSuperAdmin)
	if err != nil {
		return jsonErr(c, 500, "TOKEN_ERROR", err.Error())
	}
	if _, err := s.pool.Exec(ctx, `UPDATE auth.users SET last_login_at=NOW(), last_login_ip=$1, failed_login_count=0 WHERE id=$2`,
		c.RealIP(), userID); err != nil {
		slog.Warn("update last_login", slog.String("error", err.Error()))
	}
	s.audit(ctx, tenantID, userID, c.RealIP(), c.Request().UserAgent(), "user.login", "ok", "")
	return c.JSON(http.StatusOK, tokenResp{
		AccessToken: access, RefreshToken: refresh,
		ExpiresIn: int(accessTokenTTL.Seconds()), TokenType: "Bearer",
		UserID: userID, TenantID: tenantID,
	})
}

func (s *server) handleRefresh(c echo.Context) error {
	var req refreshReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	hash := rincoauth.SHA256Hex([]byte(req.RefreshToken))
	var (
		userID, tenantID, sessionID string
		isSuperAdmin               bool
	)
	if err := s.pool.QueryRow(ctx, `
		SELECT s.user_id::text, s.tenant_id::text, s.id::text, COALESCE(u.is_super_admin, false)
		FROM auth.sessions s
		JOIN auth.users u ON u.id = s.user_id
		WHERE s.refresh_token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW()
	`, hash).Scan(&userID, &tenantID, &sessionID, &isSuperAdmin); err != nil {
		return jsonErr(c, 401, "INVALID_REFRESH", "invalid or expired refresh token")
	}
	newRefresh, err := rincoauth.GenerateRandomToken(32)
	if err != nil {
		return jsonErr(c, 500, "TOKEN_ERROR", err.Error())
	}
	newHash := rincoauth.SHA256Hex([]byte(newRefresh))
	expires := time.Now().Add(refreshTTL)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE auth.sessions SET revoked_at=NOW(), revoked_reason='rotated' WHERE id=$1`, sessionID); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth.sessions (id, user_id, tenant_id, refresh_token_hash, ip, user_agent, expires_at, previous_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, newID(), userID, tenantID, newHash, c.RealIP(), c.Request().UserAgent(), expires, sessionID); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	access, err := s.encodeAccessToken(userID, tenantID, isSuperAdmin)
	if err != nil {
		return jsonErr(c, 500, "TOKEN_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, tokenResp{
		AccessToken: access, RefreshToken: newRefresh,
		ExpiresIn: int(accessTokenTTL.Seconds()), TokenType: "Bearer",
	})
}

func (s *server) handleLogout(c echo.Context) error {
	var req logoutReq
	_ = json.NewDecoder(c.Request().Body).Decode(&req)
	ctx := c.Request().Context()
	if req.RefreshToken != "" {
		hash := rincoauth.SHA256Hex([]byte(req.RefreshToken))
		if _, err := s.pool.Exec(ctx,
			`UPDATE auth.sessions SET revoked_at=NOW(), revoked_reason='logout' WHERE refresh_token_hash=$1`, hash); err != nil {
			return jsonErr(c, 500, "DB_ERROR", err.Error())
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	reason := "logout"
	if req.Everywhere {
		reason = "logout_all"
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE auth.sessions SET revoked_at=NOW(), revoked_reason=$2 WHERE user_id=$1 AND revoked_at IS NULL`,
		claims.UserID, reason); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	s.audit(ctx, claims.TenantID, claims.UserID, c.RealIP(), c.Request().UserAgent(), "user.logout", "ok", "")
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// =============================================================================
// Profile handlers
// =============================================================================

func (s *server) handleMe(c echo.Context) error {
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	var u userResp
	if err := s.pool.QueryRow(c.Request().Context(), `
		SELECT id::text, tenant_id::text, email, COALESCE(full_name,''), COALESCE(avatar_url,''),
		       COALESCE(phone,''), email_verified, status, mfa_enabled,
		       COALESCE(is_super_admin, false), last_login_at, created_at
		FROM auth.users WHERE id = $1 AND deleted_at IS NULL
	`, claims.UserID).Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.AvatarURL,
		&u.Phone, &u.EmailVerified, &u.Status, &u.MFAEnabled,
		&u.SuperAdmin, &u.LastLoginAt, &u.CreatedAt); err != nil {
		return jsonErr(c, 404, "USER_NOT_FOUND", "user not found")
	}
	return c.JSON(http.StatusOK, u)
}

func (s *server) handleUpdateMe(c echo.Context) error {
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	var req profileUpdateReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	updates, args := []string{}, []interface{}{}
	idx := 1
	if req.FullName != nil {
		updates = append(updates, fmt.Sprintf("full_name=$%d", idx))
		args = append(args, *req.FullName)
		idx++
	}
	if req.AvatarURL != nil {
		updates = append(updates, fmt.Sprintf("avatar_url=$%d", idx))
		args = append(args, *req.AvatarURL)
		idx++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone=$%d", idx))
		args = append(args, *req.Phone)
		idx++
	}
	if len(updates) == 0 {
		return jsonErr(c, 400, "NO_CHANGES", "no fields to update")
	}
	updates = append(updates, "updated_at=NOW()")
	args = append(args, claims.UserID)
	if _, err := s.pool.Exec(c.Request().Context(),
		"UPDATE auth.users SET "+strings.Join(updates, ", ")+" WHERE id=$"+strconv.Itoa(idx), args...); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// =============================================================================
// Password endpoints
// =============================================================================

func (s *server) handlePasswordChange(c echo.Context) error {
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	var req pwChangeReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	if len(req.New) < 8 {
		return jsonErr(c, 400, "WEAK_PASSWORD", "new password must be ≥8 chars")
	}
	ctx := c.Request().Context()
	var oldHash string
	if err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(password_hash,'') FROM auth.users WHERE id=$1`, claims.UserID).Scan(&oldHash); err != nil {
		return jsonErr(c, 404, "USER_NOT_FOUND", err.Error())
	}
	if oldHash == "" {
		return jsonErr(c, 400, "NO_PASSWORD", "no password set (oauth account)")
	}
	ok, err := verifyPwd(req.Old, oldHash)
	if err != nil || !ok {
		return jsonErr(c, 401, "INVALID_PASSWORD", "old password does not match")
	}
	newHash, err := hashPwd(req.New)
	if err != nil {
		return jsonErr(c, 500, "HASH_ERROR", err.Error())
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE auth.users SET password_hash=$1, updated_at=NOW() WHERE id=$2`,
		newHash, claims.UserID); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	_, _ = s.pool.Exec(ctx,
		`UPDATE auth.sessions SET revoked_at=NOW(), revoked_reason='password_changed' WHERE user_id=$1 AND revoked_at IS NULL`,
		claims.UserID)
	s.audit(ctx, claims.TenantID, claims.UserID, c.RealIP(), c.Request().UserAgent(), "user.password_changed", "ok", "")
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) handlePasswordResetRequest(c echo.Context) error {
	var req pwResetReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return jsonErr(c, 400, "INVALID_REQUEST", "email required")
	}
	ctx := c.Request().Context()
	var userID string
	if err := s.pool.QueryRow(ctx, `
		SELECT u.id::text FROM auth.users u
		JOIN tenant.tenants t ON t.id = u.tenant_id
		WHERE LOWER(u.email)=$1 AND ($2 = '' OR t.slug=$2) AND u.deleted_at IS NULL LIMIT 1
	`, email, req.TenantSlug).Scan(&userID); err != nil {
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
	token, err := rincoauth.GenerateRandomToken(32)
	if err != nil {
		return jsonErr(c, 500, "TOKEN_ERROR", err.Error())
	}
	hash := rincoauth.SHA256Hex([]byte(token))
	expires := time.Now().Add(resetTokenTTL)
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO auth.password_resets (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, newID(), userID, hash, expires); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	link := fmt.Sprintf("%s/reset?token=%s", firstNonEmpty(req.WebBaseURL, s.webBase), token)
	slog.Info("password reset link generated", slog.String("user_id", userID), slog.String("link", link))
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) handlePasswordResetConfirm(c echo.Context) error {
	var req pwResetConfirm
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	if len(req.New) < 8 {
		return jsonErr(c, 400, "WEAK_PASSWORD", "password must be ≥8 chars")
	}
	ctx := c.Request().Context()
	hash := rincoauth.SHA256Hex([]byte(req.Token))
	var userID string
	var usedAt *time.Time
	var expires time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT user_id::text, used_at, expires_at FROM auth.password_resets WHERE token_hash=$1`,
		hash).Scan(&userID, &usedAt, &expires); err != nil || usedAt != nil || time.Now().After(expires) {
		return jsonErr(c, 400, "INVALID_TOKEN", "invalid or expired token")
	}
	newHash, err := hashPwd(req.New)
	if err != nil {
		return jsonErr(c, 500, "HASH_ERROR", err.Error())
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE auth.users SET password_hash=$1, updated_at=NOW() WHERE id=$2`, newHash, userID); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if _, err := tx.Exec(ctx, `UPDATE auth.password_resets SET used_at=NOW() WHERE token_hash=$1`, hash); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if _, err := tx.Exec(ctx,
		`UPDATE auth.sessions SET revoked_at=NOW(), revoked_reason='password_reset' WHERE user_id=$1 AND revoked_at IS NULL`,
		userID); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// =============================================================================
// WebAuthn handlers
// =============================================================================

type waUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

func (u *waUser) WebAuthnID() []byte          { return u.ID }
func (u *waUser) WebAuthnName() string        { return u.Name }
func (u *waUser) WebAuthnDisplayName() string { return u.DisplayName }
func (u *waUser) WebAuthnIcon() string        { return "" }
func (u *waUser) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

func (s *server) handleWebAuthnRegisterBegin(c echo.Context) error {
	if s.webauthn == nil {
		return jsonErr(c, 503, "WEBAUTHN_DISABLED", "webauthn not configured")
	}
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	ctx := c.Request().Context()
	var email, fullName string
	if err := s.pool.QueryRow(ctx,
		`SELECT email, COALESCE(full_name,'') FROM auth.users WHERE id=$1`,
		claims.UserID).Scan(&email, &fullName); err != nil {
		return jsonErr(c, 404, "USER_NOT_FOUND", err.Error())
	}
	creds := s.loadCreds(ctx, claims.UserID)
	user := &waUser{ID: []byte(claims.UserID), Name: email, DisplayName: fullName, Credentials: creds}
	options, session, err := s.webauthn.BeginRegistration(user)
	if err != nil {
		return jsonErr(c, 500, "WEBAUTHN_BEGIN_FAILED", err.Error())
	}
	key := "webauthn:reg:" + newID()
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO auth.webauthn_challenges (key, flow, user_id, tenant_id, challenge, user_handle, expires_at)
		VALUES ($1, 'registration', $2, $3, $4, $5, $6)
	`, key, claims.UserID, claims.TenantID, session.Challenge, []byte(claims.UserID), session.Expires); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"challenge_key": key, "options": options})
}

func (s *server) handleWebAuthnRegisterFinish(c echo.Context) error {
	if s.webauthn == nil {
		return jsonErr(c, 503, "WEBAUTHN_DISABLED", "webauthn not configured")
	}
	var req webauthnFinishReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	var userID, tenantID string
	var challenge, userHandle []byte
	if err := s.pool.QueryRow(ctx, `
		SELECT user_id::text, tenant_id::text, challenge, COALESCE(user_handle, '{}')
		FROM auth.webauthn_challenges WHERE key=$1 AND flow='registration' AND expires_at > NOW()
	`, req.ChallengeKey).Scan(&userID, &tenantID, &challenge, &userHandle); err != nil {
		return jsonErr(c, 400, "INVALID_CHALLENGE", "challenge not found or expired")
	}
	defer s.pool.Exec(ctx, `DELETE FROM auth.webauthn_challenges WHERE key=$1`, req.ChallengeKey)

	var email, fullName string
	_ = s.pool.QueryRow(ctx,
		`SELECT email, COALESCE(full_name,'') FROM auth.users WHERE id=$1`, userID).Scan(&email, &fullName)
	user := &waUser{ID: userHandle, Name: email, DisplayName: fullName, Credentials: s.loadCreds(ctx, userID)}
	raw, _ := json.Marshal(map[string]json.RawMessage{req.ResponseName: req.Response})
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(raw))
	if err != nil {
		return jsonErr(c, 400, "WEBAUTHN_PARSE_FAILED", err.Error())
	}
	session := webauthn.SessionData{Challenge: string(challenge), UserID: userHandle, Expires: time.Now().Add(time.Minute)}
	cred, err := s.webauthn.CreateCredential(user, session, parsed)
	if err != nil {
		return jsonErr(c, 400, "WEBAUTHN_VERIFY_FAILED", err.Error())
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, t := range cred.Transport {
		transports = append(transports, string(t))
	}
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO auth.webauthn_credentials (id, user_id, tenant_id, credential_id, public_key, counter, transports, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, newID(), userID, tenantID, cred.ID, cred.PublicKey,
		cred.Authenticator.SignCount, transports, "default")
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) handleWebAuthnLoginBegin(c echo.Context) error {
	if s.webauthn == nil {
		return jsonErr(c, 503, "WEBAUTHN_DISABLED", "webauthn not configured")
	}
	email := c.QueryParam("email")
	if email == "" {
		return jsonErr(c, 400, "INVALID_REQUEST", "email query param required")
	}
	ctx := c.Request().Context()
	var userID, tenantID, fullName string
	if err := s.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, COALESCE(full_name,'') FROM auth.users
		WHERE LOWER(email)=LOWER($1) AND deleted_at IS NULL LIMIT 1
	`, email).Scan(&userID, &tenantID, &fullName); err != nil {
		return jsonErr(c, 404, "USER_NOT_FOUND", "user not found")
	}
	creds := s.loadCreds(ctx, userID)
	if len(creds) == 0 {
		return jsonErr(c, 400, "NO_CREDENTIALS", "no webauthn credentials registered")
	}
	user := &waUser{ID: []byte(userID), Name: email, DisplayName: fullName, Credentials: creds}
	options, session, err := s.webauthn.BeginLogin(user)
	if err != nil {
		return jsonErr(c, 500, "WEBAUTHN_BEGIN_FAILED", err.Error())
	}
	key := "webauthn:login:" + newID()
	allowed := make([][]byte, 0, len(creds))
	for _, cd := range creds {
		allowed = append(allowed, cd.ID)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO auth.webauthn_challenges (key, flow, user_id, tenant_id, challenge, allowed_creds, expires_at)
		VALUES ($1, 'login', $2, $3, $4, $5, $6)
	`, key, userID, tenantID, session.Challenge, allowed, session.Expires); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"challenge_key": key, "options": options})
}

func (s *server) handleWebAuthnLoginFinish(c echo.Context) error {
	if s.webauthn == nil {
		return jsonErr(c, 503, "WEBAUTHN_DISABLED", "webauthn not configured")
	}
	var req webauthnFinishReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	var userID, tenantID, fullName, email string
	var challenge []byte
	var allowedCreds [][]byte
	if err := s.pool.QueryRow(ctx, `
		SELECT user_id::text, tenant_id::text, challenge, allowed_creds
		FROM auth.webauthn_challenges WHERE key=$1 AND flow='login' AND expires_at > NOW()
	`, req.ChallengeKey).Scan(&userID, &tenantID, &challenge, &allowedCreds); err != nil {
		return jsonErr(c, 400, "INVALID_CHALLENGE", "challenge not found or expired")
	}
	defer s.pool.Exec(ctx, `DELETE FROM auth.webauthn_challenges WHERE key=$1`, req.ChallengeKey)
	_ = s.pool.QueryRow(ctx,
		`SELECT email, COALESCE(full_name,'') FROM auth.users WHERE id=$1`, userID).Scan(&email, &fullName)
	user := &waUser{ID: []byte(userID), Name: email, DisplayName: fullName, Credentials: s.loadCreds(ctx, userID)}
	raw, _ := json.Marshal(map[string]json.RawMessage{req.ResponseName: req.Response})
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(raw))
	if err != nil {
		return jsonErr(c, 400, "WEBAUTHN_PARSE_FAILED", err.Error())
	}
	session := webauthn.SessionData{Challenge: string(challenge), UserID: []byte(userID), AllowedCredentialIDs: allowedCreds, Expires: time.Now().Add(time.Minute)}
	if _, err := s.webauthn.ValidateLogin(user, session, parsed); err != nil {
		return jsonErr(c, 401, "WEBAUTHN_LOGIN_FAILED", err.Error())
	}
	access, refresh, err := s.issueTokens(ctx, userID, tenantID, "", c.RealIP(), c.Request().UserAgent(), false)
	if err != nil {
		return jsonErr(c, 500, "TOKEN_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, tokenResp{
		AccessToken: access, RefreshToken: refresh,
		ExpiresIn: int(accessTokenTTL.Seconds()), TokenType: "Bearer",
		UserID: userID, TenantID: tenantID,
	})
}

func (s *server) loadCreds(ctx context.Context, userID string) []webauthn.Credential {
	rows, err := s.pool.Query(ctx, `SELECT credential_id, public_key, counter FROM auth.webauthn_credentials WHERE user_id=$1`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []webauthn.Credential{}
	for rows.Next() {
		var id, pubKey []byte
		var counter uint64
		if err := rows.Scan(&id, &pubKey, &counter); err == nil {
			out = append(out, webauthn.Credential{ID: id, PublicKey: pubKey, Authenticator: webauthn.Authenticator{SignCount: uint32(counter)}})
		}
	}
	return out
}

// =============================================================================
// OAuth handlers
// =============================================================================

func buildOAuthClients(c *config) map[string]*oauthCfg {
	out := map[string]*oauthCfg{}
	if c.GoogleClientID != "" && c.GoogleSecret != "" {
		out["google"] = &oauthCfg{
			provider: "google",
			authURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			tokenURL: "https://oauth2.googleapis.com/token",
			userInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
			discovery:   "https://accounts.google.com",
			clientID:    c.GoogleClientID,
		}
	}
	if c.FacebookClientID != "" && c.FacebookSecret != "" {
		out["facebook"] = &oauthCfg{
			provider: "facebook",
			authURL:  "https://www.facebook.com/v18.0/dialog/oauth",
			tokenURL: "https://graph.facebook.com/v18.0/oauth/access_token",
			userInfoURL: "https://graph.facebook.com/v18.0/me?fields=id,name,first_name,last_name,email,picture.type(large)",
			clientID:    c.FacebookClientID,
		}
	}
	if c.MicrosoftClientID != "" && c.MicrosoftSecret != "" {
		out["microsoft"] = &oauthCfg{
			provider: "microsoft",
			authURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			tokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			userInfoURL: "https://graph.microsoft.com/oidc/userinfo",
			discovery:   "https://login.microsoftonline.com/common/v2.0",
			clientID:    c.MicrosoftClientID,
		}
	}
	if c.AppleClientID != "" && c.AppleSecret != "" {
		out["apple"] = &oauthCfg{
			provider: "apple",
			authURL:  "https://appleid.apple.com/auth/authorize",
			tokenURL: "https://appleid.apple.com/auth/token",
			userInfoURL: "https://appleid.apple.com/auth/userinfo",
			clientID:    c.AppleClientID,
		}
	}
	return out
}

func (s *server) handleOAuthStart(c echo.Context) error {
	provider := c.Param("provider")
	oc, ok := s.oauthClients[provider]
	if !ok {
		return jsonErr(c, 400, "UNKNOWN_PROVIDER", "provider not configured")
	}
	state, err := genState(24)
	if err != nil {
		return jsonErr(c, 500, "STATE_ERROR", err.Error())
	}
	verifier, challenge, err := genPKCE()
	if err != nil {
		return jsonErr(c, 500, "PKCE_ERROR", err.Error())
	}
	redirectAfter := c.QueryParam("redirect_after")
	tenantID := c.QueryParam("tenant_id")
	s.oauthMu.Lock()
	s.oauthStates[state] = oauthState{Provider: provider, PKCEVerifier: verifier, RedirectAfter: redirectAfter, TenantID: tenantID}
	s.oauthMu.Unlock()
	_, _ = s.pool.Exec(c.Request().Context(), `
		INSERT INTO auth.oauth_states (state, provider, tenant_id, pkce_verifier, redirect_after, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, state, provider, nullUUID(tenantID), verifier, redirectAfter, time.Now().Add(stateTTL))

	redirectURI := s.webBase + "/v1/auth/oauth/" + provider + "/callback"
	scopes := "openid email profile"
	if provider == "facebook" {
		scopes = "email public_profile"
	}
	u := fmt.Sprintf(`%s?response_type=code&redirect_uri=%s&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256`,
		oc.authURL,
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(state),
		url.QueryEscape(challenge),
	)
	if oc.clientID != "" {
		u += "&client_id=" + url.QueryEscape(oc.clientID)
	}
	return c.JSON(http.StatusOK, map[string]string{"url": u, "state": state})
}

func (s *server) handleOAuthCallback(c echo.Context) error {
	provider := c.Param("provider")
	state := c.QueryParam("state")
	code := c.QueryParam("code")
	if state == "" || code == "" {
		var req oauthCallbackReq
		_ = json.NewDecoder(c.Request().Body).Decode(&req)
		state = firstNonEmpty(state, req.State)
		code = firstNonEmpty(code, req.Code)
	}
	if state == "" || code == "" {
		return jsonErr(c, 400, "INVALID_REQUEST", "state and code required")
	}
	oc, ok := s.oauthClients[provider]
	if !ok {
		return jsonErr(c, 400, "UNKNOWN_PROVIDER", "provider not configured")
	}
	s.oauthMu.Lock()
	meta, found := s.oauthStates[state]
	if found {
		delete(s.oauthStates, state)
	}
	s.oauthMu.Unlock()
	if !found {
		_ = s.pool.QueryRow(c.Request().Context(),
			`SELECT provider, COALESCE(tenant_id::text,''), COALESCE(pkce_verifier,''), COALESCE(redirect_after,'') FROM auth.oauth_states WHERE state=$1 AND expires_at > NOW()`,
			state).Scan(&meta.Provider, &meta.TenantID, &meta.PKCEVerifier, &meta.RedirectAfter)
		_, _ = s.pool.Exec(c.Request().Context(), `DELETE FROM auth.oauth_states WHERE state=$1`, state)
	}
	if meta.PKCEVerifier == "" {
		return jsonErr(c, 400, "STATE_MISMATCH", "invalid or expired state")
	}
	ctx := c.Request().Context()
	redirectURI := s.webBase + "/v1/auth/oauth/" + provider + "/callback"
	secret := os.Getenv("AUTH_" + strings.ToUpper(provider) + "_CLIENT_SECRET")
	tok, err := exchangeCode(oc.tokenURL, code, meta.PKCEVerifier, redirectURI, oc.clientID, secret)
	if err != nil {
		return jsonErr(c, 400, "OAUTH_EXCHANGE_FAILED", err.Error())
	}
	providerUser, err := s.fetchOAuthUserInfo(ctx, oc, tok)
	if err != nil {
		return jsonErr(c, 400, "OAUTH_USERINFO_FAILED", err.Error())
	}
	if providerUser == nil || providerUser.Email == "" {
		return jsonErr(c, 400, "NO_EMAIL", "provider did not return an email")
	}

	var (
		userID, tenantID string
		isSuperAdmin     bool
	)
	if err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.tenant_id::text, COALESCE(u.is_super_admin, false)
		FROM auth.oauth_accounts oa
		JOIN auth.users u ON u.id = oa.user_id
		WHERE oa.provider = $1 AND oa.provider_user_id = $2 LIMIT 1
	`, provider, providerUser.ProviderUserID).Scan(&userID, &tenantID, &isSuperAdmin); err != nil {
		if err := s.pool.QueryRow(ctx, `
			SELECT u.id::text, u.tenant_id::text, COALESCE(u.is_super_admin, false)
			FROM auth.users u WHERE LOWER(u.email)=LOWER($1) AND ($2='' OR u.tenant_id::text=$2) AND u.deleted_at IS NULL LIMIT 1
		`, providerUser.Email, meta.TenantID).Scan(&userID, &tenantID, &isSuperAdmin); err != nil {
			if meta.TenantID == "" {
				return jsonErr(c, 400, "TENANT_REQUIRED", "tenant_id required to provision")
			}
			tx, txErr := s.pool.Begin(ctx)
			if txErr != nil {
				return jsonErr(c, 500, "DB_ERROR", txErr.Error())
			}
			defer func() { _ = tx.Rollback(ctx) }()
			userID = newID()
			tenantID = meta.TenantID
			if _, err := tx.Exec(ctx, `INSERT INTO auth.users (id, tenant_id, email, full_name, email_verified, status) VALUES ($1, $2, $3, $4, $5, 'active')`,
				userID, tenantID, providerUser.Email, providerUser.FullName, providerUser.EmailVerified); err != nil {
				return jsonErr(c, 500, "DB_ERROR", err.Error())
			}
			if _, err := tx.Exec(ctx, `INSERT INTO auth.oauth_accounts (id, user_id, tenant_id, provider, provider_user_id, provider_email, raw_profile) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				newID(), userID, tenantID, provider, providerUser.ProviderUserID, providerUser.Email, providerUser.Raw); err != nil {
				return jsonErr(c, 500, "DB_ERROR", err.Error())
			}
			if err := tx.Commit(ctx); err != nil {
				return jsonErr(c, 500, "DB_ERROR", err.Error())
			}
			isSuperAdmin = false
		} else {
			_, _ = s.pool.Exec(ctx, `INSERT INTO auth.oauth_accounts (id, user_id, tenant_id, provider, provider_user_id, provider_email, raw_profile) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (provider, provider_user_id) DO NOTHING`,
				newID(), userID, tenantID, provider, providerUser.ProviderUserID, providerUser.Email, providerUser.Raw)
		}
	}
	access, refresh, err := s.issueTokens(ctx, userID, tenantID, "", c.RealIP(), c.Request().UserAgent(), isSuperAdmin)
	if err != nil {
		return jsonErr(c, 500, "TOKEN_ERROR", err.Error())
	}
	s.audit(ctx, tenantID, userID, c.RealIP(), c.Request().UserAgent(), "user.oauth_login", "ok", provider)
	if meta.RedirectAfter != "" {
		sep := "?"
		if strings.Contains(meta.RedirectAfter, "?") {
			sep = "&"
		}
		return c.Redirect(http.StatusFound,
			meta.RedirectAfter+sep+"access_token="+access+"&refresh_token="+refresh)
	}
	return c.JSON(http.StatusOK, tokenResp{
		AccessToken: access, RefreshToken: refresh,
		ExpiresIn: int(accessTokenTTL.Seconds()), TokenType: "Bearer",
		UserID: userID, TenantID: tenantID,
	})
}

type oauthUserInfo struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	FullName       string
	FirstName      string
	LastName       string
	AvatarURL      string
	Raw            json.RawMessage
}

var oidcCache sync.Map

func (s *server) fetchOAuthUserInfo(ctx context.Context, oc *oauthCfg, tok map[string]any) (*oauthUserInfo, error) {
	access, _ := tok["access_token"].(string)
	if access == "" {
		return nil, errors.New("missing access_token")
	}
	provider := oc.provider
	if oc.discovery != "" {
		if rawID, ok := tok["id_token"].(string); ok && rawID != "" {
			if v := getCachedVerifier(oc.discovery, oc.clientID); v != nil {
				idToken, err := v.Verify(ctx, rawID)
				if err == nil {
					var claims struct {
						Sub, Email, Name, GivenName, FamilyName, Picture, Locale string
						EmailVerified bool
					}
					if err := idToken.Claims(&claims); err == nil {
						rawJSON, _ := json.Marshal(claims)
						return &oauthUserInfo{
							ProviderUserID: claims.Sub,
							Email:          claims.Email,
							EmailVerified:  claims.EmailVerified,
							FullName:       claims.Name,
							FirstName:      claims.GivenName,
							LastName:       claims.FamilyName,
							AvatarURL:      claims.Picture,
							Raw:            rawJSON,
						}, nil
					}
				}
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", oc.userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo %d: %s", resp.StatusCode, string(body))
	}
	var data map[string]any
	_ = json.Unmarshal(body, &data)
	raw, _ := json.Marshal(data)
	if provider == "facebook" {
		pic := ""
		if pp, ok := data["picture"].(map[string]any); ok {
			if inner, ok := pp["data"].(map[string]any); ok {
				pic = asString(inner["url"])
			}
		}
		return &oauthUserInfo{
			ProviderUserID: asString(data["id"]),
			Email:          asString(data["email"]),
			EmailVerified:  asString(data["email"]) != "",
			FullName:       asString(data["name"]),
			FirstName:      asString(data["first_name"]),
			LastName:       asString(data["last_name"]),
			AvatarURL:      pic,
			Raw:            raw,
		}, nil
	}
	return &oauthUserInfo{
		ProviderUserID: asString(data["sub"]),
		Email:          asString(data["email"]),
		EmailVerified:  asBool(data["email_verified"]),
		FullName:       asString(data["name"]),
		FirstName:      asString(data["given_name"]),
		LastName:       asString(data["family_name"]),
		AvatarURL:      asString(data["picture"]),
		Raw:            raw,
	}, nil
}

func getCachedVerifier(discovery, clientID string) *oidc.IDTokenVerifier {
	if v, ok := oidcCache.Load(discovery); ok {
		if p, ok2 := v.(*oidc.Provider); ok2 {
			return p.Verifier(&oidc.Config{ClientID: clientID})
		}
	}
	provider, err := oidc.NewProvider(context.Background(), discovery)
	if err != nil {
		return nil
	}
	oidcCache.Store(discovery, provider)
	return provider.Verifier(&oidc.Config{ClientID: clientID})
}

func exchangeCode(tokenURL, code, verifier, redirect, clientID, clientSecret string) (map[string]any, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", redirect)
	if clientID != "" {
		form.Set("client_id", clientID)
		form.Set("client_secret", clientSecret)
	}
	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint %d: %s", resp.StatusCode, string(body))
	}
	out := map[string]any{}
	return out, json.Unmarshal(body, &out)
}

// =============================================================================
// API key handlers
// =============================================================================

func (s *server) handleAPIKeysList(c echo.Context) error {
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	rows, err := s.pool.Query(c.Request().Context(), `
		SELECT id::text, tenant_id::text, user_id::text, name, prefix, scopes,
		       rate_limit, expires_at, created_at
		FROM auth.api_keys WHERE user_id=$1 AND revoked_at IS NULL
		ORDER BY created_at DESC LIMIT 200
	`, claims.UserID)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer rows.Close()
	keys := []apiKeyResp{}
	for rows.Next() {
		var k apiKeyResp
		if err := rows.Scan(&k.ID, &k.TenantID, &k.UserID, &k.Name, &k.Prefix, &k.Scopes,
			&k.RateLimit, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return jsonErr(c, 500, "DB_ERROR", err.Error())
		}
		keys = append(keys, k)
	}
	return c.JSON(http.StatusOK, keys)
}

func (s *server) handleAPIKeysCreate(c echo.Context) error {
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	var req apiKeyCreateReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	if req.Name == "" {
		return jsonErr(c, 400, "INVALID_REQUEST", "name required")
	}
	if req.Env == "" {
		req.Env = "live"
	}
	if req.RateLimit == 0 {
		req.RateLimit = 1000
	}
	var ttl time.Duration
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}
	key, err := generateAPIKeyRecord(claims.TenantID, claims.UserID, req.Name, req.Env, req.Scopes, req.RateLimit, ttl)
	if err != nil {
		return jsonErr(c, 500, "KEYGEN_FAILED", err.Error())
	}
	if _, err := s.pool.Exec(c.Request().Context(), `
		INSERT INTO auth.api_keys (id, tenant_id, user_id, name, prefix, hash, last_4, environment, scopes, rate_limit, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, key.Record.ID, key.Record.TenantID, key.Record.UserID, key.Record.Name, key.Record.Prefix,
		key.Record.Hash, key.Record.Last4, req.Env, key.Record.Scopes,
		key.Record.RateLimit, key.Record.ExpiresAt); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusCreated, apiKeyResp{
		ID: key.Record.ID, TenantID: key.Record.TenantID, UserID: key.Record.UserID, Name: key.Record.Name,
		Prefix: key.Record.Prefix, Scopes: key.Record.Scopes, RateLimit: key.Record.RateLimit,
		ExpiresAt: key.Record.ExpiresAt, CreatedAt: key.Record.CreatedAt,
		Key: key.Key, Last4: key.Record.Last4,
	})
}

func (s *server) handleAPIKeysRevoke(c echo.Context) error {
	claims, err := s.tryExtractClaims(c)
	if err != nil {
		return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
	}
	id := c.Param("id")
	if id == "" {
		return jsonErr(c, 400, "INVALID_REQUEST", "id required")
	}
	tag, err := s.pool.Exec(c.Request().Context(),
		`UPDATE auth.api_keys SET revoked_at=NOW(), revoked_reason='user_revoke' WHERE id=$1 AND user_id=$2`,
		id, claims.UserID)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if tag.RowsAffected() == 0 {
		return jsonErr(c, 404, "NOT_FOUND", "key not found")
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// =============================================================================
// Connect-RPC compatible handlers
// =============================================================================

func (s *server) rpcRegister(c echo.Context) error {
	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		FullName   string `json:"full_name"`
		TenantSlug string `json:"tenant_slug"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	hash, err := hashPwd(req.Password)
	if err != nil {
		return jsonErr(c, 500, "HASH_ERROR", err.Error())
	}
	ctx := c.Request().Context()
	userID := newID()
	tenantID := newID()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()
	slug := req.TenantSlug
	if slug == "" {
		slug = "rpc-" + newID()[:8]
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO tenant.tenants (id, slug, name, plan, status) VALUES ($1, $2, $3, 'starter', 'active')`,
		tenantID, slug, req.TenantSlug); err != nil {
		if isUniqueViolation(err) {
			if err := tx.QueryRow(ctx, `SELECT id::text FROM tenant.tenants WHERE slug=$1`, slug).Scan(&tenantID); err != nil {
				return jsonErr(c, 400, "TENANT_ERROR", err.Error())
			}
		} else {
			return jsonErr(c, 500, "DB_ERROR", err.Error())
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, email_verified, status)
		VALUES ($1, $2, $3, $4, $5, FALSE, 'active')
	`, userID, tenantID, strings.ToLower(req.Email), hash, req.FullName); err != nil {
		if isUniqueViolation(err) {
			return jsonErr(c, 409, "USER_EXISTS", err.Error())
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"user_id": userID, "tenant_id": tenantID})
}

func (s *server) rpcLogin(c echo.Context) error {
	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		TenantSlug string `json:"tenant_slug"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	var (
		userID, tenantID, passwordHash string
		isSuperAdmin                  bool
	)
	if err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.tenant_id::text, COALESCE(u.password_hash,''), COALESCE(u.is_super_admin, false)
		FROM auth.users u
		JOIN tenant.tenants t ON t.id = u.tenant_id
		WHERE LOWER(u.email) = $1 AND ($2 = '' OR t.slug = $2) AND u.deleted_at IS NULL LIMIT 1
	`, strings.ToLower(req.Email), req.TenantSlug).Scan(&userID, &tenantID, &passwordHash, &isSuperAdmin); err != nil {
		return jsonErr(c, 401, "INVALID_CREDENTIALS", "invalid credentials")
	}
	ok, err := verifyPwd(req.Password, passwordHash)
	if err != nil || !ok {
		return jsonErr(c, 401, "INVALID_CREDENTIALS", "invalid credentials")
	}
	access, refresh, err := s.issueTokens(ctx, userID, tenantID, "", c.RealIP(), c.Request().UserAgent(), isSuperAdmin)
	if err != nil {
		return jsonErr(c, 500, "TOKEN_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, tokenResp{
		AccessToken: access, RefreshToken: refresh,
		ExpiresIn: int(accessTokenTTL.Seconds()), TokenType: "Bearer",
		UserID: userID, TenantID: tenantID,
	})
}

func (s *server) rpcValidateToken(c echo.Context) error {
	var req struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	claims, err := s.keyRing.Decrypt(req.AccessToken)
	if err != nil {
		return jsonErr(c, 401, "INVALID_TOKEN", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"valid": true, "user_id": claims.UserID, "tenant_id": claims.TenantID, "roles": claims.Roles})
}

func (s *server) rpcRevokeToken(c echo.Context) error {
	var req struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	claims, err := s.keyRing.Decrypt(req.AccessToken)
	if err != nil {
		return jsonErr(c, 401, "INVALID_TOKEN", err.Error())
	}
	if _, err := s.pool.Exec(c.Request().Context(),
		`UPDATE auth.sessions SET revoked_at=NOW(), revoked_reason='rpc_revoke' WHERE user_id=$1`,
		claims.UserID); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) rpcGetUser(c echo.Context) error {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return jsonErr(c, 400, "INVALID_REQUEST", err.Error())
	}
	var u userResp
	if err := s.pool.QueryRow(c.Request().Context(), `
		SELECT id::text, tenant_id::text, email, COALESCE(full_name,''), COALESCE(avatar_url,''),
		       COALESCE(phone,''), email_verified, status, mfa_enabled,
		       COALESCE(is_super_admin, false), last_login_at, created_at
		FROM auth.users WHERE id=$1 AND deleted_at IS NULL
	`, req.UserID).Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.AvatarURL,
		&u.Phone, &u.EmailVerified, &u.Status, &u.MFAEnabled, &u.SuperAdmin, &u.LastLoginAt, &u.CreatedAt); err != nil {
		return jsonErr(c, 404, "USER_NOT_FOUND", err.Error())
	}
	return c.JSON(http.StatusOK, u)
}

// =============================================================================
// Middleware
// =============================================================================

func (s *server) requireAuth(_ string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, err := s.tryExtractClaims(c)
			if err != nil {
				return jsonErr(c, 401, "MISSING_TOKEN", err.Error())
			}
			isAdmin := claims.Scope == "super_admin" || (len(claims.Roles) > 0 && claims.Roles[0] == "super_admin")
			ctx := platform.WithTenant(c.Request().Context(), claims.TenantID, claims.UserID, isAdmin)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func (s *server) tryExtractClaims(c echo.Context) (*rincoauth.Claims, error) {
	h := c.Request().Header.Get("Authorization")
	if h == "" || !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("missing bearer token")
	}
	tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	if tok == "" {
		return nil, errors.New("empty token")
	}
	return s.keyRing.Decrypt(tok)
}

// =============================================================================
// Token issuance
// =============================================================================

func (s *server) issueTokens(ctx context.Context, userID, tenantID, deviceFP, ip, ua string, isSuperAdmin bool) (access, refresh string, err error) {
	access, err = s.encodeAccessToken(userID, tenantID, isSuperAdmin)
	if err != nil {
		return "", "", err
	}
	refresh, err = rincoauth.GenerateRandomToken(32)
	if err != nil {
		return "", "", err
	}
	hash := rincoauth.SHA256Hex([]byte(refresh))
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO auth.sessions (id, user_id, tenant_id, refresh_token_hash, ip, user_agent, device_fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, newID(), userID, tenantID, hash, ip, ua, deviceFP, time.Now().Add(refreshTTL)); err != nil {
		return "", "", err
	}
	if s.rdb != nil {
		_ = s.rdb.Set(ctx, "session:"+userID, refresh, refreshTTL).Err()
	}
	return access, refresh, nil
}

func (s *server) encodeAccessToken(userID, tenantID string, isSuperAdmin bool) (string, error) {
	roles := []string{"user"}
	scope := ""
	if isSuperAdmin {
		roles = []string{"super_admin"}
		scope = "super_admin"
	}
	now := time.Now()
	return s.keyRing.Encrypt(rincoauth.Claims{
		Issuer: "rinco", Audience: "rinco-app",
		Subject: userID, UserID: userID, TenantID: tenantID,
		Roles: roles, Scope: scope,
		IssuedAt: now, NotBefore: now, ExpiresAt: now.Add(accessTokenTTL),
		JTI: newID(),
	})
}

// =============================================================================
// Password helpers (Argon2id — independent of packages/go)
// =============================================================================

func hashPwd(pwd string) (string, error) {
	salt := make([]byte, argonParams.saltLen)
	if _, err := readRand(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pwd), salt, argonParams.iter, argonParams.mem, argonParams.par, argonParams.keyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonParams.mem, argonParams.iter, argonParams.par,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func verifyPwd(pwd, encoded string) (bool, error) {
	if !strings.HasPrefix(encoded, "$argon2id$") {
		return false, errors.New("not argon2id hash")
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false, errors.New("malformed")
	}
	var mem, iter uint32
	var par uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &par); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(pwd), salt, iter, mem, par, uint32(len(expected)))
	return subtle.ConstantTimeCompare(expected, got) == 1, nil
}

// =============================================================================
// API-key helper (minimal local copy of NewRecord equivalent)
// =============================================================================

type apiKeyRecord struct {
	Record struct {
		ID, TenantID, UserID, Name, Prefix, Hash, Last4, Env string
		Scopes                                              []string
		RateLimit                                           int
		ExpiresAt                                           *time.Time
		CreatedAt                                           time.Time
	}
	Key string
}

func generateAPIKeyRecord(tenantID, userID, name, env string, scopes []string, rateLimit int, ttl time.Duration) (*apiKeyRecord, error) {
	if env == "" {
		env = "live"
	}
	if rateLimit == 0 {
		rateLimit = 1000
	}
	plain, err := rincoauth.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}
	id := newID()
	now := time.Now()
	var exp *time.Time
	if ttl > 0 {
		t := now.Add(ttl)
		exp = &t
	}
	prefix := fmt.Sprintf("rinco_%s_%s", env, plain[len(plain)-8:][:5])
	r := &apiKeyRecord{}
	r.Record.ID = id
	r.Record.TenantID = tenantID
	r.Record.UserID = userID
	r.Record.Name = name
	r.Record.Prefix = prefix
	r.Record.Hash = sha256Hex(plain)
	r.Record.Last4 = last4(plain)
	r.Record.Env = env
	r.Record.Scopes = scopes
	r.Record.RateLimit = rateLimit
	r.Record.ExpiresAt = exp
	r.Record.CreatedAt = now
	r.Key = plain
	return r, nil
}

// =============================================================================
// Echo middleware (minimal equivalents, locally scoped)
// =============================================================================

func recoveryMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic recovered", slog.Any("panic", r), slog.String("path", c.Request().URL.Path))
					err = echo.NewHTTPError(500, "internal error")
				}
			}()
			return next(c)
		}
	}
}

func traceMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get("X-Trace-ID")
			if id == "" {
				id = newID()
			}
			c.Response().Header().Set("X-Trace-ID", id)
			ctx := platform.WithTraceID(c.Request().Context(), id)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func loggingMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			status := c.Response().Status
			if err != nil || status >= 400 {
				slog.Warn("request", slog.String("method", c.Request().Method), slog.String("path", c.Request().URL.Path), slog.Int("status", status), slog.Duration("dur", time.Since(start)))
			} else {
				slog.Info("request", slog.String("method", c.Request().Method), slog.String("path", c.Request().URL.Path), slog.Int("status", status), slog.Duration("dur", time.Since(start)))
			}
			return err
		}
	}
}

func metricsMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			status := "0"
			if c.Response() != nil {
				status = strconv.Itoa(c.Response().Status)
			}
			httpReqs.WithLabelValues(c.Request().Method, c.Path(), status).Inc()
			httpDur.WithLabelValues(c.Request().Method, c.Path()).Observe(time.Since(start).Seconds())
			return err
		}
	}
}

func corsMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			if origin != "" {
				h := c.Response().Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
				h.Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Trace-ID,X-Tenant-ID")
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Max-Age", "86400")
			}
			if c.Request().Method == "OPTIONS" {
				return c.NoContent(204)
			}
			return next(c)
		}
	}
}

func securityHeadersMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			return next(c)
		}
	}
}

// =============================================================================
// Misc helpers
// =============================================================================

func jsonErr(c echo.Context, status int, code, msg string) error {
	return c.JSON(status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func last4(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[len(s)-4:]
}

func readRand(b []byte) (int, error) {
	return rand.Read(b)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}

func firstNonEmpty(args ...string) string {
	for _, a := range args {
		if a != "" {
			return a
		}
	}
	return ""
}

func newID() string {
	u, err := uuid.NewV7()
	if err != nil {
		return uuid.New().String()
	}
	return u.String()
}

func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func (s *server) audit(ctx context.Context, tenantID, userID, ip, ua, event, outcome, resource string) {
	if s.pool == nil {
		return
	}
	meta := []byte("{}")
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO auth.audit_logs (tenant_id, user_id, actor_ip, actor_user_agent, event, outcome, resource, metadata)
		VALUES ($1, $2, NULLIF($3,'')::inet, $4, $5, $6, NULLIF($7,''), $8)
	`, nullUUID(tenantID), nullUUID(userID), ip, ua, event, outcome, resource, meta)
	slog.Info("audit", slog.String("event", event), slog.String("outcome", outcome))
}

func genState(n int) (string, error) { return rincoauth.GenerateRandomToken(n) }

func genPKCE() (verifier, challenge string, err error) {
	v, err := rincoauth.GenerateRandomToken(32)
	if err != nil {
		return "", "", err
	}
	h := sha256.Sum256([]byte(v))
	return v, base64.RawURLEncoding.EncodeToString(h[:]), nil
}
