'use client';

import { useState, useRef, useEffect } from 'react';
import { Send, X } from 'lucide-react';
import { MessageBubble } from './MessageBubble';
import { cn } from '@/lib/utils';

interface Message {
  id: string;
  userId: string;
  userName: string;
  content: string;
  timestamp: Date;
}

interface ChatPanelProps {
  roomId: string;
  userId: string;
  userName: string;
}

export function ChatPanel({ roomId, userId, userName }: ChatPanelProps) {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      userId: 'system',
      userName: 'System',
      content: 'Welcome to the chat!',
      timestamp: new Date(),
    },
  ]);
  const [inputValue, setInputValue] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Scroll to bottom on new messages
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSendMessage = () => {
    if (!inputValue.trim()) return;

    const newMessage: Message = {
      id: Date.now().toString(),
      userId,
      userName,
      content: inputValue.trim(),
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, newMessage]);
    setInputValue('');
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-700 flex items-center justify-between">
        <h3 className="text-white font-semibold">Chat</h3>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.map((message) => (
          <MessageBubble
            key={message.id}
            message={message}
            isOwnMessage={message.userId === userId}
          />
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="p-4 border-t border-gray-700">
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="Type a message..."
            className={cn(
              'flex-1 px-4 py-2 bg-gray-700 border border-gray-600 rounded-full',
              'text-white placeholder:text-gray-400',
              'focus:outline-none focus:border-blue-500'
            )}
          />
          <button
            onClick={handleSendMessage}
            disabled={!inputValue.trim()}
            className={cn(
              'p-2 rounded-full transition-colors',
              inputValue.trim()
                ? 'bg-blue-500 hover:bg-blue-600'
                : 'bg-gray-600 cursor-not-allowed'
            )}
          >
            <Send className="w-5 h-5 text-white" />
          </button>
        </div>
      </div>
    </div>
  );
}
