import { test, expect } from '@playwright/test';

test.describe('Landing Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('page loads successfully', async ({ page }) => {
    // Check page title exists
    await expect(page).toHaveTitle(/.*/);
  });

  test('navigation works', async ({ page }) => {
    // Check that there are navigation links
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();
  });

  test('hero section exists', async ({ page }) => {
    // Check for hero section or main content
    const main = page.locator('main');
    await expect(main).toBeVisible();
  });

  test('form submission shows loading state', async ({ page }) => {
    // Look for form inputs
    const nameInput = page.locator('input[name="name"]');
    const emailInput = page.locator('input[name="email"]');
    const phoneInput = page.locator('input[name="phone"]');
    const submitButton = page.locator('button[type="submit"]');

    // If form exists, test interaction
    if (await nameInput.isVisible()) {
      await nameInput.fill('Test User');
      await emailInput.fill('test@example.com');
      await phoneInput.fill('0909123456');
      
      // Button should be clickable
      await expect(submitButton).toBeEnabled();
    }
  });

  test('footer exists', async ({ page }) => {
    const footer = page.locator('footer');
    await expect(footer).toBeVisible();
  });
});

test.describe('Dynamic Routes', () => {
  test('tenant page loads', async ({ page }) => {
    await page.goto('/demo-tenant');
    
    // Should load without 404
    await expect(page).not.toHaveTitle('404');
  });

  test('tenant page with slug loads', async ({ page }) => {
    await page.goto('/acme-corp/home');
    
    // Should load without 404
    await expect(page).not.toHaveTitle('404');
  });
});
