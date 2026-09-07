# Meeting UI

WebRTC video meeting application for RINCO platform.

## Features

- **Pre-join screen**: Device selection and preview
- **Video grid**: Responsive grid for multiple participants
- **Controls**: Mute, camera, screen share, recording
- **Chat**: Real-time encrypted messaging
- **Participant list**: View all participants
- **Fullscreen mode**: Immersive viewing

## Tech Stack

- **Framework**: Next.js 15
- **Language**: TypeScript
- **State Management**: Zustand
- **Styling**: Tailwind CSS

## Getting Started

```bash
bun install
bun dev
```

Access at http://localhost:3003

## URL Structure

- `/` - Home/landing page
- `/meeting/[room_id]` - Meeting room

## WebRTC Flow

1. User joins pre-join page, selects devices
2. Get user media (camera + microphone)
3. Connect to signaling server (WebSocket)
4. Exchange SDP offers/answers via signaling
5. Establish P2P connections with ICE candidates
6. Stream media between participants

## Signaling Server

Requires a WebSocket signaling server at `NEXT_PUBLIC_SIGNALING_URL`.
