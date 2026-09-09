/**
 * Unit tests for the shared UI utility helpers in
 * ``packages/frontend/ui/lib/utils.ts``.
 *
 * Run with:
 *     npx tsx packages/frontend/ui/test/utils.test.ts
 *
 * Self-contained — no vitest/jest required.
 */
import {
  cn,
  formatCurrency,
  formatNumber,
  formatDate,
  formatDateTime,
  slugify,
  truncate,
  getInitials,
  sleep,
  debounce,
  generateId,
  generateUuid,
  clamp,
  parseFloatSafe,
  INVALID_FORMAT_PLACEHOLDER,
} from "../lib/utils";

// -------------------- tiny test runner --------------------
let passed = 0;
let failed = 0;
function assert(cond: unknown, label: string): void {
  if (cond) {
    passed++;
  } else {
    failed++;
    console.error(`FAIL ${label}`);
  }
}
function assertEqual<T>(actual: T, expected: T, label: string): void {
  if (actual === expected) {
    passed++;
  } else {
    failed++;
    console.error(
      `FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(
        expected,
      )}`,
    );
  }
}
const tests: Array<{ name: string; fn: () => void | Promise<void> }> = [];
function test(name: string, fn: () => void | Promise<void>): void {
  tests.push({ name, fn });
}

// -------------------- cn --------------------
test("cn merges classes and resolves tailwind conflicts", () => {
  assertEqual(cn("px-2", "px-4"), "px-4", "later px wins");
  assertEqual(
    cn("text-red-500", { "text-blue-500": true }),
    "text-blue-500",
    "object syntax works",
  );
  assertEqual(
    cn("a", false && "b", undefined, "c"),
    "a c",
    "falsy values ignored",
  );
});

// -------------------- formatCurrency --------------------
test("formatCurrency formats VND by default", () => {
  const formatted = formatCurrency(1_000_000);
  // Exact formatting depends on ICU data but the prefix should be there.
  assert(formatted.includes("1.000.000"), `1.000.000 visible: ${formatted}`);
  assert(formatted.includes("₫") || formatted.includes("đ"), "₫ symbol visible");
});

test("formatCurrency handles negative & zero", () => {
  const a = formatCurrency(0);
  assert(a.includes("0"), `0 visible: ${a}`);
  const b = formatCurrency(-500);
  assert(b.includes("500"), `500 visible in negative: ${b}`);
});

test("formatCurrency returns sentinel for non-finite input", () => {
  assertEqual(formatCurrency(NaN), INVALID_FORMAT_PLACEHOLDER, "NaN");
  assertEqual(formatCurrency(Infinity), INVALID_FORMAT_PLACEHOLDER, "Infinity");
  assertEqual(
    formatCurrency(null as unknown as number),
    INVALID_FORMAT_PLACEHOLDER,
    "null",
  );
  assertEqual(
    formatCurrency(undefined as unknown as number),
    INVALID_FORMAT_PLACEHOLDER,
    "undefined",
  );
});

test("formatCurrency falls back to plain number for unknown currency codes", () => {
  const out = formatCurrency(1000, "XYZ");
  assert(typeof out === "string", "string output");
  assert(out.includes("XYZ"), `currency code echoed: ${out}`);
});

// -------------------- formatNumber --------------------
test("formatNumber adds thousand separators", () => {
  assert(formatNumber(1234567).includes("1.234.567"), "separator visible");
});

test("formatNumber sentinel for bad input", () => {
  assertEqual(formatNumber(NaN), INVALID_FORMAT_PLACEHOLDER, "NaN");
  assertEqual(
    formatNumber(undefined as unknown as number),
    INVALID_FORMAT_PLACEHOLDER,
    "undefined",
  );
});

// -------------------- formatDate / formatDateTime --------------------
test("formatDate accepts Date and string", () => {
  const out1 = formatDate(new Date(2024, 0, 5));
  const out2 = formatDate("2024-01-05");
  assert(out1.length > 0, "date renders");
  assert(out2.length > 0, "string renders");
  // Both should contain "01" (day) and "2024" (year).
  assert(out1.includes("01") && out1.includes("2024"), "01 + 2024 present");
  assert(out2.includes("01") && out2.includes("2024"), "01 + 2024 present");
});

test("formatDate returns sentinel for invalid input", () => {
  assertEqual(formatDate(null), INVALID_FORMAT_PLACEHOLDER, "null");
  assertEqual(formatDate(undefined), INVALID_FORMAT_PLACEHOLDER, "undefined");
  assertEqual(formatDate("not-a-date"), INVALID_FORMAT_PLACEHOLDER, "garbage");
  assertEqual(
    formatDate(new Date("not-a-date")),
    INVALID_FORMAT_PLACEHOLDER,
    "invalid Date",
  );
});

test("formatDateTime renders hh:mm component", () => {
  const d = new Date(2024, 5, 15, 9, 5);
  const out = formatDateTime(d);
  assert(out.includes("09") || out.includes("9"), `hour visible: ${out}`);
  assert(out.includes("05"), `minute visible: ${out}`);
});

// -------------------- slugify --------------------
test("slugify normalises Vietnamese", () => {
  assertEqual(slugify("Tiếng Việt có dấu"), "tieng-viet-co-dau", "vi diacritics");
  assertEqual(slugify("Hello World"), "hello-world", "ascii");
  assertEqual(slugify("  spaces  "), "spaces", "trim leading/trailing");
  assertEqual(slugify("multiple   spaces"), "multiple-spaces", "collapse");
  assertEqual(slugify("a!@#$b"), "a-b", "non-alphanumeric");
  assertEqual(slugify(""), "", "empty");
  assertEqual(slugify(null), "", "null");
  assertEqual(slugify(undefined), "", "undefined");
});

// -------------------- truncate --------------------
test("truncate respects length boundary", () => {
  assertEqual(truncate("hello", 10), "hello", "shorter than limit");
  assertEqual(truncate("hello world", 5), "hello...", "longer than limit");
  assertEqual(truncate("hello", 5), "hello", "exact length");
  assertEqual(truncate("hi", 5, "…"), "hi", "shorter with custom suffix");
  assertEqual(truncate("hello world", 5, "…"), "hello…", "custom suffix");
});

test("truncate handles bad length and null input", () => {
  assertEqual(truncate("hello", 0), "", "length 0");
  assertEqual(truncate("hello", -1), "", "negative length");
  assertEqual(truncate(null, 5), "", "null");
  assertEqual(truncate(undefined, 5), "", "undefined");
});

// -------------------- getInitials --------------------
test("getInitials basic cases", () => {
  assertEqual(getInitials("John Doe"), "JD", "two words");
  assertEqual(getInitials("alice"), "A", "single word");
  assertEqual(getInitials("Mary Jane Smith"), "MJ", "first two only");
  assertEqual(getInitials(""), "", "empty");
  assertEqual(getInitials(null), "", "null");
  assertEqual(getInitials(undefined), "", "undefined");
  assertEqual(getInitials("   "), "", "whitespace only");
  assertEqual(getInitials("a b c d"), "AB", "max 2");
});

// -------------------- sleep --------------------
test("sleep resolves after delay", async () => {
  const start = Date.now();
  await sleep(40);
  const elapsed = Date.now() - start;
  assert(elapsed >= 30, `elapsed ${elapsed}ms`);
});

test("sleep resolves immediately for invalid input", async () => {
  const start = Date.now();
  await sleep(-1);
  await sleep(NaN);
  assert(Date.now() - start < 10, "immediate");
});

// -------------------- debounce --------------------
test("debounce coalesces rapid calls", async () => {
  let calls = 0;
  const fn = () => calls++;
  const d = debounce(fn, 20);
  d();
  d();
  d();
  assertEqual(calls, 0, "not called yet");
  await sleep(50);
  assertEqual(calls, 1, "called once");
});

test("debounce.cancel prevents pending call", async () => {
  let calls = 0;
  const fn = () => calls++;
  const d = debounce(fn, 20);
  d();
  d.cancel();
  await sleep(50);
  assertEqual(calls, 0, "cancelled");
});

test("debounce.flush fires immediately with last args", async () => {
  let lastArg: number | undefined;
  const fn = (n: number) => (lastArg = n);
  const d = debounce(fn, 1000);
  d(1);
  d(2);
  d.flush();
  assertEqual(lastArg, 2, "flush fires with latest args");
});

test("debounce supports zero wait", async () => {
  let calls = 0;
  const d = debounce(() => calls++, 0);
  d();
  await sleep(20);
  assertEqual(calls, 1, "called");
});

// -------------------- generateId / generateUuid --------------------
test("generateId returns non-empty string", () => {
  const a = generateId();
  assert(typeof a === "string" && a.length > 0, "non-empty");
});

test("generateUuid matches v4 pattern", () => {
  const u = generateUuid();
  const re = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
  assert(re.test(u), `valid v4 uuid: ${u}`);
});

test("generateUuid produces unique values", () => {
  const set = new Set<string>();
  for (let i = 0; i < 50; i++) set.add(generateUuid());
  assertEqual(set.size, 50, "all unique");
});

// -------------------- clamp --------------------
test("clamp basic cases", () => {
  assertEqual(clamp(5, 0, 10), 5, "in range");
  assertEqual(clamp(-1, 0, 10), 0, "below min");
  assertEqual(clamp(15, 0, 10), 10, "above max");
  assertEqual(clamp(NaN, 0, 10), 0, "NaN -> min");
  assertEqual(clamp(Infinity, 0, 10), 10, "Infinity -> max");
});

// -------------------- parseFloatSafe --------------------
test("parseFloatSafe handles various inputs", () => {
  assertEqual(parseFloatSafe("123.45"), 123.45, "decimal string");
  assertEqual(parseFloatSafe("-7"), -7, "negative");
  assertEqual(parseFloatSafe("0"), 0, "zero");
  assertEqual(parseFloatSafe("abc", -1), -1, "invalid -> fallback");
  assertEqual(parseFloatSafe("", 99), 99, "empty -> fallback");
  assertEqual(parseFloatSafe(null, 5), 5, "null -> fallback");
  assertEqual(parseFloatSafe(undefined, 5), 5, "undefined -> fallback");
});

// -------------------- runner --------------------
(async () => {
  for (const t of tests) {
    console.log(`-- ${t.name}`);
    await t.fn();
  }
  console.log(`\nResults: ${passed} passed, ${failed} failed`);
  if (failed > 0) process.exit(1);
})();
