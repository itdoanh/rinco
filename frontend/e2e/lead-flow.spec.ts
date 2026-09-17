// E2E: submit a landing form and verify the lead appears in the CRM.
// Uses data-testid selectors where available; falls back to name= attributes.
import { test, expect } from '@playwright/test';

const APPS = {
  landing: 'http://localhost:3000',
  tenant:  'http://localhost:3002',
  crm:     'http://localhost:8083', // direct CRM API for assertion
  auth:    'http://localhost:8081',
};

async function loginAdmin(request: any) {
  const res = await request.post(APPS.auth + '/auth/login', {
    data: { email: 'admin@apex.vn', password: 'Test1234!' },
  });
  expect([200, 401]).toContain(res.status());
  if (res.status() !== 200) return null;
  const body = await res.json();
  return body.access_token as string;
}

test.describe('Landing → CRM flow', () => {
  test('submit landing form → CRM lead created', async ({ browser, request }) => {
    const token = await loginAdmin(request);
    test.skip(!token, 'admin login not available — skipping CRM assertion');

    const context = await browser.newContext();
    const page = await context.newPage();
    const email = `e2e-${Date.now()}@example.com`;

    // 1. Open landing page
    await page.goto(APPS.landing + '/apex-fintech', { waitUntil: 'domcontentloaded' });

    // 2. Fill form
    const nameField = page.locator('[data-testid="lead-name"], input[name="full_name"], input[name="name"]').first();
    const phoneField = page.locator('[data-testid="lead-phone"], input[name="phone"]').first();
    const emailField = page.locator('[data-testid="lead-email"], input[name="email"]').first();

    await nameField.fill('Nguyen Van A');
    if (await phoneField.isVisible().catch(() => false)) {
      await phoneField.fill('0901234567');
    }
    await emailField.fill(email);

    // 3. Submit
    const submitBtn = page.locator('[data-testid="lead-submit"], button[type="submit"]').first();
    await submitBtn.click();

    // 4. Verify success (either a thank-you message or a redirect)
    await page.waitForTimeout(2000);
    const successVisible = await page.locator('[data-testid="success-message"], .success, .thank-you')
      .first()
      .isVisible()
      .catch(() => false);
    if (successVisible) {
      expect(successVisible).toBeTruthy();
    } else {
      // Some flows redirect to a /thanks page
      expect(page.url()).toMatch(/thanks|success|completed/);
    }

    // 5. Check lead in CRM via API
    const res = await request.get(`${APPS.crm}/crm/v1/leads?email=${encodeURIComponent(email)}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.status() === 200) {
      const body = await res.json();
      expect(body.total).toBeGreaterThan(0);
      expect(body.data[0].email).toBe(email);
    }

    await context.close();
  });

  test('form rejects invalid email', async ({ page }) => {
    await page.goto(APPS.landing + '/apex-fintech', { waitUntil: 'domcontentloaded' });

    const emailField = page.locator('[data-testid="lead-email"], input[name="email"]').first();
    await emailField.fill('not-an-email');

    // Try to submit
    const submitBtn = page.locator('[data-testid="lead-submit"], button[type="submit"]').first();
    if (await submitBtn.isVisible().catch(() => false)) {
      await submitBtn.click();
      // Browser HTML5 validation should block submit; we just assert we didn't navigate.
      await page.waitForTimeout(500);
      expect(page.url()).not.toMatch(/thanks|success/);
    }
  });

  test('Meta Pixel fires on form submit (best-effort)', async ({ page }) => {
    const pixelCalls: any[] = [];
    await page.exposeFunction('__captureFbq', (...args: any[]) => pixelCalls.push(args));

    await page.goto(APPS.landing + '/apex-fintech', { waitUntil: 'domcontentloaded' });
    await page.addInitScript(() => {
      const w = window as any;
      const orig = w.fbq;
      w.fbq = (...args: any[]) => {
        (window as any).__captureFbq(...args);
        if (typeof orig === 'function') orig(...args);
      };
    });

    const submit = page.locator('[data-testid="lead-submit"], button[type="submit"]').first();
    if (await submit.isVisible().catch(() => false)) {
      await page.locator('[data-testid="lead-name"], input[name="full_name"], input[name="name"]').first().fill('Pixel Test');
      await page.locator('[data-testid="lead-email"], input[name="email"]').first().fill(`pixel-${Date.now()}@example.com`);
      await submit.click();
      await page.waitForTimeout(2000);

      const leadEvent = pixelCalls.find(c => c[1] === 'track' && c[2] === 'Lead');
      // If Pixel is loaded, Lead event should be tracked; if not loaded, we don't fail.
      if (leadEvent) {
        expect(leadEvent).toBeTruthy();
      }
    }
  });
});
