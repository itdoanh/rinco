import { test, expect } from '@playwright/test';

test.describe('Meeting UI', () => {
  test('meeting room loads', async ({ page }) => {
    await page.goto('/meeting/test-room-123');
    
    // Should load without error
    await page.waitForTimeout(2000);
  });

  test('control bar is visible', async ({ page }) => {
    await page.goto('/meeting/test-room-123');
    
    // Look for control buttons
    const controls = page.locator('[class*="control"]');
    await expect(controls.first()).toBeVisible({ timeout: 5000 });
  });
});
