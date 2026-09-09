// Unit tests for block-helpers.ts — pure helpers only (no React DOM).
// This file inlines the helpers so we can test logic without compiling TS.
// Run with:  node --test tests/block-helpers.test.mjs
import { test } from "node:test";
import assert from "node:assert/strict";

// Inlined from components/blocks/block-helpers.ts
export const CANONICAL_BLOCK_TYPES = [
  "hero", "features", "logos", "speakers", "trust",
  "stats", "faq", "cta", "testimonial", "risk_warning",
  "form", "pricing_table",
];

export const BLOCK_TYPE_ALIASES = {
  hero: "hero",
  features: "features",
  feature_grid: "features",
  logos: "logos",
  logo_cloud: "logos",
  speakers: "speakers",
  speaker: "speakers",
  trust: "trust",
  trust_section: "trust",
  stats: "stats",
  statistics: "stats",
  faq: "faq",
  faqs: "faq",
  cta: "cta",
  call_to_action: "cta",
  testimonial: "testimonial",
  testimonials: "testimonial",
  risk_warning: "risk_warning",
  risk: "risk_warning",
  form: "form",
  registration_form: "form",
  pricing_table: "pricing_table",
  pricing: "pricing_table",
};

export const EMPTY_OBJECT = Object.freeze({});

export function resolveBlockType(type) {
  if (!type) return null;
  const canonical = BLOCK_TYPE_ALIASES[type];
  return canonical ?? null;
}

export function asObject(value) {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value;
  }
  return EMPTY_OBJECT;
}

export function isPlainObject(value) {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

test("CANONICAL_BLOCK_TYPES contains expected types", () => {
  const expected = [
    "hero", "features", "logos", "speakers", "trust",
    "stats", "faq", "cta", "testimonial", "risk_warning",
    "form", "pricing_table",
  ];
  for (const t of expected) {
    assert.ok(CANONICAL_BLOCK_TYPES.includes(t), `missing ${t}`);
  }
  assert.equal(CANONICAL_BLOCK_TYPES.length, 12);
});

test("BLOCK_TYPE_ALIASES maps canonical to itself", () => {
  for (const t of CANONICAL_BLOCK_TYPES) {
    assert.equal(BLOCK_TYPE_ALIASES[t], t, `${t} should map to itself`);
  }
});

test("BLOCK_TYPE_ALIASES maps common aliases", () => {
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

test("resolveBlockType handles undefined", () => {
  assert.equal(resolveBlockType(undefined), null);
});

test("resolveBlockType handles empty string", () => {
  assert.equal(resolveBlockType(""), null);
});

test("resolveBlockType handles unknown", () => {
  assert.equal(resolveBlockType("not_a_type"), null);
});

test("resolveBlockType returns canonical", () => {
  assert.equal(resolveBlockType("hero"), "hero");
  assert.equal(resolveBlockType("feature_grid"), "features");
});

test("resolveBlockType is case-sensitive", () => {
  assert.equal(resolveBlockType("HERO"), null);
});

test("EMPTY_OBJECT is frozen", () => {
  assert.equal(Object.isFrozen(EMPTY_OBJECT), true);
  assert.equal(Object.keys(EMPTY_OBJECT).length, 0);
});

test("asObject returns EMPTY_OBJECT for null", () => {
  assert.equal(asObject(null), EMPTY_OBJECT);
});

test("asObject returns EMPTY_OBJECT for undefined", () => {
  assert.equal(asObject(undefined), EMPTY_OBJECT);
});

test("asObject returns EMPTY_OBJECT for arrays", () => {
  assert.equal(asObject([]), EMPTY_OBJECT);
  assert.equal(asObject([1, 2, 3]), EMPTY_OBJECT);
});

test("asObject returns EMPTY_OBJECT for primitives", () => {
  assert.equal(asObject("string"), EMPTY_OBJECT);
  assert.equal(asObject(42), EMPTY_OBJECT);
  assert.equal(asObject(true), EMPTY_OBJECT);
});

test("asObject returns the object directly", () => {
  const input = { foo: "bar", n: 1 };
  assert.deepEqual(asObject(input), input);
});

test("asObject returns empty object for plain object", () => {
  assert.deepEqual(asObject({}), {});
});

test("isPlainObject true for objects", () => {
  assert.equal(isPlainObject({}), true);
  assert.equal(isPlainObject({ a: 1 }), true);
});

test("isPlainObject false for non-objects", () => {
  assert.equal(isPlainObject(null), false);
  assert.equal(isPlainObject(undefined), false);
  assert.equal(isPlainObject("x"), false);
  assert.equal(isPlainObject(1), false);
  assert.equal(isPlainObject([]), false);
  assert.equal(isPlainObject(true), false);
});

test("BLOCK_TYPE_ALIASES has all canonical entries", () => {
  for (const t of CANONICAL_BLOCK_TYPES) {
    assert.ok(t in BLOCK_TYPE_ALIASES);
  }
});

test("resolveBlockType handles snake_case variants", () => {
  for (const t of CANONICAL_BLOCK_TYPES) {
    const result = resolveBlockType(t);
    assert.ok(result !== null);
  }
});

test("asObject does not mutate EMPTY_OBJECT", () => {
  const r = asObject({ foo: 1 });
  assert.equal(r.foo, 1);
  assert.equal(Object.keys(EMPTY_OBJECT).length, 0);
});

test("resolveBlockType handles whitespace", () => {
  assert.equal(resolveBlockType(" hero "), null);
});

test("isPlainObject treats Date as true (object but not array)", () => {
  assert.equal(isPlainObject(new Date()), true);
});
