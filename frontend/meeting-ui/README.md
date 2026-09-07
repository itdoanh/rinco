# Meeting UI

Video conferencing interface for RINCO Platform with WebRTC support.

## Features

- WebRTC peer-to-peer video/audio
- WebSocket signaling
- Screen sharing
- Real-time chat
- Participant management
- Responsive video grid

## Tech Stack

- Next.js 15
- TypeScript
- Tailwind CSS
- Zustand (state management)
- simple-peer (WebRTC abstraction)

## Setup

```bash
npm install
npm run dev
```

## Environment Variables

```
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_SIGNALING_URL=ws://localhost:8080/ws
```

## API Endpoints

- `GET /api/signaling` - Get signaling status
- `POST /api/signaling` - Send signaling message
- `GET /api/room` - Get room info
- `POST /api/room` - Create room
- `DELETE /api/room` - Delete room
