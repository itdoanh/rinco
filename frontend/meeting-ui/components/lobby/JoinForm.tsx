"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Video } from "lucide-react";
import DeviceCheck from "./DeviceCheck";

/**
 * Lobby join form — collect room id + display name, run a quick
 * device check, then navigate to the meeting room.
 */
export interface JoinFormProps {
  defaultRoomId?: string;
}

export function JoinForm({ defaultRoomId = "" }: JoinFormProps) {
  const router = useRouter();
  const [roomId, setRoomId] = useState(defaultRoomId);
  const [name, setName] = useState("");
  const [previewStream, setPreviewStream] = useState<MediaStream | null>(null);
  const [error, setError] = useState<string | null>(null);

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim() || !roomId.trim()) return;
    if (!previewStream) {
      setError("Vui lòng hoàn thành kiểm tra thiết bị trước khi tham gia.");
      return;
    }
    const url = `/meeting/${encodeURIComponent(roomId.trim())}?name=${encodeURIComponent(name.trim())}`;
    router.push(url);
  };

  return (
    <form
      data-testid="lobby-join-form"
      onSubmit={submit}
      className="space-y-4"
    >
      <div className="flex justify-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500 to-blue-700 text-white">
          <Video className="h-8 w-8" />
        </div>
      </div>
      <h1 className="text-center text-3xl font-bold text-slate-900">
        RINCO Meeting
      </h1>
      <p className="text-center text-sm text-slate-500">
        Họp trực tuyến không giới hạn
      </p>

      <Input
        data-testid="username"
        type="text"
        placeholder="Tên hiển thị"
        value={name}
        onChange={(e) => setName(e.target.value)}
        required
      />
      <Input
        data-testid="room-id"
        type="text"
        placeholder="Mã phòng"
        value={roomId}
        onChange={(e) => setRoomId(e.target.value)}
        required
      />

      <DeviceCheck onComplete={(s) => { setPreviewStream(s); setError(null); }} />

      {error && (
        <p
          role="alert"
          className="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700"
        >
          {error}
        </p>
      )}

      <Button
        data-testid="join-button"
        type="submit"
        size="lg"
        className="w-full"
        disabled={!name.trim() || !roomId.trim() || !previewStream}
      >
        Tham gia
      </Button>
      <button
        type="button"
        onClick={() => {
          if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
            setRoomId(crypto.randomUUID().slice(0, 8));
          } else {
            setRoomId(Math.random().toString(36).slice(2, 10));
          }
        }}
        className="block w-full text-sm text-blue-600 hover:underline"
      >
        Tạo phòng mới
      </button>
    </form>
  );
}

export default JoinForm;
