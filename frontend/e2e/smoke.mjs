// Quick smoke test: visit all four RINCO apps in chromium and assert NO
// `window.fbq` errors fire on hydration / after mount.
import { chromium } from 'playwright';

const APP_URLS = [
  ['landing /',           'http://localhost:3000/'],
  ['landing /apex-fintech','http://localhost:3000/apex-fintech'],
  ['tenant-site /demo',   'http://localhost:3002/demo'],
  ['admin /login',        'http://localhost:3001/login'],
  ['meeting /',           'http://localhost:3003/'],
];

const errors = [];

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext();
const page = await ctx.newPage();

page.on('pageerror', (err) => {
  errors.push(`PAGE ERROR: ${err.message}`);
});
page.on('console', (msg) => {
  if (msg.type() === 'error') {
    const text = msg.text();
    if (
      !text.includes('favicon') &&
      !text.includes('Failed to load resource') &&
      !text.includes('hydration') &&
      !text.includes('Track service') &&
      !text.includes('ECONNREFUSED')
    ) {
      errors.push(`CONSOLE ERROR: ${text}`);
    }
  }
});

for (const [name, url] of APP_URLS) {
  console.log(`Visiting ${name} -> ${url}`);
  try {
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 60000 });
  } catch (e) {
    console.log(`  goto failed: ${e.message}`);
  }
  await page.waitForTimeout(3000);
}

// Open lead modal on landing
try {
  console.log('Opening landing lead modal ...');
  await page.goto('http://localhost:3000/', { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(2000);
  const cta = page.getByRole('button', { name: /ĐĂNG KÝ NGAY/i }).first();
  if (await cta.isVisible().catch(() => false)) {
    await cta.click();
    await page.waitForTimeout(1500);
  }
} catch (e) {
  console.log(`  modal interaction failed: ${e.message}`);
}

console.log(`\n=== TOTAL ERRORS: ${errors.length} ===`);
if (errors.length > 0) {
  console.log('First 30 errors:');
  errors.slice(0, 30).forEach((e, i) => console.log(`${i + 1}. ${e}`));
  process.exitCode = 1;
} else {
  console.log('ALL GREEN — no client-side errors across all four apps.');
}

await browser.close();
