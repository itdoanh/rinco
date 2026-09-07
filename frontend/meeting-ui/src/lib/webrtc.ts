import axios from 'axios';

const ICE_SERVERS: RTCIceServer[] = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun1.l.google.com:19302' },
  { urls: 'stun:stun2.l.google.com:19302' },
];

export interface PeerConnectionOptions {
  onTrack?: (stream: MediaStream) => void;
  onIceCandidate?: (candidate: RTCIceCandidate) => void;
  onConnectionStateChange?: (state: RTCPeerConnectionState) => void;
  onIceConnectionStateChange?: (state: RTCIceConnectionState) => void;
}

export class PeerConnection {
  private pc: RTCPeerConnection;
  private remoteStream: MediaStream;
  private options: PeerConnectionOptions;

  constructor(options: PeerConnectionOptions = {}) {
    this.options = options;
    this.remoteStream = new MediaStream();

    this.pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });

    // Set up event handlers
    this.pc.ontrack = (event) => {
      event.streams[0].getTracks().forEach((track) => {
        this.remoteStream.addTrack(track);
      });
      this.options.onTrack?.(this.remoteStream);
    };

    this.pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.options.onIceCandidate?.(event.candidate);
      }
    };

    this.pc.onconnectionstatechange = () => {
      this.options.onConnectionStateChange?.(this.pc.connectionState);
    };

    this.pc.oniceconnectionstatechange = () => {
      this.options.onIceConnectionStateChange?.(this.pc.iceConnectionState);
    };
  }

  async addTrack(track: MediaStreamTrack, stream: MediaStream): Promise<RTCRtpSender> {
    return this.pc.addTrack(track, stream);
  }

  async createOffer(): Promise<RTCSessionDescriptionInit> {
    const offer = await this.pc.createOffer();
    await this.pc.setLocalDescription(offer);
    return offer;
  }

  async createAnswer(offer: RTCSessionDescriptionInit): Promise<RTCSessionDescriptionInit> {
    await this.pc.setRemoteDescription(new RTCSessionDescription(offer));
    const answer = await this.pc.createAnswer();
    await this.pc.setLocalDescription(answer);
    return answer;
  }

  async handleAnswer(answer: RTCSessionDescriptionInit): Promise<void> {
    await this.pc.setRemoteDescription(new RTCSessionDescription(answer));
  }

  async addIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    await this.pc.addIceCandidate(new RTCIceCandidate(candidate));
  }

  getRemoteStream(): MediaStream {
    return this.remoteStream;
  }

  close(): void {
    this.pc.close();
  }

  get connectionState(): RTCPeerConnectionState {
    return this.pc.connectionState;
  }

  get iceConnectionState(): RTCIceConnectionState {
    return this.pc.iceConnectionState;
  }
}

// Helper function to get media stream
export async function getLocalStream(constraints?: MediaStreamConstraints): Promise<MediaStream> {
  const defaultConstraints: MediaStreamConstraints = {
    audio: true,
    video: {
      width: { ideal: 1280 },
      height: { ideal: 720 },
      facingMode: 'user',
    },
  };

  return navigator.mediaDevices.getUserMedia(constraints || defaultConstraints);
}

// Helper function to get screen share stream
export async function getScreenStream(): Promise<MediaStream> {
  return navigator.mediaDevices.getDisplayMedia({
    video: true,
    audio: false,
  });
}

// Helper function to replace track
export async function replaceTrack(
  sender: RTCRtpSender,
  newTrack: MediaStreamTrack
): Promise<void> {
  await sender.replaceTrack(newTrack);
}

// Helper function to stop all tracks in a stream
export function stopStream(stream: MediaStream | null): void {
  if (stream) {
    stream.getTracks().forEach((track) => track.stop());
  }
}
