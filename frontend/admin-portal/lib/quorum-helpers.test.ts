/**
 * Unit tests for admin-portal/lib/quorum-helpers.ts.
 *
 * Runs via plain `tsx` + `node:assert`.
 */
import { strict as assert } from "node:assert";
import { formatRemaining, formatCountdown } from "./quorum-helpers";

let passed = 0;
let failed = 0;
function it_(name: string, fn: () => void) {
  try {
    fn();
    console.log("  ✓", name);
    passed++;
  } catch (err) {
    console.error("  ✗", name, "-", (err as Error).message);
    failed++;
  }
}

// formatRemaining
it_("formatRemaining returns positive seconds for future date", () => {
  const now = 1700000000000;
  const future = new Date(now + 60_000).toISOString();
  assert.equal(formatRemaining(future, now), 60);
});
it_("formatRemaining returns negative seconds for past date", () => {
  const now = 1700000000000;
  const past = new Date(now - 30_000).toISOString();
  assert.equal(formatRemaining(past, now), -30);
});
it_("formatRemaining returns 0 for same instant", () => {
  const now = 1700000000000;
  const same = new Date(now).toISOString();
  assert.equal(formatRemaining(same, now), 0);
});
it_("formatRemaining rounds partial seconds", () => {
  const now = 1700000000000;
  const future = new Date(now + 1500).toISOString(); // 1.5 seconds
  assert.equal(formatRemaining(future, now), 2);
});

// formatCountdown
it_("formatCountdown 0 returns 0:00", () => {
  assert.equal(formatCountdown(0), "0:00");
});
it_("formatCountdown negative returns 0:00", () => {
  assert.equal(formatCountdown(-5), "0:00");
});
it_("formatCountdown under a minute", () => {
  assert.equal(formatCountdown(45), "0:45");
});
it_("formatCountdown zero-pads seconds", () => {
  assert.equal(formatCountdown(7), "0:07");
  assert.equal(formatCountdown(3), "0:03");
});
it_("formatCountdown minute boundary", () => {
  assert.equal(formatCountdown(60), "1:00");
});
it_("formatCountdown multi-minute", () => {
  assert.equal(formatCountdown(125), "2:05");
  assert.equal(formatCountdown(599), "9:59");
});
it_("formatCountdown big value", () => {
  assert.equal(formatCountdown(3600), "60:00");
  assert.equal(formatCountdown(7200), "120:00");
});

console.log(`\nResults: ${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
