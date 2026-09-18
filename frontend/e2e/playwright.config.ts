import { defineConfig, devices } from '@playwright/test';

/**
 * Cross-app Playwright suite for RINCO frontend.
 *
 * Tests live under `tests/` and `*.spec.ts` at root for backward compatibility
 * with existing specs.  Global setup at config/global-setup.ts verifies the
 * 4 dev servers are reachable and writes logs/wst-setup.txt.
 *
 * Dev servers expected to be running on ports 3000-3003 before invoking
 * `npx playwright test`.  In CI, webServer block starts them automatically.
 */
const apps = {
  landing: 'http://localhost:3000',
  admin:   'http://localhost:3001',
  tenant:  'http://localhost:3002',
  meeting: 'http://localhost:3003',
};

const isCI = !!process.env.CI;

export default defineConfig({
  testDir: '.',
  testMatch: ['tests/**/*.spec.ts', '*.spec.ts'],
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  timeout: 120_000,
  expect: { timeout: 20_000 },
  reporter: [
    ['list'],
    ['json', { outputFile: 'logs/wst-results.json' }],
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
  ],
  globalSetup: './config/global-setup.ts',
  use: {
    baseURL: apps.landing,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    navigationTimeout: 60_000,
    actionTimeout: 20_000,
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],

  // Start all 4 dev servers automatically in CI mode.
  webServer: isCI ? {
    command: 'cd ../landing && npm run dev & cd ../admin-portal && npm run dev & cd ../tenant-site && npm run dev & cd ../meeting-ui && npm run dev & wait',
    port: 3000,
    timeout: 240_000,
    reuseExistingServer: true,
    url: 'http://localhost:3000',
  } : undefined,
});