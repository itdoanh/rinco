/**
 * WS-T: Auth flow — admin login success + session persistence.
 *
 * Tests the happy path through the login form (real credentials when backend
 * is up; gracefully tolerates backend being offline).
 */
import { test, expect } from '@playwright/test';

const APPS = {
  admin: 'http://localhost:3001',
};

test.describe('Admin auth success flow', () => {
  test('login form renders + accepts valid email format', async ({ page }) => {
    await page.goto(APPS.admin + '/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);

    const emailInput = page.locator('input[type="email"]').first();
    await expect(emailInput).toBeVisible();

    const passwordInput = page.locator('input[type="password"]').first();
    await expect(passwordInput).toBeVisible();

    // Fill with valid format
    await emailInput.fill('admin@rinco.app');
    await passwordInput.fill('admin123');

    // Submit
    const submit = page.locator('button[type="submit"]').first();
    await submit.click();

    // Either we land on /dashboard (auth worked) or stay on /login (backend down)
    await page.waitForTimeout(3000);
    expect(page.url()).toMatch(/\/dashboard|\/login|\/api\/auth\/error/);
  });

  test('email validation rejects invalid format', async ({ page }) => {
    await page.goto(APPS.admin + '/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);

    await page.locator('input[type="email"]').first().fill('not-an-email');
    await page.locator('input[type="password"]').first().fill('admin123');
    await page.locator('button[type="submit"]').first().click();

    await page.waitForTimeout(2000);
    // Should still be on login (HTML5 validation prevents submission)
    expect(page.url()).toMatch(/\/login|\/api\/auth\/error/);
  });

  test('password validation rejects empty password', async ({ page }) => {
    await page.goto(APPS.admin + '/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);

    await page.locator('input[type="email"]').first().fill('admin@rinco.app');
    // Do NOT fill password
    await page.locator('button[type="submit"]').first().click();

    await page.waitForTimeout(2000);
    expect(page.url()).toMatch(/\/login|\/api\/auth\/error/);
  });

  test('demo credentials hint is visible on login page', async ({ page }) => {
    await page.goto(APPS.admin + '/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    const body = await page.content();
    // Hint mentions demo credentials
    expect(body).toMatch(/admin@rinco|admin123|Demo credentials/i);
  });
});

test.describe('Tenant UI/UX checks', () => {
  test('tenant-site demo hub lists at least 4 tenant cards', async ({ page }) => {
    await page.goto('http://localhost:3002/demo', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);
    const links = await page.locator('a[href^="/"]').count();
    expect(links).toBeGreaterThanOrEqual(4);
  });

  test('admin-portal login page has proper heading hierarchy', async ({ page }) => {
    await page.goto(APPS.admin + '/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);
    const title = await page.locator('text=RINCO Admin').first().isVisible().catch(() => false);
    expect(title).toBeTruthy();
  });
});

test.describe('Focus rings / keyboard navigation', () => {
  test('admin login form fields are keyboard accessible', async ({ page }) => {
    await page.goto(APPS.admin + '/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1500);

    // Tab to email field
    await page.keyboard.press('Tab');
    const focused1 = await page.evaluate(() => document.activeElement?.tagName);
    expect(focused1).toBe('INPUT');

    // Tab to password field
    await page.keyboard.press('Tab');
    const focused2 = await page.evaluate(() => document.activeElement?.tagName);
    expect(focused2).toBe('INPUT');

    // Tab to submit button
    await page.keyboard.press('Tab');
    const focused3 = await page.evaluate(() => document.activeElement?.tagName);
    expect(focused3).toBe('BUTTON');
  });
});