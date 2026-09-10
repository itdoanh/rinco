/**
 * Unit tests for ``block-helpers.ts`` (pure logic, no React rendering).
 * Runs via plain `tsx` + `node:assert`.
 */
import { strict as assert } from "node:assert";
import {
  BLOCK_TYPE_ALIASES,
  CANONICAL_BLOCK_TYPES,
  asObject,
  isPlainObject,
  resolveBlockType,
  EMPTY_OBJECT,
} from "@/components/blocks/block-helpers";

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

// CANONICAL_BLOCK_TYPES
it_("CANONICAL_BLOCK_TYPES contains all expected types", () => {
  for (const t of ["hero","features","logos","speakers","trust","stats","faq","cta","testimonial","risk_warning","form","pricing_table"]) {
    assert.ok(CANONICAL_BLOCK_TYPES.includes(t), `missing ${t}`);
  }
});
it_("CANONICAL_BLOCK_TYPES has 12 entries", () => {
  assert.equal(CANONICAL_BLOCK_TYPES.length, 12);
});
it_("CANONICAL_BLOCK_TYPES entries are unique", () => {
  assert.equal(new Set(CANONICAL_BLOCK_TYPES).size, CANONICAL_BLOCK_TYPES.length);
});

// BLOCK_TYPE_ALIASES
it_("BLOCK_TYPE_ALIASES maps snake_case aliases to canonical types", () => {
  assert.equal(BLOCK_TYPE_ALIASES.feature_grid, "features");
  assert.equal(BLOCK_TYPE_ALIASES.logo_cloud, "logos");
  assert.equal(BLOCK_TYPE_ALIASES.speaker, "speakers");
  assert.equal(BLOCK_TYPE_ALIASES.trust_section, "trust");
  assert.equal(BLOCK_TYPE_ALIASES.statistics, "stats");
  assert.equal(BLOCK_TYPE_ALIASES.faqs, "faq");
  assert.equal(BLOCK_TYPE_ALIASES.call_to_action, "cta");
  assert.equal(BLOCK_TYPE_ALIASES.testimonials, "testimonial");
  assert.equal(BLOCK_TYPE_ALIASES.risk, "risk_warning");
  assert.equal(BLOCK_TYPE_ALIASES.registration_form, "form");
  assert.equal(BLOCK_TYPE_ALIASES.pricing, "pricing_table");
});
it_("BLOCK_TYPE_ALIASES preserves identity for canonical types", () => {
  for (const t of CANONICAL_BLOCK_TYPES) {
    assert.equal(BLOCK_TYPE_ALIASES[t], t);
  }
});

// resolveBlockType
it_("resolveBlockType returns canonical type for canonical input", () => {
  assert.equal(resolveBlockType("hero"), "hero");
  assert.equal(resolveBlockType("features"), "features");
});
it_("resolveBlockType returns canonical type for alias input", () => {
  assert.equal(resolveBlockType("feature_grid"), "features");
  assert.equal(resolveBlockType("logo_cloud"), "logos");
  assert.equal(resolveBlockType("call_to_action"), "cta");
});
it_("resolveBlockType returns null for unknown input", () => {
  assert.equal(resolveBlockType("unknown_type"), null);
  assert.equal(resolveBlockType("not_a_block"), null);
});
it_("resolveBlockType returns null for undefined", () => {
  assert.equal(resolveBlockType(undefined), null);
});
it_("resolveBlockType returns null for empty string", () => {
  assert.equal(resolveBlockType(""), null);
});
it_("resolveBlockType is case sensitive", () => {
  assert.equal(resolveBlockType("HERO"), null);
  assert.equal(resolveBlockType("Hero"), null);
});

// asObject
it_("asObject returns the value when it is a plain object", () => {
  const obj = { a: 1, b: "two" };
  assert.equal(asObject(obj), obj);
});
it_("asObject returns the value for empty object", () => {
  const obj = {};
  assert.equal(asObject(obj), obj);
});
it_("asObject returns EMPTY_OBJECT for null", () => {
  assert.equal(asObject(null), EMPTY_OBJECT);
});
it_("asObject returns EMPTY_OBJECT for undefined", () => {
  assert.equal(asObject(undefined), EMPTY_OBJECT);
});
it_("asObject returns EMPTY_OBJECT for primitives", () => {
  assert.equal(asObject(42), EMPTY_OBJECT);
  assert.equal(asObject("string"), EMPTY_OBJECT);
  assert.equal(asObject(true), EMPTY_OBJECT);
});
it_("asObject returns EMPTY_OBJECT for arrays", () => {
  assert.equal(asObject([1, 2, 3]), EMPTY_OBJECT);
  assert.equal(asObject([]), EMPTY_OBJECT);
});
it_("asObject returns the value for nested objects", () => {
  const obj = { a: { b: { c: 1 } } };
  assert.equal(asObject(obj), obj);
});

// isPlainObject
it_("isPlainObject returns true for plain objects", () => {
  assert.equal(isPlainObject({}), true);
  assert.equal(isPlainObject({ a: 1 }), true);
});
it_("isPlainObject returns false for null", () => {
  assert.equal(isPlainObject(null), false);
});
it_("isPlainObject returns false for undefined", () => {
  assert.equal(isPlainObject(undefined), false);
});
it_("isPlainObject returns false for primitives", () => {
  assert.equal(isPlainObject(42), false);
  assert.equal(isPlainObject("string"), false);
  assert.equal(isPlainObject(true), false);
});
it_("isPlainObject returns false for arrays", () => {
  assert.equal(isPlainObject([]), false);
  assert.equal(isPlainObject([1, 2]), false);
});

// EMPTY_OBJECT
it_("EMPTY_OBJECT is frozen", () => {
  assert.equal(Object.isFrozen(EMPTY_OBJECT), true);
});
it_("EMPTY_OBJECT has no keys", () => {
  assert.equal(Object.keys(EMPTY_OBJECT).length, 0);
});

console.log(`\nResults: ${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
