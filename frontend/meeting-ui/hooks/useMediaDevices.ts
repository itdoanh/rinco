"use client";

import { useEffect, useState } from "react";

/**
 * Media device enumeration hook.
 *
 * Lists cameras, microphones, and (when supported) speakers.  The hook
 * automatically re-enumerates whenever the browser fires a
 * `devicechange` event, so users plugging in a USB webcam see it
 * without a page reload.
 */
export interface MediaDeviceList {
  cameras: MediaDeviceInfo[];
  microphones: MediaDeviceInfo[];
  speakers: MediaDeviceInfo[];
}

const empty: MediaDeviceList = { cameras: [], microphones: [], speakers: [] };

export function useMediaDevices(): MediaDeviceList {
  const [devices, setDevices] = useState<MediaDeviceList>(empty);

  useEffect(() => {
    if (
      typeof navigator === "undefined" ||
      !navigator.mediaDevices?.enumerateDevices
    ) {
      return;
    }

    const update = async (): Promise<void> => {
      try {
        const list = await navigator.mediaDevices.enumerateDevices();
        setDevices({
          cameras: list.filter((d) => d.kind === "videoinput"),
          microphones: list.filter((d) => d.kind === "audioinput"),
          speakers: list.filter((d) => d.kind === "audiooutput"),
        });
      } catch {
        // Permission denied or no devices — keep empty list.
        setDevices(empty);
      }
    };

    void update();
    navigator.mediaDevices.addEventListener("devicechange", update);
    return () => {
      navigator.mediaDevices.removeEventListener("devicechange", update);
    };
  }, []);

  return devices;
}
