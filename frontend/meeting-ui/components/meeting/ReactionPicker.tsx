"use client";

import { useState } from "react";

/**
 * Floating emoji reaction picker — sends a one-shot reaction event to
 * everyone in the meeting.  Currently local-only (it animates an
 * emoji that floats up the screen) so we can showcase the UI even
 * without a real chat-engine channel for reactions.  When the
 * meeting-ui gets a reactions channel the float animation should be
 * replaced with a ``signaling.sendReaction()`` call.
 */
export interface ReactionPickerProps {
  onReact?: (emoji: string) => void;
}

const REACTIONS = ["👍", "👏", "❤️", "😂", "🎉", "🔥"] as const;

export function ReactionPicker({ onReact }: ReactionPickerProps) {
  const [burst, setBurst] = useState<{ id: number; emoji: string } | null>(
    null,
  );

  const handle = (emoji: string) => {
    const id = Date.now();
    setBurst({ id, emoji });
    onReact?.(emoji);
    // Auto-clear after the float animation finishes (1.5s).
    setTimeout(() => {
      setBurst((current) => (current && current.id === id ? null : current));
    }, 1500);
  };

  return (
    <div
      data-testid="reaction-picker"
      className="pointer-events-auto inline-flex items-center gap-1 rounded-full bg-slate-800/80 px-2 py-1 shadow-lg backdrop-blur"
    >
      {REACTIONS.map((emoji) => (
        <button
          key={emoji}
          type="button"
          onClick={() => handle(emoji)}
          className="rounded-full px-2 py-1 text-lg transition hover:scale-125 hover:bg-white/10"
          aria-label={`Send ${emoji}`}
        >
          {emoji}
        </button>
      ))}
      {burst && (
        <span
          key={burst.id}
          className="pointer-events-none absolute -top-10 left-1/2 -translate-x-1/2 text-3xl reaction-float"
          aria-hidden="true"
        >
          {burst.emoji}
        </span>
      )}
    </div>
  );
}

export default ReactionPicker;
