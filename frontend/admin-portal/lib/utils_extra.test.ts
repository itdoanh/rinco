/**
 * Unit tests for admin-portal/lib/utils.ts helpers.
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
} from "../lib/utils"

let passed = 0
let failed = 0

function assert(cond: unknown, label: string): void {
  if (cond) {
    passed++
  } else {
    failed++
    console.error(`FAIL ${label}`)
  }
}

function assertEqual<T>(actual: T, expected: T, label: string): void {
  if (actual === expected) {
    passed++
  } else {
    failed++
    console.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`)
  }
}

const tests: Array<{ name: string; fn: () => void | Promise<void> }> = []
function test(name: string, fn: () => void | Promise<void>): void {
  tests.push({ name, fn })
}

// cn
test("cn merges classes", () => {
  assertEqual(cn("a", "b"), "a b", "concatenates")
})
test("cn resolves tailwind conflicts", () => {
  assertEqual(cn("px-2", "px-4"), "px-4", "later wins")
})
test("cn ignores falsy", () => {
  assertEqual(cn("a", false, null, undefined, "b"), "a b", "filters falsy")
})

// formatCurrency
test("formatCurrency renders VND", () => {
  const out = formatCurrency(1000)
  assert(typeof out === "string" && out.length > 0, "non-empty")
})
test("formatCurrency USD", () => {
  const out = formatCurrency(1000, "USD")
  assert(typeof out === "string", "USD string")
  assert(out.includes("$") || out.includes("US"), "USD symbol")
})

// formatNumber
test("formatNumber adds separators", () => {
  const out = formatNumber(1234567)
  assert(out.includes("."), "separator")
})

// formatDate
test("formatDate from Date object", () => {
  const d = new Date(2024, 0, 5)
  const out = formatDate(d)
  assert(out.length > 0, "non-empty")
})
test("formatDate from string", () => {
  const out = formatDate("2024-01-05")
  assert(out.length > 0, "string ok")
})

// formatDateTime
test("formatDateTime renders time", () => {
  const d = new Date(2024, 0, 5, 9, 5)
  const out = formatDateTime(d)
  assert(out.length > 0, "non-empty")
})

// slugify
test("slugify basics", () => {
  assertEqual(slugify("Hello World"), "hello-world", "ascii")
})
test("slugify strips diacritics", () => {
  assertEqual(slugify("Tiếng Việt"), "tieng-viet", "vietnamese")
})
test("slugify collapses spaces", () => {
  assertEqual(slugify("a    b"), "a-b", "collapse")
})
test("slugify strips non-alphanumeric", () => {
  assertEqual(slugify("a!@#$b"), "a-b", "strip symbols")
})

// truncate
test("truncate short", () => {
  assertEqual(truncate("hi", 10), "hi", "no truncate")
})
test("truncate long", () => {
  const out = truncate("hello world", 5)
  assert(out.length <= 8, "truncated")
  assert(out.startsWith("hello"), "has prefix")
})

// getInitials
test("getInitials two words", () => {
  assertEqual(getInitials("John Doe"), "JD", "JD")
})
test("getInitials single word", () => {
  assertEqual(getInitials("Alice"), "A", "A")
})
test("getInitials three words", () => {
  assertEqual(getInitials("Mary Jane Smith"), "MJ", "first two")
})

// sleep
test("sleep delays", async () => {
  const start = Date.now()
  await sleep(30)
  assert(Date.now() - start >= 25, "delayed")
})

// debounce
test("debounce coalesces", async () => {
  let calls = 0
  const fn = () => calls++
  const d = debounce(fn, 20)
  d()
  d()
  d()
  await sleep(50)
  assertEqual(calls, 1, "one call")
})

// generateId
test("generateId non-empty", () => {
  const id = generateId()
  assert(typeof id === "string" && id.length > 0, "non-empty")
})
test("generateId unique", () => {
  const a = generateId()
  const b = generateId()
  assert(a !== b, "different")
})

;(async () => {
  for (const t of tests) {
    console.log(`-- ${t.name}`)
    await t.fn()
  }
  console.log(`\nResults: ${passed} passed, ${failed} failed`)
  if (failed > 0) process.exit(1)
})()
