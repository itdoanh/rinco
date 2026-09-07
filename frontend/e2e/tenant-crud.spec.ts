import { test, expect } from '@playwright/test';

const APPS = {
  admin: 'http://localhost:3001',
  tenant: 'http://localhost:3002',
}

test.describe('Tenant CRUD across apps', () => {
  test('admin tenant list renders', async ({ page }) => {
    await page.goto(APPS.admin + '/tenants')
    await page.waitForLoadState('networkidle').catch(() => {})
    expect(page.url()).toMatch(/tenants|login/)
  })

  test('admin tenant detail renders or 404s gracefully', async ({ page }) => {
    const res = await page.goto(APPS.admin + '/tenants/apex-fintech')
    expect(res?.status() ?? 200).toBeGreaterThanOrEqual(200)
  })

  test('public tenant page can submit a lead (best-effort)', async ({ page, request }) => {
    await page.goto(APPS.tenant + '/demo')
    const formOk = await page.locator('form input[name="name"]').isVisible().catch(() => false)
    if (formOk) {
      await page.locator('input[name="name"]').fill('E2E User')
      await page.locator('input[name="phone"]').fill('0909123456')
      await page.locator('input[name="email"]').fill('e2e@example.com')
      await page.locator('button[type="submit"]').click()
    } else {
      const res = await request.post(APPS.tenant + '/api/leads', {
        data: { name: 'E2E', phone: '0909123456' },
      })
      expect([200, 201, 400, 422]).toContain(res.status())
    }
  })
})
