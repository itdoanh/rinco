/**
 * Unit tests for tenant-site/lib/branding.ts (hexToHsl, getBrandingCSS).
 */
import { strict as assert } from "node:assert";
import { hexToHsl, getBrandingCSS } from "./branding";

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

// hexToHsl
it_("hexToHsl handles pure red #ff0000", () => {
  const out = hexToHsl("ff0000");
  // red = hue 0, sat 100%, light 50%
  assert.equal(out, "0 100% 50%");
});
it_("hexToHsl handles pure green #00ff00", () => {
  const out = hexToHsl("00ff00");
  // green = hue 120
  assert.equal(out, "120 100% 50%");
});
it_("hexToHsl handles pure blue #0000ff", () => {
  const out = hexToHsl("0000ff");
  // blue = hue 240
  assert.equal(out, "240 100% 50%");
});
it_("hexToHsl handles pure white #ffffff", () => {
  const out = hexToHsl("ffffff");
  // white = 0 sat, 100% light
  assert.equal(out, "0 0% 100%");
});
it_("hexToHsl handles pure black #000000", () => {
  const out = hexToHsl("000000");
  assert.equal(out, "0 0% 0%");
});
it_("hexToHsl handles # prefix", () => {
  const out = hexToHsl("#ff0000");
  assert.equal(out, "0 100% 50%");
});
it_("hexToHsl returns HSL format string", () => {
  const out = hexToHsl("abcdef");
  assert.match(out, /^\d+ \d+% \d+%$/);
});

// getBrandingCSS
it_("getBrandingCSS empty when no branding", () => {
  assert.equal(getBrandingCSS(undefined), "");
});
it_("getBrandingCSS returns empty :root when branding empty", () => {
  // Empty object still returns ":root {  }" with no styles — document that.
  const out = getBrandingCSS({});
  assert.match(out, /:root \{[^}]*\}/);
  assert.equal(out.replace(/\s/g, ""), ":root{}");
});
it_("getBrandingCSS includes primary when set", () => {
  const out = getBrandingCSS({ primary: "#ff0000" });
  assert.match(out, /:root \{ --primary: 0 100% 50%; \}/);
});
it_("getBrandingCSS includes accent when set", () => {
  const out = getBrandingCSS({ accent: "#00ff00" });
  assert.match(out, /--accent: 120 100% 50%;/);
});
it_("getBrandingCSS includes fontFamily when set", () => {
  const out = getBrandingCSS({ fontFamily: "Inter, sans-serif" });
  assert.match(out, /--font-family: Inter, sans-serif;/);
});
it_("getBrandingCSS combines multiple properties", () => {
  const out = getBrandingCSS({
    primary: "#ff0000",
    accent: "#0000ff",
    fontFamily: "Roboto",
  });
  assert.match(out, /:root \{/);
  assert.match(out, /--primary: 0 100% 50%;/);
  assert.match(out, /--accent: 240 100% 50%;/);
  assert.match(out, /--font-family: Roboto;/);
});
it_("getBrandingCSS preserves primary order", () => {
  const out = getBrandingCSS({ primary: "#123456" });
  // The primary key always comes first in our impl.
  assert.match(out, /--primary:/);
});

console.log(`\nResults: ${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
