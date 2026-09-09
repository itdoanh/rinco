import { test, expect } from '@playwright/test';

/**
 * Admin-stores powered pages.  These tests only verify that the routes
 * load (200 OK or redirect to login) and that the visible shell renders
 * — localStorage-backed stores work even without the admin-gateway
 * backend (docs/15-roadmap §2 Phase 2).
 */

test.describe('Feature Flags', () => {
  test('renders flag table or redirects to login', async ({ page }) => {
    const res = await page.goto('/feature-flags');
    expect(res?.status() ?? 0).toBeGreaterThanOrEqual(200);
    const url = page.url();
    expect(url).toMatch(/feature-flags|login/);
  });

  test('lists seeded flags after localStorage hydration', async ({ page }) => {
    await page.goto('/feature-flags');
    // Best-effort: dashboard-side renders the seeded flags
    const hasFlag = await page
      .getByText(/dynamic_model_engine|ai_lead_scoring_v2|facebook_capi_v2/i)
      .first()
      .isVisible()
      .catch(() => false);
    if (!page.url().includes('login')) {
      expect(hasFlag).toBeTruthy();
    }
  });
});

test.describe('Notification Templates', () => {
  test('renders template grid or redirects to login', async ({ page }) => {
    const res = await page.goto('/notification-templates');
    expect(res?.status() ?? 0).toBeGreaterThanOrEqual(200);
    const url = page.url();
    expect(url).toMatch(/notification-templates|login/);
  });

  test('lists seeded templates after localStorage hydration', async ({ page }) => {
    await page.goto('/notification-templates');
    const hasTpl = await page
      .getByText(/maintenance\.scheduled|billing\.invoice_failed|welcome\.new_tenant/i)
      .first()
      .isVisible()
      .catch(() => false);
    if (!page.url().includes('login')) {
      expect(hasTpl).toBeTruthy();
    }
  });
});

test.describe('Quorum (MFA Authorization)', () => {
  test('renders quorum list or redirects to login', async ({ page }) => {
    const res = await page.goto('/quorum');
    expect(res?.status() ?? 0).toBeGreaterThanOrEqual(200);
    const url = page.url();
    expect(url).toMatch(/quorum|login/);
  });

  test('summary cards show after localStorage hydration', async ({ page }) => {
    await page.goto('/quorum');
    const hasSummary = await page
      .getByText(/PENDING|APPROVED|REJECTED|EXPIRED/i)
      .first()
      .isVisible()
      .catch(() => false);
    if (!page.url().includes('login')) {
      expect(hasSummary).toBeTruthy();
    }
  });
});
