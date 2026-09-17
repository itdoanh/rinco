"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Send, X } from "lucide-react";
import type { ChatMessage } from "@/lib/types";

/**
 * In-meeting chat panel.
 *
 * Pure UI — the WebSocket round-trip is owned by ``useChat`` in the
 * parent.  This component just renders the message list and exposes
 * ``onSend`` for the parent's transport layer.
 */
export interface ChatProps {
  messages: ChatMessage[];
  onSend: (text: string) => void;
  onClose?: () => void;
  currentUserId?: string;
}

function formatTime(ts: number): string {
  try {
    return new Date(ts).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "";
  }
}

export function Chat({ messages, onSend, onClose, currentUserId }: ChatProps) {
  const [text, setText] = useState("");
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = () => {
    const trimmed = text.trim();
    if (!trimmed) return;
    onSend(trimmed);
    setText("");
  };

  return (
    <aside
      data-testid="chat-panel"
      className="flex h-full w-full flex-col border-l border-slate-700 bg-slate-800 text-white"
    >
      <header className="flex items-center justify-between border-b border-slate-700 px-4 py-3">
        <h3 className="font-semibold">Chat</h3>
        {onClose && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            aria-label="Đóng chat"
          >
            <X className="h-4 w-4" />
          </Button>
        )}
      </header>

      <div className="flex-1 space-y-3 overflow-y-auto p-4">
        {messages.length === 0 ? (
          <p className="py-8 text-center text-sm text-slate-400">
            Chưa có tin nhắn nào.
          </p>
        ) : (
          messages.map((msg) => {
            const isOwn = currentUserId ? msg.senderId === currentUserId : false;
            return (
              <div
                key={msg.id}
                className={
                  "flex flex-col gap-1 rounded-lg p-3 " +
                  (isOwn
                    ? "bg-blue-600/30 text-right"
                    : "bg-slate-700 text-left")
                }
              >
                <div className="flex items-center justify-between text-xs text-slate-300">
                  <span className="font-semibold">{msg.senderName}</span>
                  <span>{formatTime(msg.timestamp)}</span>
                </div>
                <div className="text-sm text-white">{msg.content}</div>
              </div>
            );
          })
        )}
        <div ref={endRef} />
      </div>

      <div className="flex gap-2 border-t border-slate-700 p-3">
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSend();
            }
          }}
          placeholder="Nhập tin nhắn..."
          className="flex-1 border-slate-600 bg-slate-700 text-white placeholder:text-slate-400"
        />
        <Button onClick={handleSend} disabled={!text.trim()}>
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </aside>
  );
}

export default Chat;
