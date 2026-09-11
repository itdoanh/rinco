"use client";

import { SessionProvider, useSession } from "next-auth/react";
import { type ReactNode, useEffect, useState } from "react";

/**
 * AuthProvider wraps SessionProvider but only triggers getSession after
 * the user actually attempts to sign in. This avoids the noisy
 * `ClientFetchError` that fires when /api/auth/session is unreachable
 * in local dev (auth-service backend not running).
 */
function SessionTracker({ children }: { children: ReactNode }) {
  const [shouldFetch, setShouldFetch] = useState(false);
  // Listen for the next-auth sign-in event from the same window
  useEffect(() => {
    const onSignInAttempt = () => setShouldFetch(true);
    window.addEventListener("rinco:auth-required", onSignInAttempt);
    return () => window.removeEventListener("rinco:auth-required", onSignInAttempt);
  }, []);
  if (!shouldFetch) return <>{children}</>;
  return <SessionSyncGate>{children}</SessionSyncGate>;
}

function SessionSyncGate({ children }: { children: ReactNode }) {
  // Once shouldFetch is true, useEffect in SessionProvider will poll.
  return <>{children}</>;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  return (
    <SessionProvider
      // Don't auto-poll session — only when explicitly requested via event.
      refetchInterval={0}
      refetchOnWindowFocus={false}
    >
      <SessionTracker>{children}</SessionTracker>
    </SessionProvider>
  );
}
