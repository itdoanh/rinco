import { test, expect } from '@playwright/test';

test.describe('Admin Portal Authentication', () => {
  test('login page loads', async ({ page }) => {
    await page.goto('/');
    
    // Should redirect to login or show login form
    await expect(page.locator('body')).toBeVisible();
  });

  test('login form validation', async ({ page }) => {
    await page.goto('/(auth)/login');

    // Check for email input
    const emailInput = page.locator('input[name="email"]');
    const passwordInput = page.locator('input[name="password"]');
    const submitButton = page.locator('button[type="submit"]');

    if (await emailInput.isVisible()) {
      // Try submitting empty form
      await submitButton.click();
      
      // Should show validation error
      await expect(emailInput).toHaveAttribute('required', '');
    }
  });

  test('login with invalid credentials', async ({ page }) => {
    await page.goto('/(auth)/login');

    const emailInput = page.locator('input[name="email"]');
    const passwordInput = page.locator('input[name="password"]');

    if (await emailInput.isVisible()) {
      await emailInput.fill('invalid@example.com');
      await passwordInput.fill('wrongpassword');
      
      const submitButton = page.locator('button[type="submit"]');
      await submitButton.click();

      // Wait for error message or redirect
      await page.waitForTimeout(1000);
    }
  });

  test('successful login redirects to dashboard', async ({ page }) => {
    await page.goto('/(auth)/login');

    const emailInput = page.locator('input[name="email"]');
    const passwordInput = page.locator('input[name="password"]');

    if (await emailInput.isVisible()) {
      // Note: This test would need actual valid credentials in a real environment
      await emailInput.fill('admin@rinco.vn');
      await passwordInput.fill('password');
      
      const submitButton = page.locator('button[type="submit"]');
      await submitButton.click();

      // Should redirect to dashboard
      await expect(page).toHaveURL(/\/dashboard/);
    }
  });
});

test.describe('Dashboard Access Control', () => {
  test('unauthenticated user redirected to login', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Should redirect to login page
    await expect(page).toHaveURL(/\(auth\)\/login/);
  });

  test('authenticated user can access dashboard', async ({ page }) => {
    // Set auth token in localStorage
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'fake-token');
      localStorage.setItem('user', JSON.stringify({
        id: '1',
        email: 'admin@rinco.vn',
        name: 'Admin',
      }));
    });

    await page.goto('/dashboard');
    
    // Should see dashboard content
    await expect(page.locator('text=Dashboard')).toBeVisible({ timeout: 5000 });
  });
});
