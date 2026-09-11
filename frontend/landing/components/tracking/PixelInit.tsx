"use client";

import { useEffect } from "react";
import { initPixel } from "@/lib/pixel";

interface PixelInitProps {
  pixelId?: string;
}

/**
 * Client component that initializes the Meta Pixel once on mount.
 *
 * Delegates the actual stub + script creation to `initPixel` in lib/pixel.ts
 * so both `PixelInit` and `Tracker` share the same safe stub.
 */
export function PixelInit({ pixelId }: PixelInitProps) {
  useEffect(() => {
    initPixel(pixelId);
  }, [pixelId]);
  return null;
}

export default PixelInit;
