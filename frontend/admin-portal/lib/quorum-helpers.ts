/**
 * Small formatting helpers extracted from the Quorum page so they can be
 * unit-tested without spinning up a Next.js client component.
 */

export function formatRemaining(expiresAt: string, now: number): number {
  return Math.round((new Date(expiresAt).getTime() - now) / 1000);
}

export function formatCountdown(seconds: number): string {
  if (seconds <= 0) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}
