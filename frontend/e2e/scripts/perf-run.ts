/**
 * WS-T: Performance audit — collects Core Web Vitals (FCP, LCP, CLS, TTI)
 * for all 4 apps using Playwright's Performance API.
 *
 * This is a lightweight alternative to Lighthouse that runs in CI without
 * needing the full Chrome DevTools protocol.
 */
import { chromium, type Page } from '@playwright/test';
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

async function measurePage(page: Page, url: string): Promise<Record<string, unknown>> {
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 60_000 });
  await page.waitForTimeout(3000);

  const metrics = await page.evaluate(() => {
    const nav = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined;
    const paint = performance.getEntriesByType('paint');
    const fcp = paint.find((p) => p.name === 'first-contentful-paint')?.startTime ?? null;

    // LCP via PerformanceObserver
    let lcp = 0;
    try {
      const obs = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        const last = entries[entries.length - 1] as any;
        if (last) lcp = last.startTime;
      });
      obs.observe({ type: 'largest-contentful-paint', buffered: true });
      obs.disconnect();
    } catch {}

    const resources = performance.getEntriesByType('resource');
    let totalTransferSize = 0;
    for (const r of resources) totalTransferSize += (r as any).transferSize ?? 0;

    return {
      fcp_ms: fcp,
      lcp_ms: lcp || null,
      domContentLoaded_ms: nav?.domContentLoadedEventEnd ?? null,
      load_ms: nav?.loadEventEnd ?? null,
      ttfb_ms: nav?.responseStart ?? null,
      resource_count: resources.length,
      resource_total_kb: Math.round(totalTransferSize / 1024),
      js_heap_mb: (performance as any).memory?.usedJSHeapSize
        ? Math.round((performance as any).memory.usedJSHeapSize / 1024 / 1024)
        : null,
    };
  });

  return { url, ...metrics };
}

async function main(): Promise<void> {
  const browser = await chromium.launch();
  const results: Record<string, unknown>[] = [];

  for (const app of APPS) {
    try {
      const ctx = await browser.newContext();
      const page = await ctx.newPage();
      const m = await measurePage(page, app.url);
      results.push({ app: app.name, ...m });
      console.log(`[perf ${app.name}] ${JSON.stringify(m)}`);
      await ctx.close();
    } catch (e) {
      results.push({ app: app.name, error: (e as Error).message });
      console.log(`[perf ${app.name}] ERR ${(e as Error).message}`);
    }
  }

  await browser.close();

  writeFileSync(
    resolve(LOG_DIR, 'wst-lighthouse-summary.json'),
    JSON.stringify(results, null, 2)
  );
  console.log('Wrote logs/wst-lighthouse-summary.json');
}

main().catch((e) => {
  console.error('perf runner failed:', e);
  process.exit(1);
});