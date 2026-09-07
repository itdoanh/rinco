# Auth Service

Authentication, authorization, WebAuthn, OAuth, and API key management for the RINCO platform.

## Highlights

- **Email/password auth** with Argon2id (high parameters: m=64 MiB, t=3, p=4) and auto-tenant on first user
- **PASETO-like tokens** (HMAC-SHA256, base64url) — access (15 min) + rotating refresh (30 days)
- **WebAuthn / FIDO2** registration & login (TouchID, YubiKey, Windows Hello, ...)
- **OAuth** social login (Google, Facebook, Microsoft) with PKCE state in Valkey
- **API keys** with prefix, SHA-256 hash, scopes, expiry, last-used tracking
- **Multi-tenant** isolation via PostgreSQL `current_tenant_id` (RLS-ready)
- **Session revocation** with refresh-token hashing and previous-id chain
- **Password reset** with one-time token (24 h TTL)
- **Structured logging** (`slog`), **OpenTelemetry** tracing, **Prometheus** metrics
- **Connect-RPC** handlers for internal service-to-service calls
- **Graceful shutdown** (SIGINT/SIGTERM, drain 30 s)

## Tech

- Go 1.23+ • Echo v4 • pgx/v5 • redis/go-redis/v9 • go-webauthn/webauthn v0.10
- OpenTelemetry SDK + otelecho middleware
- prometheus/client_golang

## Environment

| Variable | Default | Description |
|---|---|---|
| `AUTH_HTTP_ADDR` | `:8081` | HTTP listen address |
| `AUTH_DATABASE_URL` | _required_ | PostgreSQL DSN |
| `AUTH_VALKEY_URL` | `redis://localhost:6379/0` | Redis/Valkey for sessions, WebAuthn challenges, OAuth state |
| `AUTH_PASETO_KEY` | _required_ | HMAC key (≥ 32 bytes, hex) |
| `AUTH_PASETO_PUBLIC` | _required_ | Public HMAC key for validation (hex) |
| `AUTH_WEB_BASE_URL` | `http://localhost:8081` | Used in OAuth callbacks |
| `AUTH_RP_ID` | `localhost` | WebAuthn Relying Party ID |
| `AUTH_RP_ORIGIN` | `http://localhost:8081` | WebAuthn RP origin (comma-separated) |
| `AUTH_ACCESS_TTL` | `15m` | Access token lifetime |
| `AUTH_REFRESH_TTL` | `720h` | Refresh token lifetime (30 d) |
| `AUTH_OAUTH_GOOGLE_CLIENT_ID` / `_SECRET` | _optional_ | Google OAuth |
| `AUTH_OAUTH_FACEBOOK_CLIENT_ID` / `_SECRET` | _optional_ | Facebook OAuth |
| `AUTH_OAUTH_MICROSOFT_CLIENT_ID` / `_SECRET` | _optional_ | Microsoft OAuth |

See `.env.example` for a full template.

## Database

Migrations are embedded and run on startup. Tables (schema `auth`):

- `users` (id, tenant_id, email, password_hash, full_name, email_verified, status, ...)
- `sessions` (id, user_id, tenant_id, refresh_token_hash, ip, user_agent, expires_at, previous_id)
- `password_resets` (id, user_id, token_hash, expires_at, used_at)
- `webauthn_credentials` (id, user_id, tenant_id, credential_id, public_key, counter, transports, name)
- `webauthn_challenges` (key, flow, user_id, tenant_id, challenge, user_handle, expires_at)
- `oauth_accounts` (id, user_id, tenant_id, provider, provider_user_id, provider_email, raw_profile)
- `api_keys` (id, user_id, tenant_id, prefix, hash, name, scopes, last_used_at, expires_at, revoked_at)

If schema is empty, the service auto-creates the `auth` schema and tenant schema `tenant`.

## HTTP API

Base URL: `http://localhost:8081`. All requests/responses are JSON.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/healthz` | – | Liveness |
| GET | `/readyz` | – | DB + Valkey ping |
| GET | `/metrics` | – | Prometheus metrics |
| POST | `/v1/auth/register` | – | Create user (auto-tenant if first) |
| POST | `/v1/auth/login` | – | Email/password → tokens |
| POST | `/v1/auth/refresh` | – | Rotate refresh token |
| POST | `/v1/auth/logout` | – | Revoke current session |
| GET | `/v1/auth/me` | Bearer | Current user profile |
| PUT | `/v1/auth/me` | Bearer | Update profile (name, locale, tz, avatar) |
| POST | `/v1/auth/password/change` | Bearer | Change password (old + new) |
| POST | `/v1/auth/password/reset/request` | – | Email a reset token (returns token if no SMTP) |
| POST | `/v1/auth/password/reset/confirm` | – | Consume token + new password |
| POST | `/v1/auth/webauthn/register/begin` | Bearer | Start WebAuthn registration |
| POST | `/v1/auth/webauthn/register/finish` | Bearer | Verify & persist credential |
| POST | `/v1/auth/webauthn/login/begin` | – | Start WebAuthn login |
| POST | `/v1/auth/webauthn/login/finish` | – | Verify & issue tokens |
| GET | `/v1/auth/oauth/:provider/start` | – | Redirect to provider (PKCE) |
| GET | `/v1/auth/oauth/:provider/callback` | – | Provider → tokens |
| GET | `/v1/auth/api-keys` | Bearer | List keys (only prefix/name shown) |
| POST | `/v1/auth/api-keys` | Bearer | Create key (returns full key ONCE) |
| DELETE | `/v1/auth/api-keys/:id` | Bearer | Revoke key |

## Connect-RPC (internal)

`POST /rpc/auth.v1.AuthService/<Method>` with `application/json` or `application/proto`.

| Method | Request | Response |
|---|---|---|
| `Register` | `email, password, full_name` | `user_id, tenant_id` |
| `Login` | `email, password, ip, user_agent` | `access_token, refresh_token, user_id, tenant_id` |
| `ValidateToken` | `access_token` | `user_id, tenant_id, roles, scope, expires_at` |
| `RevokeToken` | `access_token` | `ok` |
| `GetUser` | `user_id` | `email, full_name, status, tenant_id` |
| `ProvisionUser` | `email, full_name, tenant_id` | `user_id` |

All RPC handlers log via slog with `trace_id` from the request context.

## Curl examples

```bash
# Health
curl -s localhost:8081/healthz

# Register
curl -s -X POST localhost:8081/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"correct horse battery staple","full_name":"Alice"}'

# Login
TOKENS=$(curl -s -X POST localhost:8081/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"correct horse battery staple"}')
ACCESS=$(echo "$TOKENS" | jq -r .access_token)
REFRESH=$(echo "$TOKENS" | jq -r .refresh_token)

# Me
curl -s localhost:8081/v1/auth/me -H "Authorization: Bearer $ACCESS"

# Refresh
curl -s -X POST localhost:8081/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH\"}"

# Logout
curl -s -X POST localhost:8081/v1/auth/logout \
  -H "Authorization: Bearer $ACCESS"

# Change password
curl -s -X POST localhost:8081/v1/auth/password/change \
  -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' \
  -d '{"old_password":"correct horse battery staple","new_password":"new strong passw0rd!"}'

# Password reset
TOKEN=$(curl -s -X POST localhost:8081/v1/auth/password/reset/request \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com"}' | jq -r .token)
curl -s -X POST localhost:8081/v1/auth/password/reset/confirm \
  -H 'Content-Type: application/json' \
  -d "{\"token\":\"$TOKEN\",\"new_password\":\"new strong passw0rd!\"}"

# WebAuthn register begin
curl -s -X POST localhost:8081/v1/auth/webauthn/register/begin \
  -H "Authorization: Bearer $ACCESS" -H 'Content-Type: application/json' -d '{}'

# WebAuthn register finish (use a WebAuthn client lib to produce the response)
curl -s -X POST localhost:8081/v1/auth/webauthn/register/finish \
  -H "Authorization: Bearer $ACCESS" -H 'Content-Type: application/json' \
  -d '{"response_name":"attestation","response":{...}}'

# OAuth Google start (302 redirect)
curl -i localhost:8081/v1/auth/oauth/google/start

# API keys
curl -s localhost:8081/v1/auth/api-keys -H "Authorization: Bearer $ACCESS"
curl -s -X POST localhost:8081/v1/auth/api-keys \
  -H "Authorization: Bearer $ACCESS" -H 'Content-Type: application/json' \
  -d '{"name":"ci","scopes":["read"],"expires_in_days":90}'
# Response: { "id": "...", "key": "rko_xxx_SOMETHING", "prefix": "rko_xxx", "name": "ci", "scopes": ["read"] }
curl -s -X DELETE localhost:8081/v1/auth/api-keys/<id> -H "Authorization: Bearer $ACCESS"

# Connect-RPC: validate token (called by other services)
curl -s -X POST localhost:8081/rpc/auth.v1.AuthService/ValidateToken \
  -H 'Content-Type: application/json' \
  -d "{\"access_token\":\"$ACCESS\"}"
```

## Observability

- All logs go through `slog` (JSON) with `service=auth-service`, `trace_id`, `span_id`.
- Every request gets a `X-Trace-ID` header; if absent one is generated.
- `/metrics` exposes `http_requests_total`, `http_request_duration_seconds`, and process metrics.
- Tracing is initialized with OTLP HTTP exporter (no-op if `OTEL_EXPORTER_OTLP_ENDPOINT` unset).

## Development

```bash
# Tidy
go mod tidy

# Build
go build -o bin/auth-service ./cmd/main.go

# Run locally
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable \
AUTH_PASETO_KEY=$(openssl rand -hex 32) \
AUTH_PASETO_PUBLIC=$(openssl rand -hex 32) \
./bin/auth-service

# Vet
go vet ./...

# Test (no tests yet — placeholders)
go test ./...
```

## Docker

```bash
docker build -t rinco/auth-service:latest .
docker run --rm -p 8081:8081 \
  -e AUTH_DATABASE_URL=postgres://... \
  -e AUTH_PASETO_KEY=... \
  -e AUTH_PASETO_PUBLIC=... \
  rinco/auth-service:latest
```
