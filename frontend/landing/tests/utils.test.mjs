// Tests for landing/lib/utils.ts - inlined for pure Node test runner compatibility.

let passed = 0;
let failed = 0;

function assert(cond, label) {
  if (cond) passed++;
  else {
    failed++;
    console.error("FAIL " + label);
  }
}

function assertEqual(actual, expected, label) {
  if (actual === expected) passed++;
  else {
    failed++;
    console.error("FAIL " + label + ": got " + JSON.stringify(actual) + " want " + JSON.stringify(expected));
  }
}

function formatNumber(num) {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K';
  }
  return num.toString();
}

function formatPercentage(value, decimals = 1) {
  return value.toFixed(decimals) + '%';
}

function slugify(text) {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '');
}

function truncate(text, maxLength) {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength).trim() + '...';
}

function parseQueryParams(search) {
  const params = new URLSearchParams(search);
  const result = {};
  params.forEach((value, key) => {
    result[key] = value;
  });
  return result;
}

// ===== formatNumber =====
assertEqual(formatNumber(0), '0', 'formatNumber 0');
assertEqual(formatNumber(500), '500', 'formatNumber 500');
assertEqual(formatNumber(1000), '1.0K', 'formatNumber 1000');
assertEqual(formatNumber(1500), '1.5K', 'formatNumber 1500');
assertEqual(formatNumber(999999), '1000.0K', 'formatNumber 999999');
assertEqual(formatNumber(1000000), '1.0M', 'formatNumber 1000000');
assertEqual(formatNumber(2500000), '2.5M', 'formatNumber 2500000');

// ===== formatPercentage =====
assertEqual(formatPercentage(50), '50.0%', 'formatPercentage 50');
assertEqual(formatPercentage(50, 0), '50%', 'formatPercentage 50 0 decimals');
assertEqual(formatPercentage(33.333, 2), '33.33%', 'formatPercentage 33.333 2 decimals');
assertEqual(formatPercentage(0), '0.0%', 'formatPercentage 0');
assertEqual(formatPercentage(100), '100.0%', 'formatPercentage 100');

// ===== slugify =====
assertEqual(slugify('Hello World'), 'hello-world', 'slugify basic');
assertEqual(slugify('Tiếng Việt có dấu'), 'tieng-viet-co-dau', 'slugify vietnamese');
assertEqual(slugify('  Multiple   Spaces  '), 'multiple-spaces', 'slugify spaces');
assertEqual(slugify('!!! Special !!! Chars !!!'), 'special-chars', 'slugify special');
assertEqual(slugify('---leading-and-trailing---'), 'leading-and-trailing', 'slugify trim dashes');
assertEqual(slugify('ABC123'), 'abc123', 'slugify alphanumeric');

// ===== truncate =====
assertEqual(truncate('short', 100), 'short', 'truncate short');
assertEqual(truncate('a'.repeat(200), 50), 'a'.repeat(50).trim() + '...', 'truncate long');
assertEqual(truncate('hello world', 5), 'hello...', 'truncate to 5');
assertEqual(truncate('exact', 5), 'exact', 'truncate exact length');

// ===== parseQueryParams =====
const empty = parseQueryParams('');
assertEqual(Object.keys(empty).length, 0, 'parseQueryParams empty');

const one = parseQueryParams('?foo=bar');
assertEqual(one.foo, 'bar', 'parseQueryParams one param');

const two = parseQueryParams('?a=1&b=2');
assertEqual(two.a, '1', 'parseQueryParams a=1');
assertEqual(two.b, '2', 'parseQueryParams b=2');

const encoded = parseQueryParams('?email=hello%40example.com');
assertEqual(encoded.email, 'hello@example.com', 'parseQueryParams url decode');

const noq = parseQueryParams('foo=bar&baz=qux');
assertEqual(noq.foo, 'bar', 'parseQueryParams no leading ?');

console.log("\nResults: " + passed + " passed, " + failed + " failed");
if (failed > 0) process.exit(1);
