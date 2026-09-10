/**
 * Unit tests for tenant-site/lib/schema.ts (leadSchema validation).
 */
import { strict as assert } from "node:assert";
import { leadSchema } from "./schema";

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

// Valid inputs
it_("leadSchema accepts valid name/phone/email", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "0909123456", email: "alice@example.com" });
  assert.equal(r.success, true);
});
it_("leadSchema accepts empty email", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "0909123456", email: "" });
  assert.equal(r.success, true);
});
it_("leadSchema accepts missing email", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "0909123456" });
  assert.equal(r.success, true);
});
it_("leadSchema accepts 10-char phone", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "1234567890" });
  assert.equal(r.success, true);
});

// Invalid inputs
it_("leadSchema rejects name shorter than 2", () => {
  const r = leadSchema.safeParse({ name: "A", phone: "0909123456" });
  assert.equal(r.success, false);
});
it_("leadSchema rejects phone shorter than 10", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "12345" });
  assert.equal(r.success, false);
});
it_("leadSchema rejects invalid email", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "0909123456", email: "not-an-email" });
  assert.equal(r.success, false);
});
it_("leadSchema rejects missing name", () => {
  const r = leadSchema.safeParse({ phone: "0909123456" });
  assert.equal(r.success, false);
});
it_("leadSchema rejects missing phone", () => {
  const r = leadSchema.safeParse({ name: "Alice" });
  assert.equal(r.success, false);
});

// Error message verification (Vietnamese)
it_("leadSchema returns Vietnamese error for short name", () => {
  const r = leadSchema.safeParse({ name: "A", phone: "0909123456" });
  assert.equal(r.success, false);
  if (!r.success) {
    const messages = r.error.issues.map((i) => i.message).join(" | ");
    assert.match(messages, /Họ và tên/);
  }
});
it_("leadSchema returns Vietnamese error for short phone", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "12345" });
  assert.equal(r.success, false);
  if (!r.success) {
    const messages = r.error.issues.map((i) => i.message).join(" | ");
    assert.match(messages, /Số điện thoại/);
  }
});
it_("leadSchema returns Vietnamese error for invalid email", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "0909123456", email: "bad" });
  assert.equal(r.success, false);
  if (!r.success) {
    const messages = r.error.issues.map((i) => i.message).join(" | ");
    assert.match(messages, /Email/);
  }
});

// Edge cases
it_("leadSchema rejects empty name", () => {
  const r = leadSchema.safeParse({ name: "", phone: "0909123456" });
  assert.equal(r.success, false);
});
it_("leadSchema rejects empty phone", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: "" });
  assert.equal(r.success, false);
});
it_("leadSchema rejects non-string name", () => {
  const r = leadSchema.safeParse({ name: 123, phone: "0909123456" });
  assert.equal(r.success, false);
});
it_("leadSchema rejects non-string phone", () => {
  const r = leadSchema.safeParse({ name: "Alice", phone: 123456 });
  assert.equal(r.success, false);
});

console.log(`\nResults: ${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
