/**
 * WebRTC utilities for peer-to-peer connections
 */

export interface RTCConfig {
  iceServers: RTCIceServer[];
}

export const defaultRTCConfig: RTCConfig = {
  iceServers: [
    { urls: "stun:stun.l.google.com:19302" },
    { urls: "stun:stun1.l.google.com:19302" },
  ],
};

/**
 * Create a peer connection
 */
export function createPeerConnection(
  config: RTCConfiguration = defaultRTCConfig.iceServers as RTCIceServer[]
): RTCPeerConnection {
  return new RTCPeerConnection({ iceServers: config });
}

/**
 * Add tracks from a MediaStream to a peer connection
 */
export function addTracksToConnection(
  pc: RTCPeerConnection,
  stream: MediaStream
): RTCRtpSender[] {
  return stream.getTracks().map((track) => pc.addTrack(track, stream));
}

/**
 * Create an offer for initiating a connection
 */
export async function createOffer(
  pc: RTCPeerConnection
): Promise<RTCSessionDescriptionInit> {
  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);
  return offer;
}

/**
 * Create an answer for responding to an offer
 */
export async function createAnswer(
  pc: RTCPeerConnection
): Promise<RTCSessionDescriptionInit> {
  const answer = await pc.createAnswer();
  await pc.setLocalDescription(answer);
  return answer;
}

/**
 * Set remote description from an offer/answer
 */
export async function setRemoteDescription(
  pc: RTCPeerConnection,
  description: RTCSessionDescriptionInit
): Promise<void> {
  await pc.setRemoteDescription(new RTCSessionDescription(description));
}

/**
 * Add ICE candidate
 */
export async function addIceCandidate(
  pc: RTCPeerConnection,
  candidate: RTCIceCandidateInit
): Promise<void> {
  await pc.addIceCandidate(new RTCIceCandidate(candidate));
}

/**
 * Get user media (camera and microphone)
 */
export async function getUserMedia(
  video: boolean = true,
  audio: boolean = true
): Promise<MediaStream> {
  return navigator.mediaDevices.getUserMedia({
    video: video
      ? {
          width: { ideal: 1280 },
          height: { ideal: 720 },
          facingMode: "user",
        }
      : false,
    audio: audio
      ? {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        }
      : false,
  });
}

/**
 * Get screen share stream
 */
export async function getDisplayMedia(): Promise<MediaStream> {
  return navigator.mediaDevices.getDisplayMedia({
    video: {
      cursor: "always",
    },
    audio: false,
  });
}

/**
 * Get available media devices
 */
export async function getMediaDevices(): Promise<{
  cameras: MediaDeviceInfo[];
  microphones: MediaDeviceInfo[];
  speakers: MediaDeviceInfo[];
}> {
  const devices = await navigator.mediaDevices.enumerateDevices();
  
  return {
    cameras: devices.filter((d) => d.kind === "videoinput"),
    microphones: devices.filter((d) => d.kind === "audioinput"),
    speakers: devices.filter((d) => d.kind === "audiooutput"),
  };
}

/**
 * Switch camera
 */
export async function switchCamera(
  stream: MediaStream,
  deviceId: string
): Promise<MediaStream> {
  const newStream = await navigator.mediaDevices.getUserMedia({
    video: { deviceId: { exact: deviceId } },
    audio: false,
  });

  // Replace video track
  const videoTrack = newStream.getVideoTracks()[0];
  const oldVideoTrack = stream.getVideoTracks()[0];
  
  if (oldVideoTrack) {
    stream.removeTrack(oldVideoTrack);
    oldVideoTrack.stop();
  }
  
  stream.addTrack(videoTrack);
  return stream;
}

/**
 * Switch microphone
 */
export async function switchMicrophone(
  stream: MediaStream,
  deviceId: string
): Promise<MediaStream> {
  const newStream = await navigator.mediaDevices.getUserMedia({
    video: false,
    audio: { deviceId: { exact: deviceId } },
  });

  // Replace audio track
  const audioTrack = newStream.getAudioTracks()[0];
  const oldAudioTrack = stream.getAudioTracks()[0];
  
  if (oldAudioTrack) {
    stream.removeTrack(oldAudioTrack);
    oldAudioTrack.stop();
  }
  
  stream.addTrack(audioTrack);
  return stream;
}

/**
 * Analyze audio for speaking detection
 */
export function createAudioAnalyzer(
  stream: MediaStream
): {
  analyser: AnalyserNode;
  isSpeaking: () => boolean;
  destroy: () => void;
} {
  const audioContext = new AudioContext();
  const source = audioContext.createMediaStreamSource(stream);
  const analyser = audioContext.createAnalyser();
  
  analyser.fftSize = 256;
  source.connect(analyser);
  
  const dataArray = new Uint8Array(analyser.frequencyBinCount);
  
  return {
    analyser,
    isSpeaking: () => {
      analyser.getByteFrequencyData(dataArray);
      const average = dataArray.reduce((a, b) => a + b) / dataArray.length;
      return average > 30; // Threshold for speaking
    },
    destroy: () => {
      audioContext.close();
    },
  };
}
