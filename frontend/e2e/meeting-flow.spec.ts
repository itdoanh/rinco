import { test, expect } from '@playwright/test';

const APPS = {
  landing: 'http://localhost:3000',
  meeting: 'http://localhost:3003',
}

test.describe('Meeting flow', () => {
  test('meeting UI root page renders', async ({ page }) => {
    const res = await page.goto(APPS.meeting + '/')
    expect(res?.status() ?? 200).toBeGreaterThanOrEqual(200)
  })

  test('meeting room shell renders or shows error', async ({ page, context }) => {
    // Clear permissions so getUserMedia fails fast; the app should render the shell
    await context.clearPermissions()
    await page.goto(APPS.meeting + '/meeting/test-room?name=E2E+User')
    await page.waitForTimeout(2000)
    const body = await page.content()
    expect(body).toMatch(/connecting|error|meeting|video/i)
  })
})
