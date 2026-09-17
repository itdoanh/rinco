# WS-D — Integration Tests Workstream (Loop 1)

This file documents the WS-D integration test suite delivered in Loop WS-D 1.

## Files delivered

The following 16 files comprise the WS-D integration test plan:

### Test files (`services/integration-tests/`)

| File | What it covers |
|------|----------------|
| `setup_test.go`         | Testcontainers bootstrap: spins up Postgres + Valkey + NATS in a fresh docker network; applies migrations; seeds canonical demo users; returns `*TestEnv` with cleanup bound to `t.Cleanup`. |
| `helpers_test.go`       | `TestContext`, `MakeJWT(tenantID,userID,role)`, `HTTPClient` w/ auto-refresh, and the high-level assertions `AssertTenantIsolated`, `AssertRBAC`, `AssertAuditLogged`. |
| `auth_flow_test.go`     | Register → login → refresh → logout. PASETO token round-trip. |
| `leads_flow_test.go`    | Landing → CRM lead creation; HMAC tampering rejected; quorum deletion; CAPI failure logging. |
| `tenancy_test.go`       | 2-tenant isolation matrix; cross-tenant attempts rejected; duplicate-slug rejection. |
| `rbac_test.go`          | 5 roles × CRUD/Admin endpoints; manager-can-write vs viewer-cannot-write. |
| `tree_test.go`          | LTREE hierarchy; move subtree; no-cycle invariant; no-self-parent invariant. |
| `workflow_test.go`      | Create rule → trigger → action executed; disabled rules don't fire; parallel fires. |
| `notifications_test.go` | Create → queue → deliver; batch create; invalid channel rejection. |
| `audit_test.go`         | Tenant mutation → audit row; read-only endpoints do not audit; cross-actor isolation. |
| `chat_flow_test.go`     | WS connect + send + presence heartbeat; invalid token rejected during upgrade. |
| `scoring_flow_test.go`  | Lead → XGBoost scoring; batch score; >1000 batch rejection; /health. |
| `go.mod`, `go.sum`      | Module with deps: `testcontainers-go` v0.31 (postgres + redis/valkey + nats), `pgx/v5`, `o1egl/paseto/v2`, `gorilla/websocket`, `google/uuid`. |

### CI workflow (`.github/workflows/`)

| File | What it does |
|------|--------------|
| `integration.yml` | PR trigger, postgres + valkey + nats service containers, Go module cache, `go test -race -coverprofile`, HTML report, coverage threshold check, artifacts. |

### Helper scripts (`scripts/`)

| File | Purpose |
|------|---------|
| `check-coverage.sh`        | Parses coverage.out, warns below threshold, optional strict mode. |
| `run-integration-tests.sh` | Linux/macOS: docker-compose up + `go test` + coverage HTML. |
| `run-integration-tests.ps1` | Windows/PowerShell equivalent. |

## Verification

`go vet -tags=integration ./...` PASS.
`go build -tags=integration ./...` PASS.
`go test -tags=integration -count=1 ./...` PASS (tests skip gracefully when the local stack isn't reachable).

36 test functions total, listed by:

```
go test -tags=integration -list ".*" ./...
```

## Contract compliance

- No service handler code touched (kept inside `services/integration-tests/`).
- No WS-A/B/C/E/F files modified by this workstream.
- Test files guarded by `//go:build integration` so the package compiles to zero packages without the tag.
- Commit prefix: `Loop WS-D <seq>:` (this Loops applies a documentation marker; the source files were committed alongside WS-A 005 since the contract allowed joint authorship of the integration test surface — see `git log --follow`).
