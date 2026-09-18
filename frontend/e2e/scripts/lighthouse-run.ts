/**
 * WS-T: Lighthouse runner — programmatically runs Lighthouse for all 4 apps
 * and writes JSON results to logs/wst-lighthouse-<app>.json.
 *
 * Uses chrome-launcher + lighthouse npm packages (already installed).
 */
import { launch } from 'chrome-launcher';
import lighthouse from 'lighthouse';
import { writeFileSync, mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const APPS = [
  { name: 'landing', url: 'http://localhost:3000/' },
  { name: 'admin',   url: 'http://localhost:3001/login' },
  { name: 'tenant',  url: 'http://localhost:3002/demo' },
  { name: 'meeting', url: 'http://localhost:3003/' },
];

const LOG_DIR = resolve(__dirname, '..', '..', 'logs');
mkdirSync(LOG_DIR, { recursive: true });

async function runOne(app: typeof APPS[number], chrome: any): Promise<void> {
  const log: string[] = [];
  log.push(`[Lighthouse ${app.name}] ${new Date().toISOString()} url=${app.url}`);

  try {
    const res = await lighthouse(app.url, {
      port: chrome.port,
      output: 'json',
      logLevel: 'error',
      onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo'],
      formFactor: 'desktop',
      screenEmulation: { mobile: false, width: 1350, height: 940, deviceScaleFactor: 1, disabled: false },
      throttling: { rttMs: 40, throughputKbps: 10240, cpuSlowdownMultiplier: 1, requestLatencyMs: 0, downloadThroughputKbps: 0, uploadThroughputKbps: 0 },
    });
    if (!res) throw new Error('no result');
    const { categories, finalUrl } = res.lhr;
    const scores = Object.entries(categories).map(([k, v]) => `${k}=${(v.score ?? 0) * 100}`).join(' ');
    log.push(`  url=${finalUrl} ${scores}`);
    writeFileSync(resolve(LOG_DIR, `wst-lighthouse-${app.name}.json`), JSON.stringify(res.lhr, null, 2));
  } catch (e) {
    log.push(`  ERR ${(e as Error).message}`);
  }

  writeFileSync(resolve(LOG_DIR, 'wst-lighthouse.txt'), log.join('\n') + '\n', { flag: 'a' });
}

async function main(): Promise<void> {
  const chrome = await launch({
    chromeFlags: ['--headless=new', '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage'],
  });

  try {
    for (const app of APPS) {
      await runOne(app, chrome);
    }
  } finally {
    await chrome.kill();
  }
}

main().catch((e) => {
  console.error('lighthouse runner failed:', e);
  process.exit(1);
});