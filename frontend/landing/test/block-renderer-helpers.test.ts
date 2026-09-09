/**
 * BlockRenderer helper unit tests.
 *
 * Run with:
 *     npx tsx frontend/landing/test/block-renderer-helpers.test.ts
 *
 * or via the package script (added separately by CI).  These tests
 * don't require vitest/jest; they're a self-contained script using a
 * tiny home-grown assertion helper so we get coverage without adding
 * another runner to the toolchain.
 */
import {
  BLOCK_TYPE_ALIASES,
  CANONICAL_BLOCK_TYPES,
  EMPTY_OBJECT,
  resolveBlockType,
  asObject,
  isPlainObject,
} from "../components/blocks/block-helpers";

// Light-weight assertion helpers so we don't need a runner.
let failed = 0;
let passed = 0;
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

const tests: Array<{ name: string; fn: () => void }> = [];

function test(name: string, fn: () => void): void {
  tests.push({ name, fn });
}

test("CANONICAL_BLOCK_TYPES contains the expected list", () => {
  assertEqual(CANONICAL_BLOCK_TYPES.length, 12, "12 canonical types");
  assert(CANONICAL_BLOCK_TYPES.includes("hero"), "hero present");
  assert(CANONICAL_BLOCK_TYPES.includes("pricing_table"), "pricing_table present");
  assert(CANONICAL_BLOCK_TYPES.includes("risk_warning"), "risk_warning present");
  assert(CANONICAL_BLOCK_TYPES.includes("form"), "form present");
});

test("BLOCK_TYPE_ALIASES maps snake_case to canonical", () => {
  assertEqual(BLOCK_TYPE_ALIASES["feature_grid"], "features", "feature_grid");
  assertEqual(BLOCK_TYPE_ALIASES["logo_cloud"], "logos", "logo_cloud");
  assertEqual(BLOCK_TYPE_ALIASES["call_to_action"], "cta", "call_to_action");
  assertEqual(BLOCK_TYPE_ALIASES["risk_warning"], "risk_warning", "risk_warning");
  assertEqual(BLOCK_TYPE_ALIASES["pricing_table"], "pricing_table", "pricing_table");
  // Every canonical type must map to itself.
  for (const t of CANONICAL_BLOCK_TYPES) {
    assertEqual(BLOCK_TYPE_ALIASES[t], t, `self-alias ${t}`);
  }
});

test("resolveBlockType returns canonical type for known blocks", () => {
  for (const t of CANONICAL_BLOCK_TYPES) {
    assertEqual(resolveBlockType(t), t, `canonical ${t}`);
  }
  assertEqual(
    resolveBlockType("feature_grid"),
    "features",
    "feature_grid -> features",
  );
  assertEqual(
    resolveBlockType("registration_form"),
    "form",
    "registration_form -> form",
  );
  assertEqual(
    resolveBlockType("call_to_action"),
    "cta",
    "call_to_action -> cta",
  );
});

test("resolveBlockType returns null for unknown and missing input", () => {
  assertEqual(resolveBlockType(undefined), null, "undefined -> null");
  assertEqual(resolveBlockType(""), null, "empty string -> null");
  assertEqual(
    resolveBlockType("unknown_block_xyz"),
    null,
    "unknown -> null",
  );
  assertEqual(
    resolveBlockType(null as unknown as string),
    null,
    "null cast -> null",
  );
});

test("asObject normalises any block payload", () => {
  assertEqual(
    Object.keys(asObject({ a: 1 })).length,
    1,
    "plain object preserves keys",
  );
  // Non-object payloads all collapse to the shared EMPTY_OBJECT sentinel.
  assertEqual(asObject(null), EMPTY_OBJECT, "null -> EMPTY_OBJECT");
  assertEqual(asObject(undefined), EMPTY_OBJECT, "undefined -> EMPTY_OBJECT");
  assertEqual(asObject("string"), EMPTY_OBJECT, "string -> EMPTY_OBJECT");
  assertEqual(asObject(123), EMPTY_OBJECT, "number -> EMPTY_OBJECT");
  assertEqual(asObject([]), EMPTY_OBJECT, "array -> EMPTY_OBJECT");
  // Nested array children are preserved.
  const arr = asObject({ items: [1, 2, 3] });
  assert(arr.items instanceof Array, "array child preserved");
});

test("EMPTY_OBJECT is frozen so callers can't mutate it", () => {
  assert(Object.isFrozen(EMPTY_OBJECT), "frozen");
  let threw = false;
  try {
    // @ts-expect-error - mutation in non-strict context throws in strict
    (EMPTY_OBJECT as Record<string, unknown>).x = 1;
  } catch {
    threw = true;
  }
  assert(threw, "mutation throws");
});

test("isPlainObject narrows object-vs-non-object", () => {
  assert(isPlainObject({}), "object literal");
  assert(!isPlainObject(null), "null is not");
  assert(!isPlainObject(undefined), "undefined is not");
  assert(!isPlainObject([]), "array is not (even though typeof === 'object')");
  assert(!isPlainObject("foo"), "string is not");
  assert(!isPlainObject(42), "number is not");
});

// ---- runner ----
for (const t of tests) {
  console.log(`-- ${t.name}`);
  t.fn();
}
console.log(`\nResults: ${passed} passed, ${failed} failed`);
if (failed > 0) {
  process.exit(1);
}
