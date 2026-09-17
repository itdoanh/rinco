"use client";

import { useEffect, useState } from "react";

/**
 * Subscribes to the participants array exposed by the meeting store
 * (Zustand) and returns a memoised copy.  We don't deeply memoise —
 * callers receive a fresh array reference on every state change so
 * React's prop-equality checks work the same as with a plain
 * ``useState`` hook.
 */
export interface ParticipantSnapshot {
  id: string;
  name: string;
  isMuted: boolean;
  isVideoOn: boolean;
  isScreenSharing: boolean;
}

export function useParticipants(_roomId: string): {
  participants: ParticipantSnapshot[];
  count: number;
} {
  const [participants, setParticipants] = useState<ParticipantSnapshot[]>([]);

  useEffect(() => {
    // Dynamic import to avoid pulling zustand into the SSR bundle
    // when the hook isn't actually used yet.  The store is a singleton
    // so subscribe() will fire once for every state change.
    let unsubscribe: (() => void) | undefined;
    void import("@/lib/store").then(({ useMeetingStore }) => {
      const sync = () => {
        const state = useMeetingStore.getState();
        setParticipants(
          state.participants.map((p) => ({
            id: p.id,
            name: p.name,
            isMuted: p.isMuted,
            isVideoOn: p.isVideoOn,
            isScreenSharing: p.isScreenSharing,
          })),
        );
      };
      sync();
      unsubscribe = useMeetingStore.subscribe(sync);
    });
    return () => {
      unsubscribe?.();
    };
  }, []);

  return { participants, count: participants.length };
}
