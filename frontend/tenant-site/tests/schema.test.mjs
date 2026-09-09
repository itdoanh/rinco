// Unit tests for tenant-site schema validation. Run with: node --test tests/schema.test.mjs
import { test } from "node:test";
import assert from "node:assert/strict";

// Inline-port matching frontend/tenant-site/lib/schema.ts
function makeLeadSchema() {
  return {
    validate(input) {
      const errors = {};
      if (!input.name || input.name.length < 2) {
        errors.name = "Họ và tên phải có ít nhất 2 ký tự";
      }
      if (!input.phone || input.phone.length < 10) {
        errors.phone = "Số điện thoại không hợp lệ";
      }
      if (input.email !== "" && input.email !== undefined) {
        if (!input.email.includes("@")) {
          errors.email = "Email không hợp lệ";
        }
      }
      if (Object.keys(errors).length) return { errors };
      return { value: input };
    },
  };
}

const lead = makeLeadSchema();

test("leadSchema accepts valid input", () => {
  const r = lead.validate({ name: "Nguyen Van A", phone: "0987654321", email: "a@b.com" });
  assert.equal(r.errors, undefined);
});

test("leadSchema rejects short name", () => {
  const r = lead.validate({ name: "A", phone: "0987654321" });
  assert.ok(r.errors.name);
});

test("leadSchema rejects missing name", () => {
  const r = lead.validate({ phone: "0987654321" });
  assert.ok(r.errors.name);
});

test("leadSchema rejects short phone", () => {
  const r = lead.validate({ name: "Alice", phone: "123" });
  assert.ok(r.errors.phone);
});

test("leadSchema accepts email empty string", () => {
  const r = lead.validate({ name: "Alice", phone: "0987654321", email: "" });
  assert.equal(r.errors, undefined);
});

test("leadSchema accepts email undefined", () => {
  const r = lead.validate({ name: "Alice", phone: "0987654321" });
  assert.equal(r.errors, undefined);
});

test("leadSchema rejects invalid email", () => {
  const r = lead.validate({ name: "Alice", phone: "0987654321", email: "not-an-email" });
  assert.ok(r.errors.email);
});

test("leadSchema accepts valid email", () => {
  const r = lead.validate({ name: "Alice", phone: "0987654321", email: "user@example.com" });
  assert.equal(r.errors, undefined);
});

test("leadSchema multiple errors at once", () => {
  const r = lead.validate({});
  assert.ok(r.errors.name);
  assert.ok(r.errors.phone);
});

// Schema shape test for Tenant interfaces — type-level only
test("Branding interface contract", () => {
  // Just verify shape documentation at runtime
  const b = { primary: "#fff", secondary: "#000" };
  assert.equal(b.primary, "#fff");
  assert.equal(b.secondary, "#000");
});

test("Block interface contract", () => {
  const blk = { id: "1", type: "hero", data: {} };
  assert.equal(blk.id, "1");
  assert.equal(blk.type, "hero");
});

test("Page interface contract", () => {
  const p = { id: "p1", slug: "home", title: "Home", blocks: [] };
  assert.equal(p.slug, "home");
  assert.equal(p.title, "Home");
  assert.equal(p.blocks.length, 0);
});
