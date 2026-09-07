import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Video } from "lucide-react";
import { useState } from "react";

export default function HomePage() {
  const [roomId, setRoomId] = useState("");

  const generateRoomId = () => {
    return Math.random().toString(36).substring(2, 10);
  };

  const createRoom = () => {
    const newRoomId = generateRoomId();
    window.location.href = `/meeting/${newRoomId}`;
  };

  const joinRoom = () => {
    if (roomId.trim()) {
      window.location.href = `/meeting/${roomId.trim()}`;
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
      <div className="w-full max-w-lg">
        {/* Logo */}
        <div className="text-center mb-12">
          <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center mx-auto mb-6">
            <Video className="w-10 h-10 text-white" />
          </div>
          <h1 className="text-4xl font-bold text-white mb-2">RINCO Meeting</h1>
          <p className="text-gray-400">Video conferencing for teams</p>
        </div>

        {/* Actions */}
        <div className="space-y-4">
          {/* Create room */}
          <div className="bg-gray-800 rounded-2xl p-6">
            <h2 className="text-lg font-bold text-white mb-4">Start a new meeting</h2>
            <Button onClick={createRoom} className="w-full" size="lg">
              Create Room
            </Button>
          </div>

          {/* Join room */}
          <div className="bg-gray-800 rounded-2xl p-6">
            <h2 className="text-lg font-bold text-white mb-4">Join a meeting</h2>
            <div className="space-y-4">
              <div>
                <Label htmlFor="roomId" className="text-gray-300">
                  Room ID
                </Label>
                <Input
                  id="roomId"
                  value={roomId}
                  onChange={(e) => setRoomId(e.target.value)}
                  placeholder="Enter room ID"
                  className="mt-2 bg-gray-700 border-gray-600 text-white"
                />
              </div>
              <Button
                onClick={joinRoom}
                disabled={!roomId.trim()}
                variant="outline"
                className="w-full"
                size="lg"
              >
                Join
              </Button>
            </div>
          </div>
        </div>

        {/* Footer */}
        <p className="text-center text-gray-500 text-sm mt-8">
          By joining, you agree to our Terms of Service and Privacy Policy
        </p>
      </div>
    </div>
  );
}
