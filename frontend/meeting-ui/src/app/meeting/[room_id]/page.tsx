'use client';

import { use } from 'react';
import { VideoRoom } from '@/components/video/VideoRoom';

interface MeetingPageProps {
  params: Promise<{
    room_id: string;
  }>;
}

export default function MeetingPage({ params }: MeetingPageProps) {
  const { room_id } = use(params);

  return <VideoRoom roomId={room_id} />;
}
