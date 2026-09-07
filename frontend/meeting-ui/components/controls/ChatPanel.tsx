"use client";

import { useState, useRef, useEffect } from "react";
import { useMeetingStore, type ChatMessage } from "@/lib/store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { X, Send } from "lucide-react";

interface ChatPanelProps {
  onClose: () => void;
}

export function ChatPanel({ onClose }: ChatPanelProps) {
  const [message, setMessage] = useState("");
  const { chatMessages, addChatMessage, roomId, userId, userName } = useMeetingStore();
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [chatMessages]);

  const sendMessage = () => {
    if (!message.trim()) return;

    addChatMessage({
      id: Date.now().toString(),
      senderId: userId,
      senderName: userName,
      content: message.trim(),
      timestamp: new Date(),
    });

    setMessage("");
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between p-4 border-b">
        <h3 className="font-bold">Chat</h3>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="w-4 h-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {chatMessages.length === 0 ? (
          <div className="text-center text-gray-500 py-8">
            <div className="text-4xl mb-2">💬</div>
            <p>No messages yet</p>
          </div>
        ) : (
          chatMessages.map((msg) => (
            <MessageBubble key={msg.id} message={msg} isOwn={msg.senderId === userId} />
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="p-4 border-t">
        <div className="flex gap-2">
          <Input
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="Type a message..."
            className="flex-1"
          />
          <Button onClick={sendMessage} disabled={!message.trim()}>
            <Send className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}

function MessageBubble({ message, isOwn }: { message: ChatMessage; isOwn: boolean }) {
  const formatTime = (date: Date) => {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  return (
    <div className={isOwn ? "text-right" : "text-left"}>
      <div className="inline-block max-w-[80%]">
        {!isOwn && (
          <p className="text-xs text-gray-400 mb-1 ml-2">{message.senderName}</p>
        )}
        <div
          className={`inline-block px-4 py-2 rounded-2xl ${
            isOwn
              ? "bg-primary text-white rounded-br-md"
              : "bg-gray-100 text-gray-900 rounded-bl-md"
          }`}
        >
          <p className="text-sm">{message.content}</p>
        </div>
        <p className="text-xs text-gray-400 mt-1 mr-2">
          {formatTime(message.timestamp)}
        </p>
      </div>
    </div>
  );
}

export default ChatPanel;
