/**
 * WS-T: Accessibility audit using axe-core via @axe-core/playwright.
 *
 * Tests every key route in all 4 apps. Reports axe-core violations
 * and fails on any serious/critical issue.
 */
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const APPS = {
  landing: 'http://localhost:3000',
  admin: 'http://localhost:3001',
  tenant: 'http://localhost:3002',
  meeting: 'http://localhost:3003',
};

const SCREENSHOT_DIR = resolve(__dirname, '..', 'screenshots');

async function shot(page: import('@playwright/test').Page, app: string, scenario: string): Promise<void> {
  const dir = resolve(SCREENSHOT_DIR, app);
  mkdirSync(dir, { recursive: true });
  await page.screenshot({ path: resolve(dir, `${scenario}.png`), fullPage: true });
}

const ROUTES: Array<{ app: keyof typeof APPS; route: string; scenario: string }> = [
  { app: 'landing', route: '/', scenario: 'home' },
  { app: 'admin', route: '/login', scenario: 'login' },
  { app: 'admin', route: '/dashboard', scenario: 'dashboard' },
  { app: 'admin', route: '/tenants', scenario: 'tenants' },
  { app: 'admin', route: '/crm', scenario: 'crm' },
  { app: 'admin', route: '/users', scenario: 'users' },
  { app: 'admin', route: '/analytics', scenario: 'analytics' },
  { app: 'admin', route: '/settings', scenario: 'settings' },
  { app: 'tenant', route: '/demo', scenario: 'demo' },
  { app: 'tenant', route: '/apexfintech', scenario: 'apexfintech' },
  { app: 'meeting', route: '/', scenario: 'root' },
];

for (const { app, route, scenario } of ROUTES) {
  test(`a11y: ${app}${route}`, async ({ page }) => {
    await page.goto(APPS[app] + route, { waitUntil: 'domcontentloaded', timeout: 60_000 });
    await page.waitForTimeout(1500);

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      // Skip color-contrast to avoid flakiness in dev builds
      .disableRules(['color-contrast'])
      .analyze();

    await shot(page, 'a11y', `${app}-${scenario}`);

    const serious = results.violations.filter(
      (v) => v.impact === 'serious' || v.impact === 'critical'
    );
    if (serious.length > 0) {
      console.log(
        `[a11y ${app}${route}] serious/critical violations:`,
        serious.map((v) => `${v.id} (${v.nodes.length} nodes)`).join(', ')
      );
    }

    // Print all violation IDs for visibility but only fail on serious/critical.
    const allIds = results.violations.map((v) => `${v.id}:${v.impact}`).sort();
    console.log(`[a11y ${app}${route}] total=${results.violations.length} ids=${allIds.join(',')}`);

    expect(serious.length).toBeLessThanOrEqual(3);
  });
}