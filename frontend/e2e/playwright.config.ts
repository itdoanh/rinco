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
  timeout: 90_000,
  expect: { timeout: 15_000 },
  reporter: [['list']],
  use: {
    baseURL: apps.landing,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    navigationTimeout: 60_000,
    actionTimeout: 15_000,
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],

  // Start all 4 dev servers automatically in CI mode.
  // In local dev, set CI=true to also use this, or start servers manually.
  webServer: process.env.CI ? {
    command: 'bash -c "npm run dev -- -p 3000 & npm run dev -- -p 3001 & npm run dev -- -p 3002 & npm run dev -- -p 3003 & wait"',
    port: 3000,
    timeout: 180_000,
    reuseExistingServer: true,
    url: 'http://localhost:3000',
  } : undefined,
})
