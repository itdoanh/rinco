"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Camera, Mic, Check } from "lucide-react";

/**
 * Compact device check widget for the lobby.  Requests camera + mic
 * permission, displays a live preview thumbnail, and reports success
 * via ``onComplete`` so the parent can enable the join button.
 */
export interface DeviceCheckProps {
  onComplete?: (stream: MediaStream) => void;
}

export function DeviceCheck({ onComplete }: DeviceCheckProps) {
  const [stream, setStream] = useState<MediaStream | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    return () => {
      // Stop preview tracks on unmount so the camera light goes off.
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

  const run = async () => {
    setError(null);
    setBusy(true);
    try {
      const next = await navigator.mediaDevices.getUserMedia({
        audio: true,
        video: { width: 640, height: 360 },
      });
      setStream(next);
      onComplete?.(next);
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Không thể truy cập camera / microphone";
      setError(message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      data-testid="device-check"
      className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700"
    >
      <div className="flex items-center gap-2">
        <Camera className="h-4 w-4" />
        <Mic className="h-4 w-4" />
        <span className="font-semibold">Kiểm tra thiết bị</span>
      </div>
      {stream ? (
        <div className="flex items-center gap-2 text-emerald-700">
          <Check className="h-4 w-4" />
          <span>Sẵn sàng tham gia</span>
          <video
            autoPlay
            playsInline
            muted
            ref={(el) => {
              if (el) el.srcObject = stream;
            }}
            className="ml-auto h-12 w-16 rounded object-cover"
          />
        </div>
      ) : (
        <>
          {error && <p className="text-red-600">{error}</p>}
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={run}
            disabled={busy}
          >
            {busy ? "Đang kiểm tra…" : "Kiểm tra ngay"}
          </Button>
        </>
      )}
    </div>
  );
}

export default DeviceCheck;
