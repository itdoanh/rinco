"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getUserMedia, getMediaDevices } from "@/lib/webrtc";
import { Video, VideoOff, Mic, MicOff, Settings } from "lucide-react";

export function RoomJoin({ roomId }: { roomId: string }) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [videoPreview, setVideoPreview] = useState<MediaStream | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isVideoOn, setIsVideoOn] = useState(true);
  const [isAudioOn, setIsAudioOn] = useState(true);

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

    return () => {
      videoPreview?.getTracks().forEach((track) => track.stop());
    };
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
    
    router.push(`/meeting/${roomId}?name=${encodeURIComponent(name)}`);
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
              >
                {isAudioOn ? <Mic className="w-5 h-5" /> : <MicOff className="w-5 h-5" />}
              </Button>
              <Button
                variant={isVideoOn ? "secondary" : "destructive"}
                size="icon"
                className="rounded-full w-12 h-12"
                onClick={toggleVideo}
              >
                {isVideoOn ? <Video className="w-5 h-5" /> : <VideoOff className="w-5 h-5" />}
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
                <Label htmlFor="name" className="text-gray-300">
                  Your name
                </Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Enter your name"
                  className="mt-2 bg-gray-700 border-gray-600 text-white"
                />
              </div>

              <Button
                onClick={joinMeeting}
                disabled={!name.trim() || isLoading}
                className="w-full"
                size="lg"
              >
                Join Meeting
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
