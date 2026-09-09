/**
 * Runtime unit tests for admin-portal/store/index.ts (main admin store).
 *
 * Tests the AdminStore tenant CRUD, filters, and Analytics/UI stores.
 */
;(globalThis as any).localStorage = {
  getItem: () => null,
  setItem: () => {},
  removeItem: () => {},
  clear: () => {},
  key: () => null,
  get length() {
    return 0
  },
}

import { useAdminStore, useAnalyticsStore, useUIStore } from "../store/index"

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

// Admin store
function testAdminSeed() {
  useAdminStore.setState({ tenants: [] })
  assertEqual(useAdminStore.getState().tenants.length, 0, "empty seed")
  assertEqual(useAdminStore.getState().selectedTenant, null, "no selection")
}

function testAdminSetTenants() {
  useAdminStore.setState({ tenants: [] })
  useAdminStore.getState().setTenants([
    { id: "t1", slug: "acme", name: "ACME", status: "active", created_at: "", mrr: 0, users_count: 0, leads_count: 0 },
  ])
  assertEqual(useAdminStore.getState().tenants.length, 1, "set")
}

function testAdminSetSelected() {
  useAdminStore.setState({ tenants: [], selectedTenant: null })
  const t = { id: "x", slug: "x", name: "X", status: "active" as const, created_at: "", mrr: 0, users_count: 0, leads_count: 0 }
  useAdminStore.getState().setSelectedTenant(t)
  assert(useAdminStore.getState().selectedTenant !== null, "selected")
  useAdminStore.getState().setSelectedTenant(null)
  assertEqual(useAdminStore.getState().selectedTenant, null, "cleared")
}

function testAdminSetFilters() {
  useAdminStore.setState({
    filters: { search: "", status: "", sortBy: "created_at", sortOrder: "desc" as const },
  })
  useAdminStore.getState().setFilters({ search: "test" })
  assertEqual(useAdminStore.getState().filters.search, "test", "search filter")
  assertEqual(useAdminStore.getState().filters.sortBy, "created_at", "sortBy preserved")
}

function testAdminAddTenant() {
  useAdminStore.setState({ tenants: [] })
  const t = { id: "1", slug: "a", name: "A", status: "active" as const, created_at: "", mrr: 0, users_count: 0, leads_count: 0 }
  useAdminStore.getState().addTenant(t)
  assertEqual(useAdminStore.getState().tenants.length, 1, "added")
}

function testAdminUpdateTenant() {
  useAdminStore.setState({ tenants: [] })
  const t = { id: "1", slug: "a", name: "A", status: "active" as const, created_at: "", mrr: 0, users_count: 0, leads_count: 0 }
  useAdminStore.getState().addTenant(t)
  useAdminStore.getState().updateTenant("1", { name: "Updated" })
  assertEqual(useAdminStore.getState().tenants[0].name, "Updated", "updated")
}

function testAdminUpdateTenantMissing() {
  useAdminStore.setState({ tenants: [] })
  const t = { id: "1", slug: "a", name: "A", status: "active" as const, created_at: "", mrr: 0, users_count: 0, leads_count: 0 }
  useAdminStore.getState().addTenant(t)
  useAdminStore.getState().updateTenant("missing", { name: "X" })
  assertEqual(useAdminStore.getState().tenants[0].name, "A", "missing unchanged")
}

function testAdminRemoveTenant() {
  useAdminStore.setState({ tenants: [] })
  const t = { id: "1", slug: "a", name: "A", status: "active" as const, created_at: "", mrr: 0, users_count: 0, leads_count: 0 }
  useAdminStore.getState().addTenant(t)
  useAdminStore.getState().removeTenant("1")
  assertEqual(useAdminStore.getState().tenants.length, 0, "removed")
}

// Analytics store
function testAnalyticsSeed() {
  useAnalyticsStore.setState({ data: null, isLoading: false })
  assertEqual(useAnalyticsStore.getState().data, null, "null data")
  assertEqual(useAnalyticsStore.getState().isLoading, false, "not loading")
}

function testAnalyticsSetData() {
  useAnalyticsStore.setState({ data: null, isLoading: false })
  const d = {
    totalTenants: 10,
    totalUsers: 100,
    totalLeads: 1000,
    mrr: 5000,
    leadsToday: 50,
    conversionRate: 0.05,
    pageviewsToday: 5000,
    apiCallsMin: 100,
  }
  useAnalyticsStore.getState().setData(d)
  assert(useAnalyticsStore.getState().data !== null, "data set")
  assertEqual(useAnalyticsStore.getState().data!.totalTenants, 10, "totalTenants")
}

function testAnalyticsSetLoading() {
  useAnalyticsStore.setState({ data: null, isLoading: false })
  useAnalyticsStore.getState().setLoading(true)
  assertEqual(useAnalyticsStore.getState().isLoading, true, "loading")
  useAnalyticsStore.getState().setLoading(false)
  assertEqual(useAnalyticsStore.getState().isLoading, false, "not loading")
}

// UI store
function testUIPortalSeed() {
  useUIStore.setState({ sidebarOpen: true })
  assertEqual(useUIStore.getState().sidebarOpen, true, "default open")
}

function testUISetSidebar() {
  useUIStore.setState({ sidebarOpen: true })
  useUIStore.getState().setSidebarOpen(false)
  assertEqual(useUIStore.getState().sidebarOpen, false, "closed")
}

function testUIToggle() {
  useUIStore.setState({ sidebarOpen: false })
  useUIStore.getState().toggleSidebar()
  assertEqual(useUIStore.getState().sidebarOpen, true, "toggle to open")
  useUIStore.getState().toggleSidebar()
  assertEqual(useUIStore.getState().sidebarOpen, false, "toggle to closed")
}

testAdminSeed()
testAdminSetTenants()
testAdminSetSelected()
testAdminSetFilters()
testAdminAddTenant()
testAdminUpdateTenant()
testAdminUpdateTenantMissing()
testAdminRemoveTenant()
testAnalyticsSeed()
testAnalyticsSetData()
testAnalyticsSetLoading()
testUIPortalSeed()
testUISetSidebar()
testUIToggle()

console.log("\nResults: " + passed + " passed, " + failed + " failed")
if (failed > 0) process.exit(1)
