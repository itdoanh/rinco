'use client';

import { useRef, useEffect } from 'react';
import { Maximize2, Minimize2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ScreenShareProps {
  stream?: MediaStream;
  isActive: boolean;
}

export function ScreenShare({ stream, isActive }: ScreenShareProps) {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  if (!isActive) {
    return null;
  }

  return (
    <div className="relative bg-gray-900 rounded-xl overflow-hidden border border-gray-700">
      <video
        ref={videoRef}
        autoPlay
        playsInline
        className="w-full h-full object-contain"
      />

      {/* Overlay */}
      <div className="absolute top-4 left-4 flex items-center gap-2">
        <div className="px-3 py-1.5 bg-blue-500 rounded-full text-sm text-white font-medium">
          Screen Sharing
        </div>
      </div>

      <button className="absolute top-4 right-4 p-2 bg-gray-800/80 hover:bg-gray-700 rounded-lg">
        <Maximize2 className="w-5 h-5 text-white" />
      </button>
    </div>
  );
}
