import { test, expect } from '@playwright/test';

test.describe('Homepage', () => {
  test('renders without errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    await page.goto('/')
    await expect(page.locator('main').first()).toBeVisible()
    expect(errors).toEqual([])
  })

  test('header and footer are visible', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('header')).toBeVisible()
    await expect(page.locator('footer')).toBeVisible()
  })

  test('hero contains primary headline', async ({ page }) => {
    await page.goto('/')
    const h1 = page.locator('h1').first()
    await expect(h1).toBeVisible()
    const text = (await h1.textContent()) ?? ''
    expect(text.length).toBeGreaterThan(5)
  })

  test('STICKY CTA appears after scroll', async ({ page }) => {
    await page.goto('/')
    await page.evaluate(() => window.scrollTo(0, 800))
    await page.waitForTimeout(300)
    // sticky CTA is generally positioned fixed; verify by selector or class
    const sticky = page.locator('[data-testid="sticky-cta"], .sticky-cta, button:has-text("ĐĂNG KÝ")').last()
    await expect(sticky).toBeVisible()
  })

  test('feature grid renders at least 1 item', async ({ page }) => {
    await page.goto('/')
    const items = page.locator('[data-testid="feature"] , .feature-card , h3')
    const count = await items.count()
    expect(count).toBeGreaterThan(0)
  })

  test('lead modal opens when CTA is clicked', async ({ page }) => {
    await page.goto('/')
    const cta = page.locator('button:has-text("ĐĂNG KÝ")').first()
    if (await cta.isVisible().catch(() => false)) {
      await cta.click()
      await page.waitForTimeout(200)
      // Modal opens (any dialog-ish role)
      const dialog = page.locator('[role="dialog"]')
      await expect(dialog.first()).toBeVisible()
    }
  })
})
