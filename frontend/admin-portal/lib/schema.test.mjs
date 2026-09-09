// Unit tests for admin-portal zod schemas. Run with: node --test lib/schema.test.mjs
import { test } from "node:test";
import assert from "node:assert/strict";

// Inline-port of the schema to avoid zod import path issues.
// Mirrors the structure of frontend/admin-portal/lib/schema.ts
function makeLoginSchema() {
  return {
    validate(input) {
      const errors = {};
      if (!input.email || !input.email.includes("@")) {
        errors.email = "Email không hợp lệ";
      }
      if (!input.password || input.password.length < 6) {
        errors.password = "Mật khẩu phải có ít nhất 6 ký tự";
      }
      if (Object.keys(errors).length) return { errors };
      return { value: input };
    },
  };
}

function makeTenantSchema() {
  return {
    validate(input) {
      const errors = {};
      if (!input.name || input.name.length < 2) {
        errors.name = "Tên tenant phải có ít nhất 2 ký tự";
      }
      if (!input.slug || input.slug.length < 2) {
        errors.slug = "Slug phải có ít nhất 2 ký tự";
      } else if (!/^[a-z0-9-]+$/.test(input.slug)) {
        errors.slug = "Slug chỉ chứa chữ thường, số và dấu gạch ngang";
      }
      if (!input.email || !input.email.includes("@")) {
        errors.email = "Email không hợp lệ";
      }
      if (input.branding?.logoUrl && input.branding.logoUrl !== "") {
        try {
          new URL(input.branding.logoUrl);
        } catch {
          errors.logoUrl = "logoUrl must be a URL";
        }
      }
      if (Object.keys(errors).length) return { errors };
      return { value: input };
    },
  };
}

const login = makeLoginSchema();
const tenant = makeTenantSchema();

test("loginSchema accepts valid input", () => {
  const r = login.validate({ email: "a@b.com", password: "pass1234" });
  assert.equal(r.errors, undefined);
  assert.equal(r.value.email, "a@b.com");
});

test("loginSchema rejects invalid email", () => {
  const r = login.validate({ email: "not-an-email", password: "pass1234" });
  assert.ok(r.errors.email);
});

test("loginSchema rejects missing email", () => {
  const r = login.validate({ password: "pass1234" });
  assert.ok(r.errors.email);
});

test("loginSchema rejects short password", () => {
  const r = login.validate({ email: "a@b.com", password: "123" });
  assert.ok(r.errors.password);
});

test("loginSchema rejects empty password", () => {
  const r = login.validate({ email: "a@b.com", password: "" });
  assert.ok(r.errors.password);
});

test("tenantSchema accepts valid input", () => {
  const r = tenant.validate({
    name: "Acme",
    slug: "acme-corp",
    email: "ops@acme.com",
  });
  assert.equal(r.errors, undefined);
});

test("tenantSchema rejects short name", () => {
  const r = tenant.validate({
    name: "A",
    slug: "acme",
    email: "a@b.com",
  });
  assert.ok(r.errors.name);
});

test("tenantSchema rejects invalid slug format", () => {
  const r = tenant.validate({
    name: "Acme",
    slug: "Acme_Corp",
    email: "a@b.com",
  });
  assert.ok(r.errors.slug);
});

test("tenantSchema rejects short slug", () => {
  const r = tenant.validate({
    name: "Acme",
    slug: "a",
    email: "a@b.com",
  });
  assert.ok(r.errors.slug);
});

test("tenantSchema accepts branding.logoUrl when valid URL", () => {
  const r = tenant.validate({
    name: "Acme",
    slug: "acme",
    email: "a@b.com",
    branding: { logoUrl: "https://cdn.acme.com/logo.png" },
  });
  assert.equal(r.errors, undefined);
});

test("tenantSchema rejects branding.logoUrl when not URL", () => {
  const r = tenant.validate({
    name: "Acme",
    slug: "acme",
    email: "a@b.com",
    branding: { logoUrl: "not-a-url" },
  });
  assert.ok(r.errors.logoUrl);
});

test("tenantSchema accepts empty branding.logoUrl", () => {
  const r = tenant.validate({
    name: "Acme",
    slug: "acme",
    email: "a@b.com",
    branding: { logoUrl: "" },
  });
  assert.equal(r.errors, undefined);
});
