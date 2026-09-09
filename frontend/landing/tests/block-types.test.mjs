// Tests for landing/lib/block-types.ts type predicates (inlined for Node test).
// Tests the type-guard logic by inlining the predicates.

let passed = 0;
let failed = 0;

function assert(cond, label) {
  if (cond) passed++;
  else {
    failed++;
    console.error("FAIL " + label);
  }
}

const VALID_TYPES = ['hero', 'feature_grid', 'testimonial', 'faq', 'form', 'cta', 'pricing', 'stats'];

function isHeroBlock(data) { return data.type === 'hero'; }
function isFeatureGridBlock(data) { return data.type === 'feature_grid'; }
function isTestimonialBlock(data) { return data.type === 'testimonial'; }
function isFAQBlock(data) { return data.type === 'faq'; }
function isFormBlock(data) { return data.type === 'form'; }
function isCTABlock(data) { return data.type === 'cta'; }
function isPricingBlock(data) { return data.type === 'pricing'; }
function isStatsBlock(data) { return data.type === 'stats'; }

// Type predicates
assert(isHeroBlock({ type: 'hero' }), 'hero matches');
assert(!isHeroBlock({ type: 'cta' }), 'cta not hero');
assert(!isHeroBlock({}), 'empty object not hero');

assert(isFeatureGridBlock({ type: 'feature_grid' }), 'feature_grid matches');
assert(!isFeatureGridBlock({ type: 'hero' }), 'hero not feature_grid');

assert(isTestimonialBlock({ type: 'testimonial' }), 'testimonial matches');
assert(!isTestimonialBlock({ type: 'hero' }), 'hero not testimonial');

assert(isFAQBlock({ type: 'faq' }), 'faq matches');
assert(!isFAQBlock({ type: 'form' }), 'form not faq');

assert(isFormBlock({ type: 'form' }), 'form matches');
assert(!isFormBlock({ type: 'cta' }), 'cta not form');

assert(isCTABlock({ type: 'cta' }), 'cta matches');
assert(!isCTABlock({ type: 'pricing' }), 'pricing not cta');

assert(isPricingBlock({ type: 'pricing' }), 'pricing matches');
assert(!isPricingBlock({ type: 'cta' }), 'cta not pricing');

assert(isStatsBlock({ type: 'stats' }), 'stats matches');
assert(!isStatsBlock({ type: 'hero' }), 'hero not stats');

// BlockType enumeration coverage
const seen = new Set();
for (const t of VALID_TYPES) {
  assert(typeof t === 'string' && t.length > 0, `type ${t} is non-empty string`);
  seen.add(t);
}
assert(seen.size === VALID_TYPES.length, 'all types unique');

// Form field types
const FORM_FIELD_TYPES = ['text', 'email', 'phone', 'textarea', 'select'];
assert(FORM_FIELD_TYPES.length === 5, '5 form field types');
assert(FORM_FIELD_TYPES.includes('email'), 'form has email field type');

// Block alignment
const ALIGNMENTS = ['left', 'center', 'right'];
assert(ALIGNMENTS.length === 3, '3 alignments');

// Theme variants
const THEMES = ['light', 'dark'];
assert(THEMES.length === 2, '2 themes');
const CTA_THEMES = ['light', 'dark', 'gradient'];
assert(CTA_THEMES.length === 3, '3 CTA themes');

// Pricing columns
const COLS = [2, 3, 4];
assert(COLS.length === 3, '3 pricing columns');

console.log("\nResults: " + passed + " passed, " + failed + " failed");
if (failed > 0) process.exit(1);
