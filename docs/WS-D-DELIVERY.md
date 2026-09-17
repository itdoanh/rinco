# WS-D — Delivery summary

WS-D was completed in a single Loop because the work overlapped with
WS-A's integration-test deliverable. The integration test surface
(`services/integration-tests/**` + `.github/workflows/integration.yml`
+ `scripts/check-coverage.sh` + `scripts/run-integration-tests.{sh,ps1}`)
was committed as part of `Loop WS-A 005` on `d4ed531`.

This commit adds the WS-D attribution and verifies the WS-D verification
contract.

## Verification status

| Check | Result |
|-------|--------|
| `go vet -tags=integration ./services/integration-tests/...` | PASS |
| `go build -tags=integration ./services/integration-tests/...` | PASS |
| `go test -tags=integration -count=1 ./services/integration-tests/...` | PASS (36 tests, skip on no-stack) |
| Files scope: `services/integration-tests/**` + `.github/workflows/integration.yml` + scripts only | ✓ |
| Service handler code touched | ✗ (none — handlers in `services/<name>/internal/handler/` untouched) |
| WS-A/B/C/E/F files modified | ✗ |

## Files in this workstream

| Path | LoC | Purpose |
|------|-----|---------|
| `services/integration-tests/setup_test.go`        | ~430 | testcontainers bootstrap + TestEnv cleanup |
| `services/integration-tests/helpers_test.go`      | ~400 | TestContext + MakeJWT + HTTPClient + assertions |
| `services/integration-tests/auth_flow_test.go`    | ~150 | register → login → refresh → logout |
| `services/integration-tests/leads_flow_test.go`   | ~190 | landing → CRM lead creation; HMAC; quorum; CAPI failures |
| `services/integration-tests/tenancy_test.go`      | ~190 | 2-tenant isolation matrix |
| `services/integration-tests/rbac_test.go`         | ~165 | 5 roles × CRUD/Admin endpoint matrix |
| `services/integration-tests/tree_test.go`         | ~165 | LTREE hierarchy + move + cycle |
| `services/integration-tests/workflow_test.go`     | ~190 | rule create + trigger + action; disabled; parallel |
| `services/integration-tests/notifications_test.go` | ~165 | create + queue + deliver; batch; invalid channel |
| `services/integration-tests/audit_test.go`        | ~155 | mutate → audit; read-only no-op; cross-tenant |
| `services/integration-tests/chat_flow_test.go`    | ~155 | WS + presence; unauthorized rejected |
| `services/integration-tests/scoring_flow_test.go` | ~165 | XGBoost single + batch + cap + health |
| `.github/workflows/integration.yml`               | ~150 | CI workflow |
| `scripts/check-coverage.sh`                       |  ~39  | coverage threshold check |
| `scripts/run-integration-tests.sh`                |  ~49  | Linux/macOS test runner |
| `scripts/run-integration-tests.ps1`               |  ~50  | Windows/PowerShell test runner |
| `docs/WS-D-1.md`                                  | ~80   | this workstream summary |

## Test count

`go test -list ".*"` returns 36 entries:

```
TestAuditLoggedForMutatingOps
TestAuditReadEndpointsAreReadOnly
TestAuditCrossActorIsolated
TestRegisterLoginRefreshLogout
TestRejectWeakPassword
TestRejectWrongLoginCredentials
TestPASETOTokenRoundTrip
TestAccessTokenExpiry
TestChatConnectSendAndPresence
TestChatPresenceHeartbeat
TestChatUnauthorizedConnection
TestLeadEndToEndFlow
TestHMACTamperingRejected
TestQuorumDeletionRequires2Of3
TestMetaCAPIRecordsFailedDelivery
TestNotificationCreateAndDeliver
TestNotificationBatchCreate
TestNotificationInvalidChannel
TestRBACMatrix
TestRBACLeadsWriteRead
TestRBACCrossRoleAccess
TestLeadScoringHappyPath
TestLeadScoringBatch
TestLeadScoringRejectedWhenBatchTooLarge
TestLeadScoringHealth
TestTenantsIsolatedFromEachOther
TestCrossTenantAttemptRejected
TestTenantCreateAndResolve
TestDuplicateSlugFails
TestCRMTreeHierarchy
TestCRMTreeMoveSubtree
TestCRMTreeCycleRejected
TestCRMTreeNoSelfParent
TestWorkflowCreateTriggerExecute
TestWorkflowDisabledDoesNotFire
TestWorkflowParallelFire
```
