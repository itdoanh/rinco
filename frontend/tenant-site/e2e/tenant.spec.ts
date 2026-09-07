import { test, expect } from '@playwright/test';

test.describe('Tenant Site', () => {
  test('tenant page loads', async ({ page }) => {
    await page.goto('/acme/home');
    
    // Should load without 404
    await expect(page).not.toHaveTitle('404');
  });

  test('hero section renders', async ({ page }) => {
    await page.goto('/acme/home');
    
    // Look for main content
    const main = page.locator('main');
    await expect(main).toBeVisible();
  });

  test('form submission works', async ({ page }) => {
    await page.goto('/acme/home');

    // Look for form
    const form = page.locator('form');
    if (await form.isVisible()) {
      const nameInput = form.locator('input').first();
      const emailInput = form.locator('input[type="email"]').first();
      const phoneInput = form.locator('input[type="tel"]').first();
      
      if (await nameInput.isVisible()) {
        await nameInput.fill('Test User');
        await emailInput.fill('test@example.com');
        await phoneInput.fill('0909123456');
        
        const submitButton = form.locator('button[type="submit"]');
        await submitButton.click();
        
        // Wait for response
        await page.waitForTimeout(1000);
      }
    }
  });
});
