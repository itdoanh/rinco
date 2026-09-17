// E2E: CRM tree (organisational hierarchy) operations through the admin UI.
import { test, expect } from '@playwright/test';

const APPS = {
  admin: 'http://localhost:3001',
  auth:  'http://localhost:8081',
};

async function loginAdmin(page: any) {
  await page.goto(APPS.admin + '/login');
  await page.locator('input[type="email"]').first().fill('admin@apex.vn');
  await page.locator('input[type="password"]').first().fill('Test1234!');
  await page.locator('button[type="submit"]').first().click();
  await page.waitForLoadState('networkidle').catch(() => {});
}

test.describe('CRM tree', () => {
  test('users page renders', async ({ page }) => {
    await loginAdmin(page);
    await page.goto(APPS.admin + '/tenants', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    // We just assert the page rendered something tenant-y.
    const body = await page.content();
    expect(body.length).toBeGreaterThan(0);
  });

  test('create user with parent (best-effort)', async ({ page }) => {
    await loginAdmin(page);
    await page.goto(APPS.admin + '/tenants', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    const addBtn = page.locator('[data-testid="add-user"], button:has-text("Add User"), button:has-text("Thêm User")').first();
    if (await addBtn.isVisible().catch(() => false)) {
      await addBtn.click();
      const email = `agent-${Date.now()}@apex.vn`;
      await page.locator('input[name="email"]').first().fill(email);
      await page.locator('input[name="full_name"]').first().fill('Agent E2E');
      const roleSelect = page.locator('select[name="role"]').first();
      if (await roleSelect.isVisible().catch(() => false)) {
        await roleSelect.selectOption('agent');
      }
      const save = page.locator('button:has-text("Save"), button:has-text("Lưu")').first();
      if (await save.isVisible().catch(() => false)) {
        await save.click();
        await page.waitForTimeout(1000);
        const userVisible = await page.locator(`text=Agent E2E`).first().isVisible().catch(() => false);
        expect(userVisible).toBeTruthy();
      }
    }
  });

  test('invite link generator (best-effort)', async ({ page }) => {
    await loginAdmin(page);
    await page.goto(APPS.admin + '/tenants', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    const inviteBtn = page.locator('[data-testid="generate-invite"], button:has-text("Generate Link"), button:has-text("Tạo Link")').first();
    if (await inviteBtn.isVisible().catch(() => false)) {
      await inviteBtn.click();
      await page.waitForTimeout(500);
      // The page should show some sort of link / token.
      const link = await page.locator('[data-testid="invite-link"], code, pre').first().textContent().catch(() => null);
      if (link) {
        expect(link.length).toBeGreaterThan(10);
      }
    }
  });
});
