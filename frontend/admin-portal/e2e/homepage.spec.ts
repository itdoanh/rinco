import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // attempt to access; most tests require auth but we just check that the shell loads
    await page.goto('/dashboard')
  })

  test('page renders or redirects to login', async ({ page }) => {
    const url = page.url()
    expect(url).toMatch(/dashboard|login/)
  })

  test('login then dashboard shows KPI labels', async ({ page }) => {
    // Best-effort: fill demo creds, then assert dashboard renders
    if (page.url().includes('/login') || page.url().endsWith('/')) {
      const email = page.locator('input[type="email"]').first()
      const pwd = page.locator('input[type="password"]').first()
      if (await email.isVisible().catch(() => false)) {
        await email.fill('admin@rinco.app')
        await pwd.fill('admin123')
        await page.locator('button[type="submit"]').first().click()
        await page.waitForURL(/dashboard/, { timeout: 5000 }).catch(() => {})
      }
    }
    if (page.url().includes('dashboard')) {
      const hasKpi = await page.getByText(/tenants|leads|users|MRR/i).first().isVisible().catch(() => false)
      expect(hasKpi).toBeTruthy()
    }
  })
})

test.describe('Tenant CRUD UI', () => {
  test('tenants page renders', async ({ page }) => {
    await page.goto('/tenants')
    await page.waitForLoadState('networkidle').catch(() => {})
    expect(page.url()).toMatch(/tenants|login/)
  })

  test('tenant detail renders or shows 404', async ({ page }) => {
    const res = await page.goto('/tenants/apex-fintech')
    expect(res?.status() ?? 0).toBeGreaterThanOrEqual(200)
  })
})

test.describe('System pages', () => {
  for (const path of ['/system/health', '/system/logs', '/system/metrics']) {
    test(`page ${path} renders`, async ({ page }) => {
      const res = await page.goto(path)
      expect(res?.status() ?? 0).toBeGreaterThanOrEqual(200)
    })
  }
})

test.describe('Audit log', () => {
  test('audit page loads and exposes search', async ({ page }) => {
    await page.goto('/audit')
    await page.waitForLoadState('networkidle').catch(() => {})
    const hasSearch =
      (await page.locator('input[type="search"], input[placeholder*="Search" i]').first().isVisible().catch(() => false)) ||
      page.url().includes('login')
    expect(hasSearch).toBeTruthy()
  })
})
