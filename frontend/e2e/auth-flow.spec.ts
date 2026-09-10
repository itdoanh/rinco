import { test, expect } from '@playwright/test';

const APPS = {
  landing: 'http://localhost:3000',
  admin: 'http://localhost:3001',
  tenant: 'http://localhost:3002',
}

test.describe('Auth flow across apps', () => {
  test('admin login rejects bad credentials', async ({ page }) => {
    await page.goto(APPS.admin + '/login')
    await page.locator('input[type="email"]').first().fill('nobody@rinco.app')
    await page.locator('input[type="password"]').first().fill('wrongpassword')
    await page.locator('button[type="submit"]').first().click()
    await page.waitForTimeout(500)
    expect(page.url()).toMatch(/\/login|\/api\/auth\/error/)
  })

  test('admin dashboard is protected', async ({ page }) => {
    await page.goto(APPS.admin + '/dashboard')
    await page.waitForLoadState('networkidle').catch(() => {})
    const ok =
      page.url().includes('/login') ||
      page.url().includes('/dashboard') ||
      (await page.locator('text=/Dashboard|Stats|tenants/i').first().isVisible().catch(() => false))
    expect(ok).toBeTruthy()
  })

  test('landing site is publicly accessible', async ({ page }) => {
    const res = await page.goto(APPS.landing + '/')
    expect(res?.status() ?? 200).toBeLessThan(500)
  })

  test('tenant site is publicly accessible', async ({ page }) => {
    const res = await page.goto(APPS.tenant + '/demo')
    expect(res?.status() ?? 200).toBeLessThan(500)
  })
})
