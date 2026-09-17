"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { getDisplayMedia } from "@/lib/webrtc";
import type { ScreenShareState } from "@/lib/types";

/**
 * Screen share hook.
 *
 * Acquires a display media stream and tracks its state.  When the
 * browser stops the share (the user clicks the "Stop sharing" OS
 * notification) the hook fires ``onStopped`` so the parent can revert
 * any UI state that was bound to ``isSharing``.
 */
export interface UseScreenShareOptions {
  onStopped?: () => void;
  /** Include the OS-level audio (e.g. music from a shared tab). */
  audio?: boolean;
}

export function useScreenShare({
  onStopped,
  audio = false,
}: UseScreenShareOptions = {}): ScreenShareState & {
  start: () => Promise<MediaStream | null>;
  stop: () => void;
} {
  const [stream, setStream] = useState<MediaStream | null>(null);
  const [isSharing, setIsSharing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const onStoppedRef = useRef(onStopped);

  useEffect(() => {
    onStoppedRef.current = onStopped;
  }, [onStopped]);

  const stop = useCallback(() => {
    setStream((current) => {
      if (current) {
        for (const track of current.getTracks()) {
          try {
            track.stop();
          } catch {
            /* ignore */
          }
        }
      }
      return null;
    });
    setIsSharing(false);
  }, []);

  const start = useCallback(async (): Promise<MediaStream | null> => {
    setError(null);
    try {
      const displayStream = await getDisplayMedia({ audio });
      setStream(displayStream);
      setIsSharing(true);
      const track = displayStream.getVideoTracks()[0];
      if (track) {
        track.addEventListener("ended", () => {
          stop();
          onStoppedRef.current?.();
        });
      }
      return displayStream;
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to start screen share";
      setError(message);
      setIsSharing(false);
      return null;
    }
  }, [audio, stop]);

  useEffect(() => {
    return () => {
      // Stop any active stream when the hook unmounts.
      setStream((current) => {
        if (current) {
          for (const track of current.getTracks()) {
            try {
              track.stop();
            } catch {
              /* ignore */
            }
          }
        }
        return null;
      });
    };
  }, []);

  return { stream, isSharing, error, start, stop };
}
