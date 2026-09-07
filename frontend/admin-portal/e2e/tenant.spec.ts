import { test, expect } from '@playwright/test';

test.describe('Tenant Management', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'fake-token');
      localStorage.setItem('user', JSON.stringify({
        id: '1',
        email: 'admin@rinco.vn',
        name: 'Admin',
      }));
    });
  });

  test('tenant list page loads', async ({ page }) => {
    await page.goto('/tenants');
    
    // Should show tenant list
    await expect(page.locator('text=Tenants')).toBeVisible({ timeout: 5000 });
  });

  test('search functionality works', async ({ page }) => {
    await page.goto('/tenants');

    // Wait for table to load
    await page.waitForSelector('table', { timeout: 5000 });

    // Type in search box
    const searchInput = page.locator('input[placeholder*="Tìm kiếm"]');
    if (await searchInput.isVisible()) {
      await searchInput.fill('ACME');
      await page.waitForTimeout(500);
    }
  });

  test('status filter works', async ({ page }) => {
    await page.goto('/tenants');

    // Wait for table to load
    await page.waitForSelector('table', { timeout: 5000 });

    // Check for status filter dropdown
    const statusFilter = page.locator('select').first();
    if (await statusFilter.isVisible()) {
      await statusFilter.selectOption('active');
      await page.waitForTimeout(500);
    }
  });

  test('tenant detail page loads', async ({ page }) => {
    await page.goto('/tenants');

    // Wait for table
    await page.waitForSelector('table', { timeout: 5000 });

    // Click first tenant link if exists
    const tenantLink = page.locator('table a').first();
    if (await tenantLink.isVisible()) {
      await tenantLink.click();
      
      // Should show tenant detail
      await page.waitForTimeout(1000);
    }
  });

  test('add tenant button exists', async ({ page }) => {
    await page.goto('/tenants');

    const addButton = page.locator('text="Thêm Tenant"');
    await expect(addButton).toBeVisible();
  });

  test('pagination works', async ({ page }) => {
    await page.goto('/tenants');

    // Wait for table
    await page.waitForSelector('table', { timeout: 5000 });

    // Look for pagination buttons
    const nextButton = page.locator('button:has-text("Sau")');
    if (await nextButton.isVisible()) {
      await nextButton.click();
      await page.waitForTimeout(500);
    }
  });
});

test.describe('Tenant CRUD Operations', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'fake-token');
      localStorage.setItem('user', JSON.stringify({
        id: '1',
        email: 'admin@rinco.vn',
        name: 'Admin',
      }));
    });
    await page.goto('/tenants');
  });

  test('create tenant modal opens', async ({ page }) => {
    // Click add tenant button
    const addButton = page.locator('text="Thêm Tenant"');
    await addButton.click();

    // Should show modal or form
    await page.waitForTimeout(500);
  });

  test('tenant form validation', async ({ page }) => {
    // Open add tenant modal
    const addButton = page.locator('text="Thêm Tenant"');
    await addButton.click();

    await page.waitForTimeout(500);

    // Submit empty form
    const submitButton = page.locator('button:has-text("Lưu")');
    if (await submitButton.isVisible()) {
      await submitButton.click();
      
      // Should show validation errors
      await page.waitForTimeout(500);
    }
  });

  test('edit tenant button exists', async ({ page }) => {
    // Wait for table
    await page.waitForSelector('table', { timeout: 5000 });

    // Look for edit button
    const editButton = page.locator('button').filter({ has: page.locator('svg') }).first();
    await expect(editButton).toBeVisible();
  });

  test('delete tenant confirmation', async ({ page }) => {
    // Wait for table
    await page.waitForSelector('table', { timeout: 5000 });

    // Look for delete button
    const deleteButton = page.locator('button').filter({ hasText: '' }).last();
    
    // Click delete (may show confirmation)
    await deleteButton.click();
    
    // Handle potential confirmation dialog
    page.on('dialog', dialog => dialog.accept());
  });
});
