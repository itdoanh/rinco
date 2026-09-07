import { defineConfig, devices } from '@playwright/test';

/**
 * Cross-app Playwright suite.
 *
 * Boots three apps (landing:3000, admin:3001, tenant:3002, meeting:3003) in
 * parallel and validates end-to-end flows that span them.
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
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : 1,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: apps.landing,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox',  use: { ...devices['Desktop Firefox'] } },
  ],
  webServer: [
    {
      command: 'cd ../landing && npm run start',
      url: apps.landing,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
    {
      command: 'cd ../admin-portal && npm run start',
      url: apps.admin,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
    {
      command: 'cd ../tenant-site && npm run start',
      url: apps.tenant,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
    {
      command: 'cd ../meeting-ui && npm run start',
      url: apps.meeting,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
})
