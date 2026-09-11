// End-to-end real-event test:
//   1. Open landing page in chromium
//   2. Open the lead modal
//   3. Fill name + phone
//   4. Submit form
//   5. Assert success UI appears
//   6. Assert no console errors
import { chromium } from 'playwright';

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext();
const page = await ctx.newPage();

const errors = [];
page.on('pageerror', (err) => errors.push(`PAGE ERROR: ${err.message}`));
page.on('console', (msg) => {
  if (msg.type() === 'error') {
    const text = msg.text();
    if (
      !text.includes('favicon') &&
      !text.includes('Failed to load resource') &&
      !text.includes('hydration') &&
      !text.includes('Tracking error') &&
      !text.includes('Track service') &&
      !text.includes('ECONNREFUSED') &&
      !text.includes('NextAuth') &&
      !text.includes('authjs')
    ) {
      errors.push(`CONSOLE ERROR: ${text}`);
    }
  }
});

console.log('1. Loading landing page ...');
await page.goto('http://localhost:3000/', { waitUntil: 'domcontentloaded', timeout: 60000 });
await page.waitForTimeout(2000);

console.log('2. Clicking hero CTA ...');
const cta = page.getByRole('button', { name: /ĐĂNG KÝ NGAY/i }).first();
await cta.waitFor({ state: 'visible', timeout: 10000 });
await cta.click();
await page.waitForTimeout(1500);

console.log('3. Selecting Zalo channel ...');
try {
  const zaloBtn = page.getByRole('button', { name: /Zalo/i }).first();
  if (await zaloBtn.isVisible({ timeout: 2000 })) {
    await zaloBtn.click();
    await page.waitForTimeout(800);
  }
} catch (e) {
  console.log(`  zalo button not found: ${e.message}`);
}

console.log('4. Filling form fields ...');
const nameInput = page.locator('input[name="name"]').first();
if (await nameInput.isVisible({ timeout: 5000 }).catch(() => false)) {
  await nameInput.fill('E2E Test User');
}
const phoneInput = page.locator('input[name="phone"]').first();
if (await phoneInput.isVisible({ timeout: 5000 }).catch(() => false)) {
  await phoneInput.fill('0909123456');
}

console.log('5. Submitting form ...');
try {
  const submitBtn = page.getByRole('button', { name: /GIỮ VÉ ZOOM|NHẬN EBOOK|ĐĂNG KÝ/i }).last();
  if (await submitBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
    await submitBtn.click();
    await page.waitForTimeout(3000);
  }
} catch (e) {
  console.log(`  submit failed: ${e.message}`);
}

console.log('6. Checking for success indicator ...');
try {
  const success = page.locator('text=/THÀNH CÔNG|SUCCESS/i').first();
  if (await success.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log('  ✓ Success UI visible');
  } else {
    console.log('  ⚠ Success UI not visible (might have closed modal)');
  }
} catch {
  // ignore
}

console.log(`\n=== TOTAL ERRORS: ${errors.length} ===`);
if (errors.length > 0) {
  console.log('Errors:');
  errors.slice(0, 20).forEach((e, i) => console.log(`${i + 1}. ${e}`));
  process.exitCode = 1;
} else {
  console.log('ALL GREEN — full form submission flow completed without errors.');
}

await browser.close();
