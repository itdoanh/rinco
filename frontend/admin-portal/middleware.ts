/**
 * Auth middleware for the admin portal.
 *
 * Protects all routes under (dashboard) so unauthenticated visitors
 * are redirected to the login page. Public routes — the login page
 * itself, the API auth handlers, and the root — are allowed through.
 *
 * Implementation note: we check for the presence of the NextAuth
 * session cookie rather than calling NextAuth's `auth()` helper,
 * which avoids re-initializing the v5 handler stack at the Edge.
 */
import { NextResponse, type NextRequest } from "next/server";

const COOKIE_NAMES = [
  "authjs.session-token",       // NextAuth v5 (https)
  "__Secure-authjs.session-token", // NextAuth v5 (secure context)
  "next-auth.session-token",   // NextAuth v4 (fallback)
  "__Secure-next-auth.session-token",
];

export function middleware(req: NextRequest) {
  const pathname = req.nextUrl.pathname;

  // Allow auth + login + root through.
  if (
    pathname.startsWith("/api/auth") ||
    pathname.startsWith("/login") ||
    pathname === "/" ||
    pathname.startsWith("/_next") ||
    pathname === "/favicon.ico"
  ) {
    return NextResponse.next();
  }

  const hasSession = COOKIE_NAMES.some((name) =>
    req.cookies.has(name),
  );

  if (!hasSession) {
    const url = new URL("/login", req.nextUrl);
    url.searchParams.set("from", pathname);
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  // Match everything except static files.
  matcher: ["/((?!_next/static|_next/image|favicon.ico|.*\\.svg|.*\\.png|.*\\.jpg).*)"],
};
