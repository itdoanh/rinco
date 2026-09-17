import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * RINCO Multi-Domain Routing Middleware
 *
 * Supports 3 routing strategies for tenant resolution:
 *  1. Subpath   — hanghoaphaisinh.net/{tenant_slug}
 *  2. Subdomain — {tenant_slug}.hanghoaphaisinh.net
 *  3. Custom    — custom domain registered in tenant.tenant_domains
 *
 * Strategy is chosen at runtime via x-rinco-host header (injected
 * by the upstream reverse-proxy / Traefik ingress based on TLS SNI).
 *
 * - If x-rinco-strategy=subdomain   → extract tenant from x-rinco-tenant header
 * - If x-rinco-strategy=custom       → lookup via x-rinco-tenant header
 * - Else                              → first segment of the URL path
 *
 * Writes resolved tenant slug to `x-tenant-slug` request header so
 * server components can read it via `headers()`.
 */

const ALLOWED_ROOT_HOSTS = new Set([
  "localhost",
  "hanghoaphaisinh.net",
  "www.hanghoaphaisinh.net",
]);

const RESERVED_PATHS = new Set([
  "_next",
  "api",
  "favicon.ico",
  "robots.txt",
  "sitemap.xml",
  "static",
  "assets",
  "admin",
]);

const RESERVED_TOP_SLUGS = new Set([
  "admin",
  "api",
  "dashboard",
  "login",
  "logout",
  "auth",
  "docs",
  "help",
]);

export function middleware(req: NextRequest) {
  const url = req.nextUrl.clone();
  const host = req.headers.get("host") ?? "";
  const hostname = host.split(":")[0].toLowerCase();

  // Skip middleware for static and Next.js internal paths
  if (RESERVED_PATHS.has(url.pathname.split("/")[1] ?? "")) {
    return NextResponse.next();
  }

  // Resolve tenant based on routing strategy
  let tenantSlug = "";

  // 1. Custom domain or subdomain signal from reverse-proxy
  const strategy = req.headers.get("x-rinco-strategy");
  const headerTenant = req.headers.get("x-rinco-tenant");

  if ((strategy === "custom" || strategy === "subdomain") && headerTenant) {
    tenantSlug = headerTenant;
  } else if (strategy === "subpath" || !strategy) {
    // 2. Subpath routing — first segment is tenant slug
    const segments = url.pathname.split("/").filter(Boolean);
    if (segments.length > 0 && !RESERVED_TOP_SLUGS.has(segments[0])) {
      tenantSlug = segments[0];
      // Rewrite URL so [tenant] dynamic segment picks it up
      // e.g. /apexfintech → /apexfintech (already correct for [tenant])
    }
  } else if (strategy === "subdomain") {
    // 3. Subdomain fallback — extract from hostname
    if (
      !ALLOWED_ROOT_HOSTS.has(hostname) &&
      hostname.endsWith(".hanghoaphaisinh.net")
    ) {
      const sub = hostname.replace(".hanghoaphaisinh.net", "");
      if (sub && !RESERVED_TOP_SLUGS.has(sub)) {
        tenantSlug = sub;
        // Rewrite URL — strip subdomain prefix from path if present
        url.pathname = url.pathname.replace(/^\/?[^/]+/, "");
        if (!url.pathname.startsWith("/")) url.pathname = "/" + url.pathname;
      }
    }
  }

  if (!tenantSlug) {
    return NextResponse.next();
  }

  // Pass tenant context to server components
  const res = NextResponse.next({ request: { headers: reqHeaders(req, tenantSlug) } });
  res.headers.set("x-tenant-slug", tenantSlug);
  return res;
}

function reqHeaders(req: NextRequest, tenant: string): Headers {
  const h = new Headers(req.headers);
  h.set("x-tenant-slug", tenant);
  return h;
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public assets
     */
    "/((?!api|_next/static|_next/image|favicon.ico|robots.txt|sitemap.xml|.*\\..*).*)",
  ],
};
