import { useEffect, useRef, useCallback, useState } from 'react';
import { useRoomStore, type Participant } from '@/store/room';

export interface WebRTCConfig {
  signalingUrl: string;
  roomId: string;
  userId: string;
  userName: string;
}

export interface UseWebRTCReturn {
  localStream: MediaStream | null;
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
  participants: Participant[];
  toggleAudio: () => void;
  toggleVideo: () => void;
  startScreenShare: () => Promise<void>;
  stopScreenShare: () => void;
  disconnect: () => void;
}

export function useWebRTC(config: WebRTCConfig): UseWebRTCReturn {
  const {
    roomId,
    userId,
    userName,
    signalingUrl,
  } = config;

  const wsRef = useRef<WebSocket | null>(null);
  const peerConnectionsRef = useRef<Map<string, RTCPeerConnection>>(new Map());
  const localStreamRef = useRef<MediaStream | null>(null);
  const screenStreamRef = useRef<MediaStream | null>(null);

  const {
    localStream,
    setLocalStream,
    participants,
    isConnected,
    setConnected,
    isConnecting,
    setConnecting,
    error,
    setError,
    addParticipant,
    removeParticipant,
    updateParticipant,
    reset,
  } = useRoomStore();

  const [isAudioEnabled, setIsAudioEnabled] = useState(true);
  const [isVideoEnabled, setIsVideoEnabled] = useState(true);

  // Initialize local media
  const initializeMedia = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: true,
        video: {
          width: { ideal: 1280 },
          height: { ideal: 720 },
          facingMode: 'user',
        },
      });

      localStreamRef.current = stream;
      setLocalStream(stream);
      return stream;
    } catch (err) {
      console.error('Failed to get media devices:', err);
      setError('Failed to access camera/microphone');
      throw err;
    }
  }, [setLocalStream, setError]);

  // Create peer connection
  const createPeerConnection = useCallback(
    (peerId: string): RTCPeerConnection => {
      const pc = new RTCPeerConnection({
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: 'stun:stun1.l.google.com:19302' },
        ],
      });

      // Add local tracks
      if (localStreamRef.current) {
        localStreamRef.current.getTracks().forEach((track) => {
          pc.addTrack(track, localStreamRef.current!);
        });
      }

      // Handle remote tracks
      pc.ontrack = (event) => {
        const [remoteStream] = event.streams;
        const existingParticipant = participants.get(peerId);

        if (existingParticipant) {
          updateParticipant(peerId, { stream: remoteStream });
        } else {
          addParticipant({
            id: peerId,
            name: peerId,
            stream: remoteStream,
            isMuted: false,
            isVideoOff: false,
            isScreenSharing: false,
            isSpeaking: false,
            joinedAt: new Date(),
          });
        }
      };

      // Handle ICE candidates
      pc.onicecandidate = (event) => {
        if (event.candidate && wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(
            JSON.stringify({
              type: 'ice_candidate',
              to: peerId,
              candidate: event.candidate,
            })
          );
        }
      };

      // Handle connection state changes
      pc.onconnectionstatechange = () => {
        console.log(`Connection state with ${peerId}:`, pc.connectionState);
        if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
          removeParticipant(peerId);
          peerConnectionsRef.current.delete(peerId);
        }
      };

      peerConnectionsRef.current.set(peerId, pc);
      return pc;
    },
    [participants, addParticipant, updateParticipant, removeParticipant]
  );

  // Handle incoming messages
  const handleMessage = useCallback(
    async (event: MessageEvent) => {
      try {
        const message = JSON.parse(event.data);
        const { type, from, data } = message;

        switch (type) {
          case 'offer': {
            const pc = createPeerConnection(from);
            await pc.setRemoteDescription(new RTCSessionDescription(data));
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);

            wsRef.current?.send(
              JSON.stringify({
                type: 'answer',
                to: from,
                data: answer,
              })
            );
            break;
          }

          case 'answer': {
            const pc = peerConnectionsRef.current.get(from);
            if (pc) {
              await pc.setRemoteDescription(new RTCSessionDescription(data));
            }
            break;
          }

          case 'ice_candidate': {
            const pc = peerConnectionsRef.current.get(from);
            if (pc && data) {
              await pc.addIceCandidate(new RTCIceCandidate(data));
            }
            break;
          }

          case 'user_joined': {
            // Create offer for new user
            const pc = createPeerConnection(from);
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);

            wsRef.current?.send(
              JSON.stringify({
                type: 'offer',
                to: from,
                data: offer,
              })
            );

            addParticipant({
              id: from,
              name: data.name || from,
              isMuted: false,
              isVideoOff: false,
              isScreenSharing: false,
              isSpeaking: false,
              joinedAt: new Date(),
            });
            break;
          }

          case 'user_left': {
            const pc = peerConnectionsRef.current.get(from);
            if (pc) {
              pc.close();
              peerConnectionsRef.current.delete(from);
            }
            removeParticipant(from);
            break;
          }

          case 'toggle_audio': {
            updateParticipant(from, { isMuted: data.enabled });
            break;
          }

          case 'toggle_video': {
            updateParticipant(from, { isVideoOff: data.enabled === false });
            break;
          }

          case 'screen_share_started': {
            updateParticipant(from, { isScreenSharing: true });
            break;
          }

          case 'screen_share_stopped': {
            updateParticipant(from, { isScreenSharing: false });
            break;
          }
        }
      } catch (err) {
        console.error('Failed to handle message:', err);
      }
    },
    [createPeerConnection, addParticipant, removeParticipant, updateParticipant]
  );

  // Connect to signaling server
  const connect = useCallback(async () => {
    setConnecting(true);
    setError(null);

    try {
      // Initialize media
      await initializeMedia();

      // Connect to WebSocket
      const wsUrl = `${signalingUrl}?room_id=${roomId}&user_id=${userId}&user_name=${encodeURIComponent(userName)}`;
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('Connected to signaling server');
        setConnected(true);
        setConnecting(false);

        // Send join message
        ws.send(
          JSON.stringify({
            type: 'join',
            room_id: roomId,
            user_id: userId,
            user_name: userName,
          })
        );
      };

      ws.onmessage = handleMessage;

      ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        setError('Connection error');
      };

      ws.onclose = () => {
        console.log('Disconnected from signaling server');
        setConnected(false);
        setConnecting(false);
      };

      wsRef.current = ws;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect');
      setConnecting(false);
    }
  }, [
    roomId,
    userId,
    userName,
    signalingUrl,
    initializeMedia,
    handleMessage,
    setConnecting,
    setError,
    setConnected,
  ]);

  // Toggle audio
  const toggleAudio = useCallback(() => {
    if (localStreamRef.current) {
      const audioTrack = localStreamRef.current.getAudioTracks()[0];
      if (audioTrack) {
        audioTrack.enabled = !audioTrack.enabled;
        setIsAudioEnabled(audioTrack.enabled);

        wsRef.current?.send(
          JSON.stringify({
            type: 'toggle_audio',
            data: { enabled: audioTrack.enabled },
          })
        );
      }
    }
  }, []);

  // Toggle video
  const toggleVideo = useCallback(() => {
    if (localStreamRef.current) {
      const videoTrack = localStreamRef.current.getVideoTracks()[0];
      if (videoTrack) {
        videoTrack.enabled = !videoTrack.enabled;
        setIsVideoEnabled(videoTrack.enabled);

        wsRef.current?.send(
          JSON.stringify({
            type: 'toggle_video',
            data: { enabled: videoTrack.enabled },
          })
        );
      }
    }
  }, []);

  // Start screen share
  const startScreenShare = useCallback(async () => {
    try {
      const screenStream = await navigator.mediaDevices.getDisplayMedia({
        video: true,
        audio: false,
      });

      screenStreamRef.current = screenStream;

      // Replace video track for all peers
      const videoTrack = screenStream.getVideoTracks()[0];
      peerConnectionsRef.current.forEach((pc) => {
        const sender = pc.getSenders().find((s) => s.track?.kind === 'video');
        if (sender) {
          sender.replaceTrack(videoTrack);
        }
      });

      // Handle screen share stop
      videoTrack.onended = () => {
        stopScreenShare();
      };

      wsRef.current?.send(
        JSON.stringify({
          type: 'screen_share_started',
        })
      );
    } catch (err) {
      console.error('Failed to start screen share:', err);
    }
  }, []);

  // Stop screen share
  const stopScreenShare = useCallback(() => {
    if (screenStreamRef.current) {
      screenStreamRef.current.getTracks().forEach((track) => track.stop());
      screenStreamRef.current = null;
    }

    // Restore camera track for all peers
    if (localStreamRef.current) {
      const videoTrack = localStreamRef.current.getVideoTracks()[0];
      peerConnectionsRef.current.forEach((pc) => {
        const sender = pc.getSenders().find((s) => s.track?.kind === 'video');
        if (sender && videoTrack) {
          sender.replaceTrack(videoTrack);
        }
      });
    }

    wsRef.current?.send(
      JSON.stringify({
        type: 'screen_share_stopped',
      })
    );
  }, []);

  // Disconnect
  const disconnect = useCallback(() => {
    // Close all peer connections
    peerConnectionsRef.current.forEach((pc) => pc.close());
    peerConnectionsRef.current.clear();

    // Close WebSocket
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    // Stop local stream
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach((track) => track.stop());
      localStreamRef.current = null;
    }

    // Stop screen stream
    if (screenStreamRef.current) {
      screenStreamRef.current.getTracks().forEach((track) => track.stop());
      screenStreamRef.current = null;
    }

    reset();
  }, [reset]);

  // Connect on mount
  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    localStream: localStreamRef.current,
    isConnected,
    isConnecting,
    error,
    participants: Array.from(participants.values()),
    toggleAudio,
    toggleVideo,
    startScreenShare,
    stopScreenShare,
    disconnect,
  };
}
