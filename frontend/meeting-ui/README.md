# Meeting UI

> **Frontend App #4** — Real-time video/audio meeting client powered by mediasoup-client + WebRTC.

## Stack

- **Bun** + **Next.js 15** (App Router, `app/meeting/[room_id]/page.tsx`)
- **TypeScript** 5.6+
- **Tailwind CSS** v4
- **mediasoup-client** v3 — browser WebRTC endpoint (SFU subscriber)
- **Zustand** — room state, participants, controls
- **TanStack Query** — signaling API calls
- **Framer Motion** — participant grid animations
- **shadcn/ui** via `@rinco/ui` shared component library

## Pages

| Route | File | Description |
|-------|------|-------------|
| `/` | `app/page.tsx` | Join/create room form |
| `/meeting/[room_id]` | `app/meeting/[room_id]/page.tsx` | Video room (full) |

## Features

- **WebRTC video/audio**: mediasoup-client connects to `webrtc-sfu`
- **Signaling**: polling + WebSocket via `GET/POST /api/signaling`
- **Participant grid**: responsive layout, active speaker highlighting
- **Controls**: mute/unmute, camera toggle, screen share, leave
- **Room state**: Zustand store tracks participants, media state, room config
- **Reconnection**: exponential backoff on signaling disconnect

## Environment Variables

```env
NEXT_PUBLIC_SFU_URL=wss://webrtc-sfu.example.com
NEXT_PUBLIC_SIGNALING_URL=/api/signaling
NEXT_PUBLIC_TURN_URL=https://turn.example.com
```

## Scripts

```bash
bun dev      # development server
bun build    # production build
bun start    # production server
bun lint     # ESLint
```
