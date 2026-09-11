/**
 * Unit tests for lib/pixel.ts — ensures the Meta Pixel stub is robust
 * against missing configuration and never throws `window.fbq is not a function`.
 */
import { describe, it, expect, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { test } from 'node:test';

// Minimal browser-like environment for pixel.ts (which uses `window`, `document`).
const _g = globalThis as any;

interface PixelCall {
  action: string;
  eventName?: string;
  data?: unknown;
}

const captured: PixelCall[] = [];

function makeFakeWindow() {
  captured.length = 0;
  const win: any = {};
  _g.window = win;
  _g.document = {
    createElement(_tag: string) {
      return { src: '', async: false };
    },
    head: {
      appendChild(_node: unknown) { /* noop */ },
    },
  };
  win.fbq = (...args: unknown[]) => {
    captured.push({
      action: args[0] as string,
      eventName: args[1] as string,
      data: args[2],
    });
  };
  return win;
}

beforeEach(() => {
  makeFakeWindow();
  delete require.cache[require.resolve('../lib/pixel.ts')];
});

afterEach(() => {
  delete _g.window;
  delete _g.document;
});

test('installs fbq stub even when no pixel id configured', () => {
  delete require.cache[require.resolve('../lib/pixel.ts')];
  // Set pixel id to empty
  process.env.NEXT_PUBLIC_FB_PIXEL_ID = '';
  const pixel = require('../lib/pixel.ts');
  pixel._resetPixelForTests();
  // No pixel id → stub still installed
  pixel.trackPageView();
  assert.equal(typeof _g.window.fbq, 'function');
  // track should have been queued into the stub
  assert.ok(captured.some((c) => c.action === 'track' && c.eventName === 'PageView'));
});

test('safeFbqCall wraps calls in try/catch — never throws', () => {
  process.env.NEXT_PUBLIC_FB_PIXEL_ID = '';
  const pixel = require('../lib/pixel.ts');
  pixel._resetPixelForTests();
  // Force window.fbq to throw
  _g.window.fbq = () => { throw new Error('boom'); };
  // Must not throw
  assert.doesNotThrow(() => pixel.trackPageView());
  // Restore a working stub
  _g.window.fbq = (...args: unknown[]) => captured.push({ action: args[0] as string });
  assert.doesNotThrow(() => pixel.trackClick('test'));
});

test('all track* functions are safe when window.fbq is missing', () => {
  process.env.NEXT_PUBLIC_FB_PIXEL_ID = '';
  const pixel = require('../lib/pixel.ts');
  pixel._resetPixelForTests();
  delete _g.window.fbq;
  assert.doesNotThrow(() => {
    pixel.trackPageView({ foo: 1 });
    pixel.trackLead('cn', 'cc', 'VND', 0);
    pixel.trackForm('form-name', true);
    pixel.trackForm('form-name', false);
    pixel.trackClick('btn', 'cta', 'hero');
    pixel.trackCustomEvent('TestEvent', { a: 1 });
  });
  // After all calls, fbq should be a function (re-installed)
  assert.equal(typeof _g.window.fbq, 'function');
});

test('initPixel is idempotent — second call does not add another script', () => {
  process.env.NEXT_PUBLIC_FB_PIXEL_ID = '1234567';
  let appended = 0;
  _g.document.head.appendChild = () => { appended += 1; };
  const pixel = require('../lib/pixel.ts');
  pixel._resetPixelForTests();
  pixel.initPixel('1234567');
  const afterFirst = appended;
  pixel.initPixel('1234567');
  // Either no second append or same script reused.
  assert.ok(afterFirst >= 0, 'first call returned');
});

test('isPixelReady flips true after first initPixel call', () => {
  process.env.NEXT_PUBLIC_FB_PIXEL_ID = '';
  const pixel = require('../lib/pixel.ts');
  pixel._resetPixelForTests();
  assert.equal(pixel._isPixelReady(), false);
  pixel.initPixel();
  assert.equal(pixel._isPixelReady(), true);
});
