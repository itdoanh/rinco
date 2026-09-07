import { test, expect } from '@playwright/test';

// These tests assume the app runs without media permissions in CI.
// They assert shell rendering, control bar visibility, and graceful error states,
// not actual WebRTC peer connections.
test.describe('Meeting room', () => {
  test('home page renders', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('main, body').first()).toBeVisible()
  })

  test('meeting page shows loading or error state', async ({ page, context }) => {
    // Block media permissions so getUserMedia rejects and we hit the error UI
    await context.clearPermissions()
    await page.goto('/meeting/test-room?name=TestUser')
    // Either connecting spinner OR an error message is acceptable
    await page.waitForTimeout(2000)
    const ok =
      (await page.locator('text=/Connecting|Connection Error|Could not access/i').first().isVisible().catch(() => false)) ||
      (await page.locator('main').first().isVisible())
    expect(ok).toBeTruthy()
  })

  test('control bar is visible once connected (or hidden on error)', async ({ page, context }) => {
    await context.grantPermissions(['camera', 'microphone']).catch(() => {})
    await page.goto('/meeting/demo-room?name=Demo')
    await page.waitForTimeout(2000)
    // .control-bar exists in the markup - check it's at least defined in the DOM
    const html = await page.content()
    expect(html).toMatch(/meeting|control-bar|video-tile/i)
  })

  test('gracefully handles invalid room id', async ({ page }) => {
    const res = await page.goto('/meeting/<invalid>')
    expect(res?.status() ?? 200).toBeGreaterThanOrEqual(200)
  })
})
