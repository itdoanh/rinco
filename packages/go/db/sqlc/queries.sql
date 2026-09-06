-- queries.sql - các câu query để sqlc generate Go functions.
-- Sau khi chạy `sqlc generate` sẽ có file queries.sql.go với:
--   GetUser(ctx, id) (User, error)
--   GetUserByEmail(ctx, ...) ...
--   CreateUser(ctx, ...) ...
--   v.v.

-- name: GetUser :one
SELECT * FROM auth.users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM auth.users WHERE tenant_id = $1 AND email = $2 AND deleted_at IS NULL LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM auth.users WHERE id = $1 LIMIT 1;

-- name: ListUsersByTenant :many
SELECT * FROM auth.users WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2;

-- name: CreateUser :one
INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, roles)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE auth.users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE auth.users SET password_hash = $1, updated_at = NOW() WHERE id = $2;

-- name: SoftDeleteUser :exec
UPDATE auth.users SET deleted_at = NOW() WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO auth.refresh_tokens (id, user_id, token_hash, device_fingerprint, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetRefreshToken :one
SELECT rt.*, u.tenant_id, u.roles
FROM auth.refresh_tokens rt
JOIN auth.users u ON u.id = rt.user_id
WHERE rt.token_hash = $1 AND rt.revoked_at IS NULL AND rt.expires_at > NOW()
LIMIT 1;

-- name: RevokeRefreshTokensByUser :exec
UPDATE auth.refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CreateAPIKey :one
INSERT INTO auth.api_keys (id, tenant_id, user_id, name, prefix, hash, last_4, scopes, rate_limit, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM auth.api_keys WHERE hash = $1 AND revoked_at IS NULL LIMIT 1;

-- name: ListAPIKeysByTenant :many
SELECT * FROM auth.api_keys WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: RevokeAPIKey :exec
UPDATE auth.api_keys SET revoked_at = NOW() WHERE id = $1;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE auth.api_keys SET last_used_at = NOW() WHERE id = $1;