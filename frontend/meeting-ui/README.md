# RINCO Meeting UI

WebRTC video conferencing frontend. Joins a room via WebSocket signaling, exchanges ICE candidates, and renders a multi-participant grid with chat and controls.

> Port: **3003** • Routes: `/`, `/meeting/[room_id]`

## Stack

- **Next.js 15** App Router
- **React 19** + TypeScript
- **@rinco/ui** shared components
- **Tailwind CSS 3**
- **WebRTC** (peer-to-peer mesh) + **WebSocket** signaling
- **zustand** meeting state (participants, local stream, chat)
- **TanStack Query**
- **Playwright** for E2E

## Setup

```bash
pnpm install
cp .env.example .env.local       # set NEXT_PUBLIC_SIGNALING_URL to your signaling server
```

## Dev workflow

```bash
pnpm dev            # next dev --port 3003 → http://localhost:3003
pnpm lint
pnpm typecheck
pnpm test:e2e
```

## Build

```bash
pnpm build
pnpm start          # production (port 3003)
```

## Project structure

```
meeting-ui/
├── app/
│   ├── layout.tsx                       # Root, dark theme
│   ├── globals.css
│   ├── page.tsx                         # "Create room" form
│   └── meeting/[room_id]/page.tsx       # In-meeting view (VideoGrid + ControlBar)
├── components/
│   ├── meeting/RoomJoin.tsx             # Pre-join preview + name entry
│   ├── video/{VideoGrid,VideoTile,ScreenShare}.tsx
│   ├── controls/{ControlBar,ParticipantList,ChatPanel}.tsx
│   └── ui/                              # Local wrappers
├── hooks/
│   ├── useWebRTC.ts                     # Peer connection lifecycle
│   └── useSignaling.ts                  # WebSocket signaling client
├── lib/
│   ├── webrtc.ts                        # createPeerConnection, getUserMedia, etc.
│   ├── signaling.ts                     # SignalingClient (WebSocket wrapper)
│   └── store.ts                         # Zustand meeting store
├── e2e/
└── playwright.config.ts
```

## How it works

```
                                ┌──────────────────────────┐
   Participant A                │   Signaling server        │
   ┌────────────┐  WebRTC P2P  │   (e.g. meeting-svc)      │
   │  A ↔ B,C   │◀────────────▶│   handles offers/answers  │
   │            │  WS signaling│   + ICE candidates        │
   └────────────┘              └──────────────────────────┘
   Participant B ─── WebRTC ───▶
                 ─── WS ───────▶
```

1. User opens `/meeting/{room_id}?name=...`
2. `useWebRTC` requests camera + mic, opens WebSocket to signaling server
3. On `join`, receives list of existing participants
4. For each peer, creates `RTCPeerConnection`, sends offer; receives answer, exchanges ICE candidates
5. State reflected in **Zustand store**, which `VideoGrid`, `ControlBar`, etc. consume

## Environment

See `.env.example`.

| Var                              | Purpose                                  |
|----------------------------------|------------------------------------------|
| `NEXT_PUBLIC_API_URL`            | Backend API base                         |
| `NEXT_PUBLIC_SIGNALING_URL`      | Signaling WebSocket URL (wss:// in prod) |
| `NEXT_PUBLIC_TURN_URL`           | TURN server (NAT traversal)              |
| `NEXT_PUBLIC_TURN_USERNAME`      | TURN username                            |
| `NEXT_PUBLIC_TURN_CREDENTIAL`    | TURN credential                          |
| `NEXT_PUBLIC_RECORDING_ENABLED`  | Show record button                       |

## Deployment

Because WebRTC needs direct peer connectivity and signaling, this app needs:

- HTTPS / WSS in production
- An open signaling endpoint (often a small WebSocket service or part of `meeting-svc`)
- TURN servers if clients are behind symmetric NATs

```bash
pnpm build
pnpm start
```
