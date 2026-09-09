/**
 * Runtime unit tests for admin-portal/store/admin-stores.ts.
 *
 * Zustand's `create` produces a hook so we exercise its `.getState()` /
 * `.setState()` API directly without React or jsdom.  The persist
 * middleware would normally try to read from `localStorage`, so we mock
 * it on globalThis.
 */

// In-memory localStorage shim
const lsStore: Record<string, string> = {}
const ls = {
  getItem: (k: string) => (k in lsStore ? lsStore[k] : null),
  setItem: (k: string, v: string) => {
    lsStore[k] = v
  },
  removeItem: (k: string) => {
    delete lsStore[k]
  },
  clear: () => {
    for (const k of Object.keys(lsStore)) delete lsStore[k]
  },
  key: (i: number) => Object.keys(lsStore)[i] ?? null,
  get length() {
    return Object.keys(lsStore).length
  },
}
;(globalThis as any).localStorage = ls

import {
  useFeatureFlagsStore,
  useNotificationTemplatesStore,
  useQuorumStore,
} from "../store/admin-stores"

let passed = 0
let failed = 0

function assert(cond: unknown, label: string): void {
  if (cond) passed++
  else {
    failed++
    console.error("FAIL " + label)
  }
}

function assertEqual<T>(actual: T, expected: T, label: string): void {
  if (actual === expected) passed++
  else {
    failed++
    console.error("FAIL " + label + ": got " + JSON.stringify(actual) + " want " + JSON.stringify(expected))
  }
}

// =============================================================================
// Feature flags
// =============================================================================

function resetFlags() {
  useFeatureFlagsStore.getState().reset()
}

function testFlagsSeed() {
  resetFlags()
  const flags = useFeatureFlagsStore.getState().flags
  assert(flags.length > 0, "seed has flags")
  const enabled = flags.filter((f) => f.enabled).length
  assert(enabled > 0, "at least one enabled flag")
}

function testFlagsToggle() {
  resetFlags()
  const before = useFeatureFlagsStore.getState().flags.find((f) => f.key === "webrtc_av1_svc")!
  assert(!before.enabled, "starts disabled")
  useFeatureFlagsStore.getState().toggle("webrtc_av1_svc")
  const after = useFeatureFlagsStore.getState().flags.find((f) => f.key === "webrtc_av1_svc")!
  assert(after.enabled, "now enabled")
  assert(after.rolloutPercentage >= 1, "rollout set on enable")
  useFeatureFlagsStore.getState().toggle("webrtc_av1_svc")
  const back = useFeatureFlagsStore.getState().flags.find((f) => f.key === "webrtc_av1_svc")!
  assert(!back.enabled, "disabled again")
  assertEqual(back.rolloutPercentage, 0, "rollout=0 when disabled")
}

function testFlagsToggleMissingKey() {
  resetFlags()
  const beforeLen = useFeatureFlagsStore.getState().flags.length
  useFeatureFlagsStore.getState().toggle("nonexistent_key")
  assertEqual(useFeatureFlagsStore.getState().flags.length, beforeLen, "no change for missing")
}

function testFlagsUpdateRollout() {
  resetFlags()
  useFeatureFlagsStore.getState().updateRollout("facebook_capi_v2", 50)
  const f = useFeatureFlagsStore.getState().flags.find((x) => x.key === "facebook_capi_v2")!
  assertEqual(f.rolloutPercentage, 50, "rollout set to 50")
}

function testFlagsUpdateRolloutClamp() {
  resetFlags()
  useFeatureFlagsStore.getState().updateRollout("facebook_capi_v2", 200)
  const f = useFeatureFlagsStore.getState().flags.find((x) => x.key === "facebook_capi_v2")!
  assertEqual(f.rolloutPercentage, 100, "clamped to 100")

  useFeatureFlagsStore.getState().updateRollout("facebook_capi_v2", -10)
  const f2 = useFeatureFlagsStore.getState().flags.find((x) => x.key === "facebook_capi_v2")!
  assertEqual(f2.rolloutPercentage, 0, "clamped to 0")
}

function testFlagsAdd() {
  resetFlags()
  const before = useFeatureFlagsStore.getState().flags.length
  useFeatureFlagsStore.getState().add({
    key: "test_new_flag",
    description: "Test",
    enabled: false,
    rolloutPercentage: 0,
    category: "experimental",
  })
  const after = useFeatureFlagsStore.getState().flags.length
  assertEqual(after, before + 1, "flag added")
  const f = useFeatureFlagsStore.getState().flags.find((x) => x.key === "test_new_flag")!
  assert(f.lastModified.length > 0, "lastModified set")
}

function testFlagsRemove() {
  resetFlags()
  useFeatureFlagsStore.getState().remove("legacy_landing_v1")
  const f = useFeatureFlagsStore.getState().flags.find((x) => x.key === "legacy_landing_v1")
  assert(!f, "removed")
}

function testFlagsIsEnabled() {
  resetFlags()
  assert(useFeatureFlagsStore.getState().isEnabled("dynamic_model_engine"), "enabled flag")
  assert(!useFeatureFlagsStore.getState().isEnabled("legacy_landing_v1"), "disabled flag")
  assert(!useFeatureFlagsStore.getState().isEnabled("does_not_exist"), "missing returns false")
}

// =============================================================================
// Notification templates
// =============================================================================

function testTemplatesSeed() {
  useNotificationTemplatesStore.getState().reset()
  const t = useNotificationTemplatesStore.getState().templates
  assert(t.length > 0, "seed templates")
}

function testTemplatesAdd() {
  useNotificationTemplatesStore.getState().reset()
  const before = useNotificationTemplatesStore.getState().templates.length
  useNotificationTemplatesStore.getState().add({
    code: "test.added",
    category: "INFO",
    channel: "IN_APP",
    subject: "Test Subject",
    bodyTemplate: "Hello {{name}}",
    variables: ["name"],
  })
  const after = useNotificationTemplatesStore.getState().templates.length
  assertEqual(after, before + 1, "added")
  const added = useNotificationTemplatesStore.getState().templates.find((t) => t.code === "test.added")
  assert(added !== undefined, "findable")
  assert(added!.id.startsWith("tpl-"), "auto-id with prefix")
}

function testTemplatesUpdate() {
  useNotificationTemplatesStore.getState().reset()
  const first = useNotificationTemplatesStore.getState().templates[0]
  useNotificationTemplatesStore.getState().update(first.id, { subject: "Updated" })
  const updated = useNotificationTemplatesStore.getState().templates.find((t) => t.id === first.id)!
  assertEqual(updated.subject, "Updated", "subject updated")
}

function testTemplatesRemove() {
  useNotificationTemplatesStore.getState().reset()
  const first = useNotificationTemplatesStore.getState().templates[0]
  useNotificationTemplatesStore.getState().remove(first.id)
  const after = useNotificationTemplatesStore.getState().templates.find((t) => t.id === first.id)
  assert(!after, "removed")
}

function testTemplatesFindByCode() {
  useNotificationTemplatesStore.getState().reset()
  const found = useNotificationTemplatesStore.getState().findByCode("welcome.new_tenant")
  assert(found !== undefined, "found")
  assertEqual(found!.code, "welcome.new_tenant", "code matches")
  const missing = useNotificationTemplatesStore.getState().findByCode("does.not.exist")
  assert(!missing, "missing returns undefined")
}

// =============================================================================
// Quorum
// =============================================================================

function testQuorumSeed() {
  useQuorumStore.getState().reset()
  const q = useQuorumStore.getState().requests
  assert(q.length > 0, "seed quorums")
}

function testQuorumCreate() {
  useQuorumStore.getState().reset()
  const before = useQuorumStore.getState().requests.length
  const created = useQuorumStore.getState().create({
    action: "test.action",
    reason: "test reason",
    initiator: "alice@example.com",
    requiredSigs: 2,
    expiresAt: new Date(Date.now() + 3600_000).toISOString(),
  })
  assertEqual(useQuorumStore.getState().requests.length, before + 1, "added")
  assertEqual(created.status, "PENDING", "start pending")
  assertEqual(created.collectedSigs.length, 1, "initiator auto-signs")
  assert(created.createdAt.length > 0, "createdAt set")
  assert(created.id.startsWith("qr-"), "id prefix")
}

function testQuorumCreateWithExplicitSigs() {
  useQuorumStore.getState().reset()
  const created = useQuorumStore.getState().create({
    action: "test",
    reason: "test",
    initiator: "alice",
    requiredSigs: 3,
    collectedSigs: ["alice", "bob"],
    expiresAt: new Date(Date.now() + 3600_000).toISOString(),
  })
  assertEqual(created.collectedSigs.length, 2, "explicit sigs used")
}

function testQuorumSign() {
  useQuorumStore.getState().reset()
  const initial = useQuorumStore.getState().requests[0]
  if (initial.status !== "PENDING") {
    // reset always gives PENDING for the second seed
    const pending = useQuorumStore.getState().requests.find((q) => q.status === "PENDING")!
    useQuorumStore.getState().sign(pending.id, "new_signer")
    const updated = useQuorumStore.getState().requests.find((q) => q.id === pending.id)!
    assert(updated.collectedSigs.includes("new_signer"), "new signer added")
    return
  }
  useQuorumStore.getState().sign(initial.id, "new_signer")
  const updated = useQuorumStore.getState().requests.find((q) => q.id === initial.id)!
  assert(updated.collectedSigs.includes("new_signer"), "new signer added")
}

function testQuorumSignDuplicate() {
  useQuorumStore.getState().reset()
  const pending = useQuorumStore.getState().requests.find((q) => q.status === "PENDING")!
  const before = useQuorumStore.getState().requests.find((q) => q.id === pending.id)!.collectedSigs.length
  const existing = pending.collectedSigs[0]
  useQuorumStore.getState().sign(pending.id, existing)
  const after = useQuorumStore.getState().requests.find((q) => q.id === pending.id)!.collectedSigs.length
  assertEqual(after, before, "duplicate sign ignored")
}

function testQuorumSignApproved() {
  useQuorumStore.getState().reset()
  const pending = useQuorumStore.getState().requests.find((q) => q.status === "PENDING")!
  const needed = pending.requiredSigs - pending.collectedSigs.length
  for (let i = 0; i < needed; i++) {
    useQuorumStore.getState().sign(pending.id, `signer${i}@x.com`)
  }
  const updated = useQuorumStore.getState().requests.find((q) => q.id === pending.id)!
  assertEqual(updated.status, "APPROVED", "approved when threshold reached")
}

function testQuorumReject() {
  useQuorumStore.getState().reset()
  const pending = useQuorumStore.getState().requests.find((q) => q.status === "PENDING")!
  useQuorumStore.getState().reject(pending.id, "rejector@x.com")
  const updated = useQuorumStore.getState().requests.find((q) => q.id === pending.id)!
  assertEqual(updated.status, "REJECTED", "rejected")
}

function testQuorumTickExpiry() {
  useQuorumStore.getState().reset()
  useQuorumStore.getState().tickExpiry()
  // The first seed is APPROVED with past expiry, the second/third are PENDING
  // with future expiry. After tick, no PENDING should be flipped to EXPIRED
  // because none of the seeds have past-while-PENDING. Verify nothing
  // regressed:
  const pendings = useQuorumStore.getState().requests.filter((q) => q.status === "PENDING")
  assert(pendings.length > 0, "still has pendings")
}

function testQuorumSignExpired() {
  useQuorumStore.getState().reset()
  // Force-create an expired pending request
  const expired = useQuorumStore.getState().create({
    action: "test.expired",
    reason: "test",
    initiator: "alice",
    requiredSigs: 2,
    expiresAt: new Date(Date.now() - 1000).toISOString(),
  })
  useQuorumStore.getState().sign(expired.id, "bob")
  const updated = useQuorumStore.getState().requests.find((q) => q.id === expired.id)!
  assertEqual(updated.status, "EXPIRED", "expired on sign")
}

function testQuorumRejectNonPending() {
  useQuorumStore.getState().reset()
  const approved = useQuorumStore.getState().requests.find((q) => q.status === "APPROVED")!
  useQuorumStore.getState().reject(approved.id, "x")
  const after = useQuorumStore.getState().requests.find((q) => q.id === approved.id)!
  assertEqual(after.status, "APPROVED", "reject on non-pending is no-op")
}

// Run all tests
testFlagsSeed()
testFlagsToggle()
testFlagsToggleMissingKey()
testFlagsUpdateRollout()
testFlagsUpdateRolloutClamp()
testFlagsAdd()
testFlagsRemove()
testFlagsIsEnabled()
testTemplatesSeed()
testTemplatesAdd()
testTemplatesUpdate()
testTemplatesRemove()
testTemplatesFindByCode()
testQuorumSeed()
testQuorumCreate()
testQuorumCreateWithExplicitSigs()
testQuorumSign()
testQuorumSignDuplicate()
testQuorumSignApproved()
testQuorumReject()
testQuorumTickExpiry()
testQuorumSignExpired()
testQuorumRejectNonPending()

console.log("\nResults: " + passed + " passed, " + failed + " failed")
if (failed > 0) process.exit(1)
