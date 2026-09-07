"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useMeetingStore, type Participant } from "@/lib/store";
import { getUserMedia, createPeerConnection, addTracksToConnection } from "@/lib/webrtc";
import { SignalingClient, type SignalingMessage } from "@/lib/signaling";

const SIGNALING_URL = process.env.NEXT_PUBLIC_SIGNALING_URL || "ws://localhost:8081";

interface UseWebRTCOptions {
  roomId: string;
  userId: string;
  userName: string;
}

export function useWebRTC({ roomId, userId, userName }: UseWebRTCOptions) {
  const [isConnecting, setIsConnecting] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  const signalingRef = useRef<SignalingClient | null>(null);
  const peerConnectionsRef = useRef<Map<string, RTCPeerConnection>>(new Map());
  
  const store = useMeetingStore();

  const handleSignalingMessage = useCallback(
    async (message: SignalingMessage) => {
      switch (message.type) {
        case "offer":
          await handleOffer(message);
          break;
        case "answer":
          await handleAnswer(message);
          break;
        case "ice-candidate":
          await handleIceCandidate(message);
          break;
        case "join":
          // New participant joined, create offer
          if (message.userId !== userId) {
            createPeerForParticipant(message.userId, message.userName);
          }
          break;
        case "leave":
          handleParticipantLeave(message.userId);
          break;
        case "participants":
          // Existing participants in room
          message.participants.forEach((p) => {
            if (p.id !== userId) {
              createPeerForParticipant(p.id, p.name);
            }
          });
          break;
        case "chat":
          store.addChatMessage({
            id: Date.now().toString(),
            senderId: message.from,
            senderName: message.fromName,
            content: message.content,
            timestamp: new Date(),
          });
          break;
      }
    },
    [userId, roomId, userName]
  );

  const handleOffer = async (message: Extract<SignalingMessage, { type: "offer" }>) => {
    const pc = peerConnectionsRef.current.get(message.from);
    if (!pc) return;

    await pc.setRemoteDescription(message.sdp);
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    
    signalingRef.current?.sendAnswer(message.from, answer);
  };

  const handleAnswer = async (message: Extract<SignalingMessage, { type: "answer" }>) => {
    const pc = peerConnectionsRef.current.get(message.from);
    if (!pc) return;

    await pc.setRemoteDescription(message.sdp);
  };

  const handleIceCandidate = async (message: Extract<SignalingMessage, { type: "ice-candidate" }>) => {
    const pc = peerConnectionsRef.current.get(message.from);
    if (!pc) return;

    await pc.addIceCandidate(message.candidate);
  };

  const createPeerForParticipant = async (participantId: string, name: string) => {
    if (peerConnectionsRef.current.has(participantId)) return;

    const pc = createPeerConnection();
    peerConnectionsRef.current.set(participantId, pc);

    // Add local tracks
    if (store.localStream) {
      addTracksToConnection(pc, store.localStream);
    }

    // Handle incoming tracks
    pc.ontrack = (event) => {
      store.updateParticipant(participantId, {
        stream: event.streams[0],
      });
    };

    // Handle ICE candidates
    pc.onicecandidate = (event) => {
      if (event.candidate) {
        signalingRef.current?.sendIceCandidate(participantId, event.candidate.toJSON());
      }
    };

    // Add participant to store
    store.addParticipant({
      id: participantId,
      name,
      isMuted: false,
      isVideoOn: true,
      isScreenSharing: false,
      isSpeaking: false,
    });

    // Create and send offer
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    signalingRef.current?.sendOffer(participantId, offer);
  };

  const handleParticipantLeave = (participantId: string) => {
    const pc = peerConnectionsRef.current.get(participantId);
    if (pc) {
      pc.close();
      peerConnectionsRef.current.delete(participantId);
    }
    store.removeParticipant(participantId);
  };

  useEffect(() => {
    const init = async () => {
      try {
        // Get user media
        const stream = await getUserMedia(true, true);
        store.setLocalStream(stream);
        store.setRoomId(roomId);

        // Connect to signaling server
        signalingRef.current = new SignalingClient({
          url: SIGNALING_URL,
          roomId,
          userId,
          userName,
          onMessage: handleSignalingMessage,
          onOpen: () => {
            store.setConnected(true);
            setIsConnecting(false);
          },
          onClose: () => {
            store.setConnected(false);
          },
          onError: (err) => {
            console.error("Signaling error:", err);
            setError("Connection failed");
            setIsConnecting(false);
          },
        });

        signalingRef.current.connect();
      } catch (err) {
        console.error("Failed to initialize:", err);
        setError("Failed to access camera/microphone");
        setIsConnecting(false);
      }
    };

    init();

    return () => {
      // Cleanup
      peerConnectionsRef.current.forEach((pc) => pc.close());
      peerConnectionsRef.current.clear();
      signalingRef.current?.leave();
      store.localStream?.getTracks().forEach((track) => track.stop());
      store.reset();
    };
  }, [roomId, userId, userName]);

  return {
    isConnecting,
    error,
  };
}

export default useWebRTC;
