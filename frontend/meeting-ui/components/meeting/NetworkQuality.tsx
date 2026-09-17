"use client";

import { useEffect, useState } from "react";

/**
 * Network quality indicator — uses the browser's WebRTC stats to
 * surface a simple 4-bar icon when ``getStats`` is supported.
 *
 * Falls back to "unknown" (a question mark) when stats aren't
 * available — typically on Safari where the ``RTCPeerConnection``
 * lacks a usable ``getStats`` implementation in the legacy API.
 */
export type NetworkQuality = "good" | "fair" | "poor" | "unknown";

export interface NetworkQualityProps {
  /** A live peer connection whose stats we should sample. */
  peerConnection?: RTCPeerConnection | null;
  className?: string;
}

function classify(rtt: number, loss: number): NetworkQuality {
  if (!Number.isFinite(rtt)) return "unknown";
  if (loss > 0.05 || rtt > 400) return "poor";
  if (loss > 0.02 || rtt > 200) return "fair";
  return "good";
}

function bars(level: NetworkQuality): string {
  switch (level) {
    case "good":
      return "▂▃▄▅";
    case "fair":
      return "▂▃▄";
    case "poor":
      return "▂";
    default:
      return "?";
  }
}

export function NetworkQualityIndicator({
  peerConnection,
  className,
}: NetworkQualityProps) {
  const [quality, setQuality] = useState<NetworkQuality>("unknown");

  useEffect(() => {
    if (!peerConnection) {
      setQuality("unknown");
      return;
    }
    let cancelled = false;
    const sample = async () => {
      try {
        const stats = await peerConnection.getStats();
        let rtt = NaN;
        let loss = 0;
        stats.forEach((report) => {
          const r = report as unknown as {
            type?: string;
            state?: string;
            currentRoundTripTime?: number;
            packetsLost?: number;
            packetsReceived?: number;
          };
          if (r.type === "candidate-pair" && r.state === "succeeded") {
            if (typeof r.currentRoundTripTime === "number") {
              rtt = r.currentRoundTripTime * 1000;
            }
          }
          if (r.type === "inbound-rtp") {
            const lost = r.packetsLost ?? 0;
            const received = r.packetsReceived ?? 0;
            if (lost + received > 0) {
              loss = lost / (lost + received);
            }
          }
        });
        if (!cancelled) setQuality(classify(rtt, loss));
      } catch {
        if (!cancelled) setQuality("unknown");
      }
    };
    void sample();
    const id = setInterval(sample, 3000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [peerConnection]);

  return (
    <span
      data-testid="network-quality"
      title={`Network quality: ${quality}`}
      className={
        "inline-flex items-center gap-1 rounded-full bg-black/40 px-3 py-1 text-xs text-white " +
        (className ?? "")
      }
    >
      <span aria-hidden="true" className="font-mono">
        {bars(quality)}
      </span>
      <span className="capitalize">{quality}</span>
    </span>
  );
}

export default NetworkQualityIndicator;
