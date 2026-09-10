import { defineConfig, devices } from '@playwright/test';

/**
 * Cross-app Playwright suite.
 *
 * Assumes apps are already running on their ports (3000-3003) via `npm run dev`.
 */
const apps = {
  landing: 'http://localhost:3000',
  admin:   'http://localhost:3001',
  tenant:  'http://localhost:3002',
  meeting: 'http://localhost:3003',
}

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.ts',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: apps.landing,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
})
