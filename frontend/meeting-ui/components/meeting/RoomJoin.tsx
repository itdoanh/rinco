"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getUserMedia, getMediaDevices } from "@/lib/webrtc";
import { Video, VideoOff, Mic, MicOff, Settings } from "lucide-react";

interface RoomJoinProps {
  roomId: string;
  /**
   * Optional callback invoked with the entered display name when the user
   * joins the meeting. When provided, the component will NOT navigate via
   * the router; instead it triggers a state change in the parent so the
   * meeting UI can mount without reloading the page.
   */
  onJoined?: (name: string) => void;
}

export function RoomJoin({ roomId, onJoined }: RoomJoinProps) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [videoPreview, setVideoPreview] = useState<MediaStream | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isVideoOn, setIsVideoOn] = useState(true);
  const [isAudioOn, setIsAudioOn] = useState(true);
  const [devices, setDevices] = useState<{
    cameras: MediaDeviceInfo[];
    microphones: MediaDeviceInfo[];
  }>({ cameras: [], microphones: [] });
  const [showSettings, setShowSettings] = useState(false);

  useEffect(() => {
    const init = async () => {
      try {
        const stream = await getUserMedia(true, true);
        setVideoPreview(stream);
      } catch (err) {
        setError("Could not access camera or microphone");
      } finally {
        setIsLoading(false);
      }
    };

    init();

    // Enumerate devices (best-effort; may require permission grant first).
    getMediaDevices()
      .then((d) =>
        setDevices({ cameras: d.cameras, microphones: d.microphones }),
      )
      .catch(() => {
        /* ignore — device picker is best-effort */
      });

    return () => {
      videoPreview?.getTracks().forEach((track) => track.stop());
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const toggleVideo = () => {
    if (videoPreview) {
      videoPreview.getVideoTracks().forEach((track) => {
        track.enabled = !isVideoOn;
      });
      setIsVideoOn(!isVideoOn);
    }
  };

  const toggleAudio = () => {
    if (videoPreview) {
      videoPreview.getAudioTracks().forEach((track) => {
        track.enabled = !isAudioOn;
      });
      setIsAudioOn(!isAudioOn);
    }
  };

  const joinMeeting = () => {
    if (!name.trim()) return;

    // Store settings in sessionStorage
    sessionStorage.setItem("meeting_name", name);
    sessionStorage.setItem("meeting_video", String(isVideoOn));
    sessionStorage.setItem("meeting_audio", String(isAudioOn));

    if (onJoined) {
      onJoined(name.trim());
    } else {
      router.push(`/meeting/${roomId}?name=${encodeURIComponent(name)}`);
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
      <div className="w-full max-w-4xl">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-white mb-2">Join Meeting</h1>
          <p className="text-gray-400">Room: {roomId}</p>
        </div>

        <div className="grid md:grid-cols-2 gap-8">
          {/* Video preview */}
          <div className="relative">
            <div className="video-tile bg-gray-800 rounded-2xl overflow-hidden">
              {isLoading ? (
                <div className="w-full h-full flex items-center justify-center">
                  <div className="text-gray-400">Loading camera...</div>
                </div>
              ) : videoPreview ? (
                <video
                  autoPlay
                  playsInline
                  muted
                  ref={(el) => {
                    if (el && videoPreview) {
                      el.srcObject = videoPreview;
                    }
                  }}
                  className="w-full h-full object-cover scale-x-[-1]"
                />
              ) : (
                <div className="w-full h-full flex items-center justify-center">
                  <div className="text-center">
                    <VideoOff className="w-16 h-16 text-gray-600 mx-auto mb-4" />
                    <p className="text-gray-400">Camera is off</p>
                  </div>
                </div>
              )}
            </div>

            {/* Video controls */}
            <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-4">
              <Button
                variant={isAudioOn ? "secondary" : "destructive"}
                size="icon"
                className="rounded-full w-12 h-12"
                onClick={toggleAudio}
                aria-label={isAudioOn ? "Mute microphone" : "Unmute microphone"}
              >
                {isAudioOn ? <Mic className="w-5 h-5" /> : <MicOff className="w-5 h-5" />}
              </Button>
              <Button
                variant={isVideoOn ? "secondary" : "destructive"}
                size="icon"
                className="rounded-full w-12 h-12"
                onClick={toggleVideo}
                aria-label={isVideoOn ? "Turn camera off" : "Turn camera on"}
              >
                {isVideoOn ? <Video className="w-5 h-5" /> : <VideoOff className="w-5 h-5" />}
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="rounded-full w-12 h-12"
                onClick={() => setShowSettings((v) => !v)}
                aria-label="Device settings"
              >
                <Settings className="w-5 h-5" />
              </Button>
            </div>
          </div>

          {/* Join form */}
          <div className="bg-gray-800 rounded-2xl p-8">
            <h2 className="text-xl font-bold text-white mb-6">Join as</h2>

            {error && (
              <div className="bg-red-500/20 border border-red-500/50 text-red-400 px-4 py-3 rounded-lg mb-6">
                {error}
              </div>
            )}

            <div className="space-y-6">
              <div>
                <Label htmlFor="userName" className="text-gray-300">
                  Your name
                </Label>
                <Input
                  id="userName"
                  name="userName"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && name.trim() && !isLoading) {
                      joinMeeting();
                    }
                  }}
                  placeholder="Enter your name"
                  className="mt-2 bg-gray-700 border-gray-600 text-white"
                />
              </div>

              {showSettings && (devices.cameras.length > 0 || devices.microphones.length > 0) && (
                <div className="bg-gray-900/60 rounded-lg p-4 space-y-3">
                  <p className="text-sm text-gray-300 font-medium">Devices</p>
                  {devices.cameras.length > 0 && (
                    <div>
                      <Label htmlFor="camera-select" className="text-gray-400 text-xs">
                        Camera
                      </Label>
                      <select
                        id="camera-select"
                        className="w-full mt-1 bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm text-white"
                        defaultValue={devices.cameras[0]?.deviceId}
                      >
                        {devices.cameras.map((d) => (
                          <option key={d.deviceId} value={d.deviceId}>
                            {d.label || `Camera ${d.deviceId.slice(0, 6)}`}
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                  {devices.microphones.length > 0 && (
                    <div>
                      <Label htmlFor="mic-select" className="text-gray-400 text-xs">
                        Microphone
                      </Label>
                      <select
                        id="mic-select"
                        className="w-full mt-1 bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm text-white"
                        defaultValue={devices.microphones[0]?.deviceId}
                      >
                        {devices.microphones.map((d) => (
                          <option key={d.deviceId} value={d.deviceId}>
                            {d.label || `Microphone ${d.deviceId.slice(0, 6)}`}
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                </div>
              )}

              <Button
                onClick={joinMeeting}
                disabled={!name.trim() || isLoading}
                className="w-full"
                size="lg"
              >
                Join
              </Button>

              <p className="text-center text-gray-400 text-sm">
                Make sure your camera and microphone are working properly.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default RoomJoin;

