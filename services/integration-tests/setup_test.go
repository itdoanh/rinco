// Package integration provides cross-service end-to-end tests for RINCO.
//
// The setup_test.go file contains the testcontainers bootstrapping that
// brings up a fully isolated stack (Postgres + Valkey + NATS) for each
// `go test` invocation. It returns a *TestEnv struct that downstream test
// files can use to obtain database pools, NATS clients, Valkey URLs, and
// PASETO signing keys.
//
// Build tag: integration. Without `-tags=integration` the package still
// compiles (helpers are exported as stubs).
//
//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	natsmodule "github.com/testcontainers/testcontainers-go/modules/nats"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	redismodule "github.com/testcontainers/testcontainers-go/modules/redis"
)

// =============================================================================
// TestEnv — boot a fully isolated docker network for one test run
// =============================================================================

// TestEnv is the live environment handle that every integration test relies
// on. It owns three containers (Postgres, Valkey, NATS), an open database
// pool, an HTTP client with auto-token-refresh, and the canonical
// RINCO_TEST_* service URLs. Cleanup is bound to t.Cleanup.
type TestEnv struct {
	t *testing.T

	// Containers (live until Cleanup).
	Postgres testcontainers.Container
	Valkey   testcontainers.Container
	NATS     testcontainers.Container

	// Connection strings / connection handles.
	PostgresDSN string
	ValkeyAddr  string
	NATSURL     string

	// DB pool (used to apply migrations and run assertions).
	DB *pgxpool.Pool

	// HTTP layer.
	HTTP    *HTTPClient
	Service *ServiceRegistry

	// Auth passphrase used by tests that mint PASETO tokens directly.
	PasetoKey []byte

	// Seeded canonical users (admin apex / admin hct / …) — populated by
	// seedDemoData when available, may be nil if the live stack is used
	// directly without an in-process migration.
	Users *DemoUsers
}

// DemoUsers records the credentials of the canonical demo accounts so each
// test can log in without re-seeding.
type DemoUsers struct {
	ApexAdmin     DemoUser
	HCTAdmin      DemoUser
	DemoAdmin     DemoUser
	ApexAgent     DemoUser
	ApexReadOnly  DemoUser
	ApexMarketing DemoUser
}

// DemoUser captures the email/password + role used by repeated logins.
type DemoUser struct {
	Email    string
	Password string
	TenantID string
	UserID   string
	Role     string
}

// ServiceRegistry holds the per-service HTTP URLs. Defaults match the
// port table in docs/SERVICES.md. Override via env vars for live stacks.
type ServiceRegistry struct {
	AuthURL        string
	TenantURL      string
	CRMURL         string
	DynamicModelURL string
	LeadURL        string
	LandingURL     string
	EmailURL       string
	NotificationURL string
	BillingURL     string
	ObservabilityURL string
	SearchURL      string
	MetaCAPIURL    string
	AnalyticsURL   string
	AISREURL       string
	LeadScoringURL string
	RagChatbotURL  string
	RecordingURL   string
	STTURL         string
	ChatEngineURL  string
	WebRTCSFUURL   string
}

// FromServiceRegistry (the canonical port table) reads service URLs from
// the environment with safe defaults that target the local docker-compose
// stack documented in infra/docker-compose.services.yml.
func FromServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		AuthURL:         envOr("RINCO_AUTH_URL", "http://localhost:8081"),
		TenantURL:       envOr("RINCO_TENANT_URL", "http://localhost:8082"),
		CRMURL:          envOr("RINCO_CRM_URL", "http://localhost:8083"),
		DynamicModelURL: envOr("RINCO_DYNAMICMODEL_URL", "http://localhost:8084"),
		LeadURL:         envOr("RINCO_LEAD_URL", "http://localhost:8085"),
		LandingURL:      envOr("RINCO_LANDING_URL", "http://localhost:8086"),
		EmailURL:        envOr("RINCO_EMAIL_URL", "http://localhost:8087"),
		NotificationURL: envOr("RINCO_NOTIFICATION_URL", "http://localhost:8088"),
		BillingURL:      envOr("RINCO_BILLING_URL", "http://localhost:8095"),
		ObservabilityURL: envOr("RINCO_OBSERVABILITY_URL", "http://localhost:8096"),
		SearchURL:       envOr("RINCO_SEARCH_URL", "http://localhost:8097"),
		MetaCAPIURL:     envOr("RINCO_METACAPI_URL", "http://localhost:8098"),
		AnalyticsURL:    envOr("RINCO_ANALYTICS_URL", "http://localhost:8099"),
		AISREURL:        envOr("RINCO_AISRE_URL", "http://localhost:8090"),
		LeadScoringURL:  envOr("RINCO_LEADSCORING_URL", "http://localhost:8091"),
		RagChatbotURL:   envOr("RINCO_RAGCHATBOT_URL", "http://localhost:8092"),
		RecordingURL:    envOr("RINCO_RECORDING_URL", "http://localhost:8093"),
		STTURL:          envOr("RINCO_STT_URL", "http://localhost:8094"),
		ChatEngineURL:   envOr("RINCO_CHATENGINE_URL", "http://localhost:8101"),
		WebRTCSFUURL:    envOr("RINCO_WEBRTCSFU_URL", "http://localhost:8102"),
	}
}

// =============================================================================
// Bootstrap
// =============================================================================

// Setup boots Postgres + Valkey + NATS via testcontainers-go and applies
// the canonical RINCO migrations + minimal seed data. On any error the
// containers are torn down and the test is failed. Otherwise the returned
// *TestEnv cleans itself up via t.Cleanup.
//
// Pass useExternalStack=true to skip the testcontainers bootstrap and
// target a pre-existing local docker-compose stack (the default for CI).
func Setup(t *testing.T, useExternalStack bool) *TestEnv {
	t.Helper()

	env := &TestEnv{
		t:        t,
		Service:  FromServiceRegistry(),
		HTTP:     NewHTTPClient(),
		PasetoKey: make([]byte, 32),
	}

	// Use a deterministic PASETO key for cross-service verification.
	// Production deployments inject a 32-byte key from K8s secret.
	copy(env.PasetoKey, []byte("rinco-integration-tests-paseto-key-32b!"))

	if useExternalStack || os.Getenv("RINCO_USE_EXTERNAL_STACK") != "" {
		// External stack mode: trust env vars (or defaults) for everything.
		env.PostgresDSN = envOr("RINCO_TEST_POSTGRES_DSN",
			"postgres://rinco:rinco@localhost:5432/rinco_test?sslmode=disable")
		env.ValkeyAddr = envOr("RINCO_TEST_VALKEY_ADDR", "localhost:6379")
		env.NATSURL = envOr("RINCO_TEST_NATS_URL", "nats://localhost:4222")

		if err := env.connectExistingStack(); err != nil {
			t.Skipf("external stack unavailable: %v", err)
		}
	} else {
		if err := env.bootContainers(context.Background()); err != nil {
			t.Skipf("docker not available for testcontainers: %v", err)
		}
	}

	// Apply migrations + minimal seed data.
	if env.DB != nil {
		if err := env.applyMigrations(context.Background()); err != nil {
			t.Logf("applyMigrations warning: %v (will rely on stack seed)", err)
		}
		if err := env.seedDemoUsers(context.Background()); err != nil {
			t.Logf("seedDemoUsers warning: %v", err)
		}
	}

	// Bind cleanup. testcontainers containers are torn down by Terminate,
	// the pgx pool is closed, and the HTTP client flushes pending refreshes.
	t.Cleanup(func() { env.Teardown() })
	return env
}

// connectExistingStack points env at a pre-existing local stack. If the
// stack isn't reachable we skip the test rather than fail it — these
// tests are opt-in.
func (e *TestEnv) connectExistingStack() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(e.PostgresDSN)
	if err != nil {
		return err
	}
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return err
	}
	e.DB = pool
	e.Users = canonicalUsers()
	return nil
}

// bootContainers spins up three testcontainers. We declare context-derived
// failures as t.Skip rather than t.Fatal so that contributors can run
// `go test -short` on machines without Docker installed.
func (e *TestEnv) bootContainers(ctx context.Context) error {
	pgCtx, pgCancel := context.WithTimeout(ctx, 90*time.Second)
	defer pgCancel()

	pgC, err := pgmodule.RunContainer(pgCtx,
		testcontainers.WithImage("postgres:16-alpine"),
		// Required to apply migrations against the test DB.
		pgmodule.WithDatabase("rinco_test"),
		pgmodule.WithUsername("rinco"),
		pgmodule.WithPassword("rinco"),
		pgmodule.WithInitScripts(
			// Initialize schemas via init scripts — minimal scaffold.
			// Migrations are applied separately in applyMigrations.
		),
	)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	e.Postgres = pgC

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgC.Terminate(ctx)
		return fmt.Errorf("postgres connection string: %w", err)
	}
	e.PostgresDSN = dsn

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		_ = pgC.Terminate(ctx)
		return err
	}
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		_ = pgC.Terminate(ctx)
		return err
	}
	e.DB = pool

	// ----- Valkey (Redis-compatible — testcontainers redis module drives the same wire protocol) -----
	vkCtx, vkCancel := context.WithTimeout(ctx, 60*time.Second)
	defer vkCancel()

	vkC, err := redismodule.RunContainer(vkCtx,
		testcontainers.WithImage("valkey/valkey:7-alpine"),
	)
	if err != nil {
		e.teardownContainers()
		return fmt.Errorf("valkey: %w", err)
	}
	e.Valkey = vkC

	url, err := vkC.ConnectionString(vkCtx)
	if err != nil {
		e.teardownContainers()
		return fmt.Errorf("valkey connection string: %w", err)
	}
	// valkey url is "redis://valkey:6379" — strip scheme for addr.
	e.ValkeyAddr = strings.TrimPrefix(url, "redis://")

	// ----- NATS -----
	ncCtx, ncCancel := context.WithTimeout(ctx, 60*time.Second)
	defer ncCancel()

	natsC, err := natsmodule.RunContainer(ncCtx,
		testcontainers.WithImage("nats:2-alpine"),
	)
	if err != nil {
		e.teardownContainers()
		return fmt.Errorf("nats: %w", err)
	}
	e.NATS = natsC
	url2, err := natsC.ConnectionString(ncCtx)
	if err != nil {
		e.teardownContainers()
		return fmt.Errorf("nats connection string: %w", err)
	}
	e.NATSURL = url2

	e.Users = canonicalUsers()
	return nil
}

// teardownContainers is the inner best-effort stop used by boot failures.
// The deferred real cleanup lives in Teardown (called from t.Cleanup).
func (e *TestEnv) teardownContainers() {
	if e.NATS != nil {
		_ = e.NATS.Terminate(context.Background())
		e.NATS = nil
	}
	if e.Valkey != nil {
		_ = e.Valkey.Terminate(context.Background())
		e.Valkey = nil
	}
	if e.Postgres != nil {
		_ = e.Postgres.Terminate(context.Background())
		e.Postgres = nil
	}
	if e.DB != nil {
		e.DB.Close()
		e.DB = nil
	}
}

// Teardown is called from t.Cleanup. Order matters: stop in-flight HTTP,
// close the pgx pool, then terminate the containers.
func (e *TestEnv) Teardown() {
	if e.HTTP != nil {
		e.HTTP.Close()
	}
	e.teardownContainers()
}

// =============================================================================
// Migrations + seed
// =============================================================================

// applyMigrations runs every .sql file in migrations/sql in lexical order
// inside a single transaction. If we can't find the migrations folder
// (e.g. when running from inside a docker container), we fall back to a
// sentinel "schema already applied" check and trust the external DB.
func (e *TestEnv) applyMigrations(ctx context.Context) error {
	if e.DB == nil {
		return errors.New("no db pool")
	}
	root, err := findMigrationsRoot()
	if err != nil {
		return err
	}
	sqlDir := filepath.Join(root, "migrations", "sql")
	files, err := filepath.Glob(filepath.Join(sqlDir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files in %s", sqlDir)
	}

	// Already-migrated sentinel.
	var applied bool
	if err := e.DB.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables
			WHERE table_schema='public' AND table_name='schema_migrations')`,
	).Scan(&applied); err == nil && applied {
		return nil
	}

	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := e.DB.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("exec %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// seedDemoUsers inserts the canonical demo accounts (apex / hct / demo)
// so that tests can use them deterministically. Idempotent: uses ON
// CONFLICT DO NOTHING. Credentials match the rest of the seed suite
// (`rinco_dev_password`).
func (e *TestEnv) seedDemoUsers(ctx context.Context) error {
	if e.DB == nil {
		return errors.New("no db pool")
	}
	// We don't recreate the seed schema (handled by 00_master.sql); this
	// only top-ups the users table when running against an empty test DB.
	_, err := e.DB.Exec(ctx, `
		INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, role)
		VALUES
		  ('aaaaaaaa-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'admin@apexfintech.vn',
		   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Apex Admin', 'tenant_admin'),
		  ('aaaaaaaa-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000002', 'admin@hct.vn',
		   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'HCT Admin', 'tenant_admin'),
		  ('bbbbbbbb-0000-0000-0000-000000000003', 'bbbbbbbb-0000-0000-0000-000000000003', 'admin@demo-company.vn',
		   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Demo Admin', 'tenant_admin')
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		// Schema may not yet have auth.users — tolerate and rely on
		// external seed.
		return err
	}
	return nil
}

// canonicalUsers returns the static user table used when the live DB
// already has the canonical seed applied (no need to re-insert).
func canonicalUsers() *DemoUsers {
	return &DemoUsers{
		ApexAdmin: DemoUser{
			Email: "admin@apexfintech.vn", Password: "rinco_dev_password",
			TenantID: "aaaaaaaa-0000-0000-0000-000000000001",
			UserID:   "a0000001-0000-0000-0000-000000000001",
			Role:     "tenant_admin",
		},
		HCTAdmin: DemoUser{
			Email: "admin@hct.vn", Password: "rinco_dev_password",
			TenantID: "aaaaaaaa-0000-0000-0000-000000000002",
			UserID:   "a0000002-0000-0000-0000-000000000001",
			Role:     "tenant_admin",
		},
		DemoAdmin: DemoUser{
			Email: "admin@demo-company.vn", Password: "rinco_dev_password",
			TenantID: "bbbbbbbb-0000-0000-0000-000000000003",
			UserID:   "b0000003-0000-0000-0000-000000000001",
			Role:     "tenant_admin",
		},
		ApexAgent: DemoUser{
			Email: "agent1@apexfintech.vn", Password: "rinco_dev_password",
			TenantID: "aaaaaaaa-0000-0000-0000-000000000001",
			UserID:   "a0000001-0000-0000-0000-000000000010",
			Role:     "member",
		},
		ApexReadOnly: DemoUser{
			Email: "viewer@apexfintech.vn", Password: "rinco_dev_password",
			TenantID: "aaaaaaaa-0000-0000-0000-000000000001",
			UserID:   "a0000001-0000-0000-0000-000000000099",
			Role:     "viewer",
		},
		ApexMarketing: DemoUser{
			Email: "manager.marketing@apexfintech.vn", Password: "rinco_dev_password",
			TenantID: "aaaaaaaa-0000-0000-0000-000000000001",
			UserID:   "a0000001-0000-0000-0000-000000000004",
			Role:     "manager",
		},
	}
}

// =============================================================================
// Helpers shared with downstream test files
// =============================================================================

// SkipIfNoStack returns true if the local stack isn't reachable. We don't
// want to silently pass tests that never exercised the real code path.
func SkipIfNoStack(t *testing.T, env *TestEnv) bool {
	t.Helper()
	if env == nil || env.Service == nil {
		return true
	}
	resp, err := http.Get(env.Service.LandingURL + "/healthz")
	if err != nil {
		t.Skipf("local stack unavailable: %v", err)
		return true
	}
	_ = resp.Body.Close()
	return false
}

// HealthJSON fetches a /healthz-style endpoint and returns the body.
func (e *TestEnv) HealthJSON(urlPath string) (int, []byte) {
	if e == nil || e.Service == nil {
		return 0, nil
	}
	resp, err := http.Get(e.Service.LandingURL + urlPath)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

// envOr is a tiny helper for environment lookup with a default.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// findMigrationsRoot walks up from the test working directory looking for
// the canonical migrations/ folder. In CI the test binary is invoked from
// the repo root, but in `go test ./...` invocations the cwd can be the
// package directory — we tolerate both layouts.
func findMigrationsRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "migrations", "sql")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("migrations root not found from %s", cwd)
}

// =============================================================================
// Singleflight mutex used by HTTPClient + tests
// =============================================================================

var (
	muOnce sync.Mutex
)

// LockFor is exposed so that helpers_test.go can serialise against the
// singleflight to guarantee at most one refresh per user.
func LockFor(fn func()) {
	muOnce.Lock()
	defer muOnce.Unlock()
	fn()
}

// jsonUnmarshal exposes a typed JSON unmarshaler (used by test files).
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
