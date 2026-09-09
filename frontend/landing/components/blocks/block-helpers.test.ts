/**
 * Unit tests for ``block-helpers.ts`` (pure logic, no React rendering).
 */
import { describe, expect, it } from "@jest/globals";
import {
  BLOCK_TYPE_ALIASES,
  CANONICAL_BLOCK_TYPES,
  asObject,
  isPlainObject,
  resolveBlockType,
  EMPTY_OBJECT,
} from "@/components/blocks/block-helpers";

describe("CANONICAL_BLOCK_TYPES", () => {
  it("contains all expected canonical block types", () => {
    expect(CANONICAL_BLOCK_TYPES).toContain("hero");
    expect(CANONICAL_BLOCK_TYPES).toContain("features");
    expect(CANONICAL_BLOCK_TYPES).toContain("logos");
    expect(CANONICAL_BLOCK_TYPES).toContain("speakers");
    expect(CANONICAL_BLOCK_TYPES).toContain("trust");
    expect(CANONICAL_BLOCK_TYPES).toContain("stats");
    expect(CANONICAL_BLOCK_TYPES).toContain("faq");
    expect(CANONICAL_BLOCK_TYPES).toContain("cta");
    expect(CANONICAL_BLOCK_TYPES).toContain("testimonial");
    expect(CANONICAL_BLOCK_TYPES).toContain("risk_warning");
    expect(CANONICAL_BLOCK_TYPES).toContain("form");
    expect(CANONICAL_BLOCK_TYPES).toContain("pricing_table");
  });

  it("has 12 canonical types", () => {
    expect(CANONICAL_BLOCK_TYPES.length).toBe(12);
  });

  it("all entries are unique", () => {
    const set = new Set(CANONICAL_BLOCK_TYPES);
    expect(set.size).toBe(CANONICAL_BLOCK_TYPES.length);
  });
});

describe("BLOCK_TYPE_ALIASES", () => {
  it("maps snake_case aliases to canonical types", () => {
    expect(BLOCK_TYPE_ALIASES.feature_grid).toBe("features");
    expect(BLOCK_TYPE_ALIASES.logo_cloud).toBe("logos");
    expect(BLOCK_TYPE_ALIASES.speaker).toBe("speakers");
    expect(BLOCK_TYPE_ALIASES.trust_section).toBe("trust");
    expect(BLOCK_TYPE_ALIASES.statistics).toBe("stats");
    expect(BLOCK_TYPE_ALIASES.faqs).toBe("faq");
    expect(BLOCK_TYPE_ALIASES.call_to_action).toBe("cta");
    expect(BLOCK_TYPE_ALIASES.testimonials).toBe("testimonial");
    expect(BLOCK_TYPE_ALIASES.risk).toBe("risk_warning");
    expect(BLOCK_TYPE_ALIASES.registration_form).toBe("form");
    expect(BLOCK_TYPE_ALIASES.pricing).toBe("pricing_table");
  });

  it("preserves identity for canonical types", () => {
    for (const t of CANONICAL_BLOCK_TYPES) {
      expect(BLOCK_TYPE_ALIASES[t]).toBe(t);
    }
  });
});

describe("resolveBlockType", () => {
  it("returns the canonical type for canonical input", () => {
    expect(resolveBlockType("hero")).toBe("hero");
    expect(resolveBlockType("features")).toBe("features");
  });

  it("returns the canonical type for alias input", () => {
    expect(resolveBlockType("feature_grid")).toBe("features");
    expect(resolveBlockType("logo_cloud")).toBe("logos");
    expect(resolveBlockType("call_to_action")).toBe("cta");
  });

  it("returns null for unknown input", () => {
    expect(resolveBlockType("unknown_type")).toBeNull();
    expect(resolveBlockType("not_a_block")).toBeNull();
  });

  it("returns null for undefined input", () => {
    expect(resolveBlockType(undefined)).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(resolveBlockType("")).toBeNull();
  });

  it("is case sensitive", () => {
    expect(resolveBlockType("HERO")).toBeNull();
    expect(resolveBlockType("Hero")).toBeNull();
  });
});

describe("asObject", () => {
  it("returns the value when it is a plain object", () => {
    const obj = { a: 1, b: "two" };
    expect(asObject(obj)).toBe(obj);
  });

  it("returns the value when it is an empty object", () => {
    const obj = {};
    expect(asObject(obj)).toBe(obj);
  });

  it("returns EMPTY_OBJECT for null", () => {
    expect(asObject(null)).toBe(EMPTY_OBJECT);
  });

  it("returns EMPTY_OBJECT for undefined", () => {
    expect(asObject(undefined)).toBe(EMPTY_OBJECT);
  });

  it("returns EMPTY_OBJECT for primitives", () => {
    expect(asObject(42)).toBe(EMPTY_OBJECT);
    expect(asObject("string")).toBe(EMPTY_OBJECT);
    expect(asObject(true)).toBe(EMPTY_OBJECT);
  });

  it("returns EMPTY_OBJECT for arrays", () => {
    expect(asObject([1, 2, 3])).toBe(EMPTY_OBJECT);
    expect(asObject([])).toBe(EMPTY_OBJECT);
  });

  it("returns the value for nested objects", () => {
    const obj = { a: { b: { c: 1 } } };
    expect(asObject(obj)).toBe(obj);
  });
});

describe("isPlainObject", () => {
  it("returns true for plain objects", () => {
    expect(isPlainObject({})).toBe(true);
    expect(isPlainObject({ a: 1 })).toBe(true);
  });

  it("returns false for null", () => {
    expect(isPlainObject(null)).toBe(false);
  });

  it("returns false for undefined", () => {
    expect(isPlainObject(undefined)).toBe(false);
  });

  it("returns false for primitives", () => {
    expect(isPlainObject(42)).toBe(false);
    expect(isPlainObject("string")).toBe(false);
    expect(isPlainObject(true)).toBe(false);
  });

  it("returns false for arrays", () => {
    expect(isPlainObject([])).toBe(false);
    expect(isPlainObject([1, 2])).toBe(false);
  });
});

describe("EMPTY_OBJECT", () => {
  it("is frozen", () => {
    expect(Object.isFrozen(EMPTY_OBJECT)).toBe(true);
  });

  it("has no keys", () => {
    expect(Object.keys(EMPTY_OBJECT)).toEqual([]);
  });
});
