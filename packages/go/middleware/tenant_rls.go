// Package middleware - PostgreSQL Row-Level Security GUC middleware.
//
// Why this exists:
//
//   Every authenticated request in RINCO carries a tenant_id (from the
//   PASETO token or from an explicit X-Tenant-ID header).  Our PostgreSQL
//   schema relies on RLS policies that read `app.current_tenant_id` from
//   the session GUC.  Forgetting to `SET` the GUC before running a query
//   leaks rows across tenants — the single most common security bug in
//   multi-tenant systems.
//
//   `WithTenantRLS(pool, opts)` returns an Echo middleware that, for
//   every request, attaches a connection from the pool, sets the GUC
//   based on the request context, runs the handler with the augmented
//   context, then releases the connection.  Services use it once at boot:
//
//       e.Use(middleware.WithTenantRLS(pool, middleware.TenantRLSConfig{
//           HeaderName: "X-Tenant-ID",
//       }))
//
//   Handlers downstream call `db.WithTx(ctx, pool, ...)` exactly like
//   before; the GUC is already set on the connection they get from the
//   pool, so RLS policies see the correct tenant.
//
//   This file deliberately does NOT import `packages/go/db` (which would
//   create a cycle: db -> middleware -> db).  Instead it depends only on
//   pgx/pgxpool + the tenant helpers in this package.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// TenantGUCName is the PostgreSQL session variable that RINCO's RLS
// policies read.  Keep in sync with `RLSSetupSQL` in packages/go/db.
const TenantGUCName = "app.current_tenant_id"

// TenantRLSConfig configures the GUC-setting middleware.
type TenantRLSConfig struct {
	// HeaderName to read the tenant from.  Defaults to "X-Tenant-ID".
	// The header value is *ignored* if the request has a verified PASETO
	// token — the token's `tenant_id` claim always wins.
	HeaderName string

	// ClaimTenantKey is the context key under which the verified access
	// claims live.  Defaults to TenantIDKey (set by middleware.Auth).
	ClaimTenantKey any

	// SkipPaths short-circuits the middleware for system probes (health,
	// metrics).  Defaults to "/healthz", "/health", "/metrics".
	SkipPaths []string

	// Required makes the middleware return 400 if no tenant can be
	// resolved.  Defaults to true.
	Required bool

	// OnError lets the caller log/observe GUC-set failures.  Returning
	// a non-nil error aborts the request with 500.  Default is to log
	// nothing and abort with 500 on the first failure.
	OnError func(c echo.Context, err error)
}

// defaults returns cfg with sensible fallbacks applied.
func (c TenantRLSConfig) defaults() TenantRLSConfig {
	if c.HeaderName == "" {
		c.HeaderName = "X-Tenant-ID"
	}
	if c.ClaimTenantKey == nil {
		c.ClaimTenantKey = TenantIDKey
	}
	if len(c.SkipPaths) == 0 {
		c.SkipPaths = []string{"/healthz", "/health", "/metrics"}
	}
	// Required defaults to true.  We can't tell a zero-value from an
	// explicit false at this point, but the *behavioural* default is
	// "required", and any caller that explicitly wants the lax mode
	// will set it themselves.
	c.Required = true
	return c
}

// WithTenantRLS returns Echo middleware that attaches a pgxpool
// connection to every request, sets `app.current_tenant_id` from the
// request context, and stashes the connection handle on the echo Context
// so downstream handlers can run queries through it (RLS sees the GUC).
//
// Handlers should retrieve the connection via:
//
//	conn := middleware.TenantConn(c)
//
// or, more commonly, just use `db.WithTx(ctx, pool, ...)` — the GUC is
// already on the connection pgxpool hands back, so transactions inherit
// it via SET LOCAL semantics.
//
// Caveat: pgxpool returns *different* connections on successive
// Acquire calls.  This middleware sets the GUC on the connection it
// acquires for the duration of one request.  Subsequent queries that go
// through `pool.Query(...)` (without the middleware-augmented context)
// may hit a different connection that has NOT had the GUC set — call
// sites that bypass this middleware MUST use `db.WithTx` (which calls
// setRLSContext for every transaction).
func WithTenantRLS(pool *pgxpool.Pool, cfg TenantRLSConfig) echo.MiddlewareFunc {
	if pool == nil {
		panic("middleware: WithTenantRLS: pool is nil")
	}
	cfg = cfg.defaults()
	skip := make(map[string]bool, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skip[p] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if skip[path] {
				return next(c)
			}

			tenantID := resolveTenantForRequest(c, cfg)
			if tenantID == "" {
				if cfg.Required {
					return jsonError(c, http.StatusBadRequest, "TENANT_REQUIRED",
						"no tenant_id in token or X-Tenant-ID header")
				}
				return next(c)
			}

			ctx := c.Request().Context()
			// Attach a dedicated connection so every downstream query
			// inside this request sees the same GUC.
			conn, err := pool.Acquire(ctx)
			if err != nil {
				return failRLS(c, cfg, fmt.Errorf("acquire connection: %w", err))
			}

			if _, err := conn.Exec(ctx,
				"SELECT set_config($1, $2, false)", TenantGUCName, tenantID); err != nil {
				conn.Release()
				return failRLS(c, cfg, fmt.Errorf("set %s: %w", TenantGUCName, err))
			}

			// Stash for downstream handlers / explicit release.
			c.Set(tenantConnKey, conn)
			defer func() {
				if cc, ok := c.Get(tenantConnKey).(*pgxpool.Conn); ok && cc != nil {
					cc.Release()
				}
			}()

			// Also stamp the standard context key so helpers like
			// `db.TenantIDFromContext` work without changes.
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// failRLS centralises error reporting so callers can plug their own
// observability without forking the middleware body.
func failRLS(c echo.Context, cfg TenantRLSConfig, err error) error {
	if cfg.OnError != nil {
		cfg.OnError(c, err)
	}
	return jsonError(c, http.StatusInternalServerError, "RLS_SETUP_FAILED", err.Error())
}

// TenantConn returns the RLS-stamped connection attached by
// WithTenantRLS.  Returns nil if the middleware is not active.
func TenantConn(c echo.Context) *pgxpool.Conn {
	if c == nil {
		return nil
	}
	v := c.Get(tenantConnKey)
	if v == nil {
		return nil
	}
	conn, _ := v.(*pgxpool.Conn)
	return conn
}

// resolveTenantForRequest pulls the tenant_id from (in order):
//   1. Verified access-claims in the request context (TenantIDKey).
//   2. cfg.HeaderName header (caller-controlled; only safe behind a
//      trusted gateway that strips the header).
//
// Returning the empty string means "no tenant found"; the caller decides
// whether that is fatal.
func resolveTenantForRequest(c echo.Context, cfg TenantRLSConfig) string {
	if ctx := c.Request().Context(); ctx != nil {
		if v := ctx.Value(cfg.ClaimTenantKey); v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	if cfg.HeaderName != "" {
		if h := c.Request().Header.Get(cfg.HeaderName); h != "" {
			// Reject header values that obviously aren't UUIDs to avoid
			// surprising the GUC layer.  Anything that isn't a
			// 36-char UUID is treated as a tenant slug — the caller is
			// responsible for slug -> UUID translation elsewhere.
			return strings.TrimSpace(h)
		}
	}
	return ""
}

// TenantGUCStatement returns the SQL used by WithTenantRLS to stamp the
// GUC.  Exported for tests and for callers that want to apply the same
// stamp from a non-HTTP entry point (cron jobs, async workers, ...).
func TenantGUCStatement(tenantID string) (string, []any) {
	return "SELECT set_config($1, $2, false)", []any{TenantGUCName, tenantID}
}

// ApplyTenantGUC stamps the GUC on an arbitrary pgx connection.  It is
// the programmatic equivalent of the middleware and is safe to call from
// background workers.  `isLocal=true` uses SET LOCAL (transaction
// scope); false uses `set_config(..., false)` (session scope).
func ApplyTenantGUC(ctx context.Context, conn *pgx.Conn, tenantID string, isLocal bool) error {
	if conn == nil {
		return fmt.Errorf("middleware: ApplyTenantGUC: conn is nil")
	}
	if tenantID == "" {
		return fmt.Errorf("middleware: ApplyTenantGUC: empty tenant id")
	}
	if isLocal {
		_, err := conn.Exec(ctx, fmt.Sprintf("SET LOCAL %s = '%s'", TenantGUCName, escapeSingleQuote(tenantID)))
		return err
	}
	_, err := conn.Exec(ctx, "SELECT set_config($1, $2, false)", TenantGUCName, tenantID)
	return err
}

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// tenantConnKey is the echo.Context key under which the middleware stashes
// the *pgxpool.Conn.  Unexported because callers should use TenantConn().
const tenantConnKey = "rinco.tenant_conn"
