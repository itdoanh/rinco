import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * RINCO Landing Multi-Domain Routing Middleware
 *
 * Resolves the tenant slug for landing-page routes that follow
 * the `/{tenant}/{page}` pattern.
 *
 * Three routing strategies (chosen by reverse-proxy header):
 *  - x-rinco-strategy=subpath   → first URL segment is the tenant
 *  - x-rinco-strategy=subdomain → tenant from x-rinco-tenant header
 *  - x-rinco-strategy=custom    → tenant from x-rinco-tenant header
 *
 * Writes the resolved slug to x-tenant-slug so server components
 * (e.g. app/[tenant]/page.tsx) can read it from headers().
 */

const RESERVED_API = ["/api", "/_next", "/admin", "/docs", "/help"];

const RESERVED_TENANT_SLUGS = new Set([
  "admin",
  "api",
  "docs",
  "help",
  "static",
  "assets",
  "favicon.ico",
  "robots.txt",
  "sitemap.xml",
  "_next",
]);

export function middleware(req: NextRequest) {
  const url = req.nextUrl.clone();
  const path = url.pathname;

  // Skip API and static routes
  if (RESERVED_API.some((p) => path.startsWith(p))) {
    return NextResponse.next();
  }

  // Skip file requests (have an extension)
  if (/\.[a-z0-9]{2,4}$/i.test(path)) {
    return NextResponse.next();
  }

  const strategy = req.headers.get("x-rinco-strategy") ?? "subpath";
  const headerTenant = req.headers.get("x-rinco-tenant");

  let tenantSlug = "";

  if (strategy === "subdomain" || strategy === "custom") {
    if (headerTenant) tenantSlug = headerTenant;
  } else {
    // subpath strategy
    const segments = path.split("/").filter(Boolean);
    if (segments.length >= 1 && !RESERVED_TENANT_SLUGS.has(segments[0])) {
      tenantSlug = segments[0];
    }
  }

  const res = NextResponse.next({
    request: { headers: reqHeaders(req, tenantSlug) },
  });
  if (tenantSlug) res.headers.set("x-tenant-slug", tenantSlug);
  return res;
}

function reqHeaders(req: NextRequest, tenant: string): Headers {
  const h = new Headers(req.headers);
  if (tenant) h.set("x-tenant-slug", tenant);
  return h;
}

export const config = {
  matcher: [
    /*
     * Skip API, static files, image optimizer and Next.js internals.
     */
    "/((?!api|_next/static|_next/image|favicon.ico|.*\\..*).*)",
  ],
};
