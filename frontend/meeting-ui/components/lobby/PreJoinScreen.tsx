"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Video, VideoOff, Mic, MicOff } from "lucide-react";

/**
 * Lobby pre-join screen — collects the user name, lets them preview
 * their camera + microphone, and finally exposes a ``Join`` callback.
 *
 * The actual ``MediaStream`` is owned by the parent (``page.tsx``) so
 * the same stream can be re-used once the meeting starts.
 */
export interface PreJoinScreenProps {
  roomId: string;
  defaultName?: string;
  isLoading?: boolean;
  error?: string | null;
  previewStream: MediaStream | null;
  videoEnabled: boolean;
  audioEnabled: boolean;
  onToggleVideo: () => void;
  onToggleAudio: () => void;
  onJoin: (name: string) => void;
}

export function PreJoinScreen({
  roomId,
  defaultName = "",
  isLoading = false,
  error = null,
  previewStream,
  videoEnabled,
  audioEnabled,
  onToggleVideo,
  onToggleAudio,
  onJoin,
}: PreJoinScreenProps) {
  const [name, setName] = useState(defaultName);

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim()) return;
    onJoin(name.trim());
  };

  return (
    <div
      data-testid="pre-join-screen"
      className="flex min-h-screen items-center justify-center bg-slate-900 p-6 text-white"
    >
      <div className="grid w-full max-w-4xl gap-6 rounded-2xl bg-slate-800/80 p-8 md:grid-cols-2">
        {/* Preview */}
        <div className="relative aspect-video overflow-hidden rounded-xl bg-slate-900">
          {previewStream && videoEnabled ? (
            <video
              autoPlay
              playsInline
              muted
              ref={(el) => {
                if (el) el.srcObject = previewStream;
              }}
              className="h-full w-full object-cover -scale-x-100"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-slate-500">
              <VideoOff className="h-12 w-12" />
            </div>
          )}
          <div className="absolute bottom-3 left-1/2 flex -translate-x-1/2 gap-2">
            <Button
              data-testid="prejoin-toggle-mic"
              variant={audioEnabled ? "secondary" : "destructive"}
              size="icon"
              className="h-10 w-10 rounded-full"
              onClick={onToggleAudio}
              aria-label={audioEnabled ? "Tắt mic" : "Bật mic"}
            >
              {audioEnabled ? <Mic className="h-4 w-4" /> : <MicOff className="h-4 w-4" />}
            </Button>
            <Button
              data-testid="prejoin-toggle-video"
              variant={videoEnabled ? "secondary" : "destructive"}
              size="icon"
              className="h-10 w-10 rounded-full"
              onClick={onToggleVideo}
              aria-label={videoEnabled ? "Tắt camera" : "Bật camera"}
            >
              {videoEnabled ? <Video className="h-4 w-4" /> : <VideoOff className="h-4 w-4" />}
            </Button>
          </div>
        </div>

        {/* Join form */}
        <form
          onSubmit={handleSubmit}
          className="flex flex-col justify-center gap-4"
        >
          <div>
            <h1 className="text-2xl font-bold">Tham gia phòng họp</h1>
            <p className="text-sm text-slate-400">Mã phòng: {roomId}</p>
          </div>
          {error && (
            <p
              role="alert"
              className="rounded-lg border border-red-500/40 bg-red-500/10 px-3 py-2 text-sm text-red-300"
            >
              {error}
            </p>
          )}
          <label className="block">
            <span className="text-sm font-semibold text-slate-300">
              Tên hiển thị
            </span>
            <input
              data-testid="display-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              minLength={1}
              placeholder="Nhập tên của bạn"
              className="mt-2 w-full rounded-lg border border-slate-600 bg-slate-700 px-4 py-2 text-white placeholder:text-slate-400 focus:border-blue-500 focus:outline-none"
            />
          </label>
          <Button
            data-testid="join-button"
            type="submit"
            size="lg"
            disabled={isLoading || !name.trim()}
            className="w-full"
          >
            {isLoading ? "Đang kết nối…" : "Tham gia"}
          </Button>
        </form>
      </div>
    </div>
  );
}

export default PreJoinScreen;
