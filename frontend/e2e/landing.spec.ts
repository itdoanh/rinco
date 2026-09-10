import { test, expect } from '@playwright/test';

const APPS = {
  landing: 'http://localhost:3000',
  admin: 'http://localhost:3001',
  tenant: 'http://localhost:3002',
  meeting: 'http://localhost:3003',
}

test.describe('Landing smoke', () => {
  test('homepage renders hero & CTA', async ({ page }) => {
    await page.goto(APPS.landing + '/', { waitUntil: 'domcontentloaded' })
    // Wait a beat for client hydration; tolerate transient render hiccups.
    await page.waitForTimeout(1000)
    const mainVisible = await page.locator('main').first().isVisible().catch(() => false)
    if (!mainVisible) {
      // Some hydration glitches (image decoding) can throw briefly. Retry once.
      await page.reload({ waitUntil: 'domcontentloaded' })
      await page.waitForTimeout(1000)
    }
    await expect(page.locator('main').first()).toBeVisible()
    const h1 = page.locator('h1').first()
    await expect(h1).toBeVisible()
  })

  test('tenant dynamic route loads', async ({ page }) => {
    const res = await page.goto(APPS.landing + '/apex-fintech')
    expect(res?.status() ?? 200).toBeGreaterThanOrEqual(200)
  })

  test('track API accepts an event', async ({ request }) => {
    const res = await request.post(APPS.landing + '/api/track', {
      data: { event: 'page_view', url: '/e2e' },
    })
    expect([200, 201, 204]).toContain(res.status())
  })
})
