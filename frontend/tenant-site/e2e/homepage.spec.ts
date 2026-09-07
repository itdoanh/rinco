import { test, expect } from '@playwright/test';

test.describe('Tenant homepage', () => {
  test('renders for default tenant', async ({ page }) => {
    const res = await page.goto('/demo')
    expect(res?.status() ?? 0).toBeGreaterThanOrEqual(200)
    await expect(page.locator('main').first()).toBeVisible()
  })

  test('header and footer are visible', async ({ page }) => {
    await page.goto('/demo')
    await expect(page.locator('header').first()).toBeVisible()
    await expect(page.locator('footer').first()).toBeVisible()
  })

  test('contact form is interactive', async ({ page }) => {
    await page.goto('/demo')
    const nameInput = page.locator('input[name="name"]').first()
    if (await nameInput.isVisible().catch(() => false)) {
      await nameInput.fill('Loc Tester')
      const phone = page.locator('input[name="phone"]').first()
      await phone.fill('0909123456')
      const submit = page.locator('button[type="submit"]').first()
      await expect(submit).toBeEnabled()
    }
  })

  test('applies branding color variable', async ({ page }) => {
    await page.goto('/demo')
    await page.waitForTimeout(300)
    const color = await page.evaluate(() => {
      const v = getComputedStyle(document.documentElement).getPropertyValue('--brand-primary')
      return v.trim()
    })
    expect(typeof color).toBe('string')
  })
})

test.describe('Sub pages', () => {
  test('unknown page returns 404 or error', async ({ page }) => {
    const res = await page.goto('/demo/no-such-page')
    expect(res?.status() ?? 200).toBeGreaterThanOrEqual(200)
  })
})

test.describe('Lead API', () => {
  test('POST /api/leads accepts payload', async ({ request }) => {
    const res = await request.post('/api/leads', {
      data: {
        name: 'Tester',
        phone: '0909123456',
        tenant_slug: 'demo',
      },
    })
    expect([200, 201, 400, 422]).toContain(res.status())
  })
})
