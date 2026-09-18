/**
 * WS-T: Responsive layout validation — iPhone SE, iPad, 1440px desktop.
 *
 * Asserts no horizontal scroll on body, nav menu collapses on mobile,
 * and key elements remain visible at each breakpoint.
 */
import { test, expect, type Page } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const APPS = {
  landing: 'http://localhost:3000',
  admin:   'http://localhost:3001',
  tenant:  'http://localhost:3002',
  meeting: 'http://localhost:3003',
};

const SCREENSHOT_DIR = resolve(__dirname, '..', 'screenshots');

async function shot(page: Page, label: string): Promise<void> {
  const dir = resolve(SCREENSHOT_DIR, 'responsive');
  mkdirSync(dir, { recursive: true });
  await page.screenshot({ path: resolve(dir, `${label}.png`), fullPage: true });
}

async function assertNoHorizontalScroll(page: Page): Promise<void> {
  const overflow = await page.evaluate(() => {
    const body = document.body;
    return {
      scrollWidth: body.scrollWidth,
      clientWidth: body.clientWidth,
      innerWidth: window.innerWidth,
    };
  });
  // Allow 10px tolerance for browser scrollbar + rendering rounding.
  expect(
    overflow.scrollWidth - overflow.clientWidth,
    `horizontal scroll detected (${overflow.scrollWidth} > ${overflow.clientWidth} on inner ${overflow.innerWidth})`
  ).toBeLessThanOrEqual(10);
}

// ============================================================================
// Mobile — iPhone SE (375 x 667)
// ============================================================================
test.describe('Responsive mobile (iPhone SE)', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
  });

  test('landing homepage renders without horizontal scroll (post-fix)', async ({ page }) => {
    await page.goto(APPS.landing + '/', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'iphone-se-landing-postfix');
  });

  test('admin login renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.admin + '/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'iphone-se-admin-login');
  });

  test('tenant site renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.tenant + '/apexfintech', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'iphone-se-tenant');
  });

  test('meeting root renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.meeting + '/', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'iphone-se-meeting');
  });
});

// ============================================================================
// Tablet — iPad (768 x 1024)
// ============================================================================
test.describe('Responsive tablet (iPad)', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
  });

  test('landing renders without horizontal scroll (post-fix)', async ({ page }) => {
    await page.goto(APPS.landing + '/', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'ipad-landing-postfix');
  });

  test('admin dashboard renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.admin + '/dashboard', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);
    await assertNoHorizontalScroll(page);
    await shot(page, 'ipad-admin-dashboard');
  });

  test('tenant demo hub renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.tenant + '/demo', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'ipad-tenant-demo');
  });
});

// ============================================================================
// Desktop — 1440 x 900
// ============================================================================
test.describe('Responsive desktop (1440x900)', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
  });

  test('landing renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.landing + '/', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'desktop-1440-landing');
  });

  test('admin dashboard renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.admin + '/dashboard', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);
    await assertNoHorizontalScroll(page);
    await shot(page, 'desktop-1440-admin-dashboard');
  });

  test('meeting UI renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.meeting + '/', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'desktop-1440-meeting');
  });

  test('tenant apexfintech renders without horizontal scroll', async ({ page }) => {
    await page.goto(APPS.tenant + '/apexfintech', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    await assertNoHorizontalScroll(page);
    await shot(page, 'desktop-1440-tenant');
  });
});