"use client";

import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";

/**
 * Full-screen screen share preview.
 *
 * Renders the local screen-share stream (acquired via
 * ``useScreenShare``) in a modal-style overlay.  The user can dismiss
 * the share with the close button which triggers ``onStop``.
 */
export interface ScreenShareProps {
  stream: MediaStream | null;
  sharerName: string;
  onStop: () => void;
}

export function ScreenShare({ stream, sharerName, onStop }: ScreenShareProps) {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  if (!stream) return null;

  return (
    <div
      data-testid="screen-share-overlay"
      className="fixed inset-0 z-40 flex flex-col bg-black"
    >
      <div className="flex items-center justify-between bg-slate-900/80 px-4 py-3 text-white">
        <div>
          <p className="text-sm uppercase tracking-wide text-slate-400">
            Đang chia sẻ màn hình
          </p>
          <p className="text-base font-semibold">{sharerName}</p>
        </div>
        <Button
          variant="destructive"
          onClick={onStop}
          className="rounded-full"
        >
          <X className="mr-2 h-4 w-4" />
          Dừng chia sẻ
        </Button>
      </div>
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted
        className="h-full w-full bg-black object-contain"
      />
    </div>
  );
}

export default ScreenShare;
