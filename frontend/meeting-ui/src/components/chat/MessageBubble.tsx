'use client';

import { cn } from '@/lib/utils';

interface Message {
  id: string;
  userId: string;
  userName: string;
  content: string;
  timestamp: Date;
}

interface MessageBubbleProps {
  message: Message;
  isOwnMessage: boolean;
}

export function MessageBubble({ message, isOwnMessage }: MessageBubbleProps) {
  const isSystem = message.userId === 'system';

  if (isSystem) {
    return (
      <div className="text-center">
        <span className="text-xs text-gray-500">
          {message.content} • {formatTime(message.timestamp)}
        </span>
      </div>
    );
  }

  return (
    <div
      className={cn(
        'flex flex-col',
        isOwnMessage ? 'items-end' : 'items-start'
      )}
    >
      <div
        className={cn(
          'max-w-[80%] px-4 py-2 rounded-2xl',
          isOwnMessage
            ? 'bg-blue-500 text-white rounded-br-md'
            : 'bg-gray-700 text-white rounded-bl-md'
        )}
      >
        {!isOwnMessage && (
          <div className="text-xs text-gray-400 mb-1">{message.userName}</div>
        )}
        <p className="text-sm">{message.content}</p>
      </div>
      <span className="text-xs text-gray-500 mt-1">
        {formatTime(message.timestamp)}
      </span>
    </div>
  );
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}
