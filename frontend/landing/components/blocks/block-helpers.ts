/**
 * Pure helpers used by ``BlockRenderer.tsx`` to dispatch JSON-shape
 * landing-page blocks to their respective React components.
 *
 * Kept in a ``.ts`` file (not ``.tsx``) so we can unit test them
 * without pulling in React DOM.  See ``block-renderer-helpers.test.ts``.
 */

/**
 * Canonical list of block types.  Anything not in this set is rejected
 * by ``resolveBlockType`` so unknown blocks never reach a React
 * component.
 */
export const CANONICAL_BLOCK_TYPES = [
  "hero",
  "features",
  "logos",
  "speakers",
  "trust",
  "stats",
  "faq",
  "cta",
  "testimonial",
  "risk_warning",
  "form",
  "pricing_table",
] as const;

export type CanonicalBlockType = (typeof CANONICAL_BLOCK_TYPES)[number];

/**
 * Maps any of the historical aliases (snake_case vs plural) to its
 * canonical block type.  Add new aliases here to support historical
 * block-type names coming back from the marketing backend.
 */
export const BLOCK_TYPE_ALIASES: Record<string, CanonicalBlockType> = {
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

/**
 * Convert an arbitrary block payload to a canonical type (or ``null``
 * when the type is unknown so the caller can decide whether to log a
 * warning or hide the block).
 */
export function resolveBlockType(type: string | undefined): CanonicalBlockType | null {
  if (!type) return null;
  const canonical = BLOCK_TYPE_ALIASES[type];
  return canonical ?? null;
}

/**
 * Sentinel object returned by :func:`asObject` for non-object inputs.
 * Consumers can compare with ``=== EMPTY_OBJECT`` to detect that the
 * payload was not an object.
 */
export const EMPTY_OBJECT: Readonly<Record<string, unknown>> = Object.freeze({});

/**
 * Narrow ``unknown`` block payload to ``Record<string, unknown>`` so
 * downstream components can read keys safely.  Returns
 * :data:`EMPTY_OBJECT` for non-object payloads (not ``null`` or
 * ``undefined``) so consumers can use the result without null-checks.
 */
export function asObject(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return EMPTY_OBJECT as Record<string, unknown>;
}

/** Convenience predicate: is the given value a non-null object? */
export function isPlainObject(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}
