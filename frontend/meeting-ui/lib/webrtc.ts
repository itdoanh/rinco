/**
 * WebRTC utilities for peer-to-peer connections.
 *
 * Pure wrappers around the browser ``RTCPeerConnection`` API.  These
 * helpers are intentionally framework-agnostic so they can be unit
 * tested with a mock implementation.
 */

export const defaultRTCConfig: RTCConfiguration = {
  iceServers: [
    { urls: "stun:stun.l.google.com:19302" },
    { urls: "stun:stun1.l.google.com:19302" },
  ],
};

/**
 * Create a peer connection using the provided configuration.
 *
 * Defaults to :data:`defaultRTCConfig` when ``config`` is omitted.
 */
export function createPeerConnection(
  config: RTCConfiguration = defaultRTCConfig,
): RTCPeerConnection {
  return new RTCPeerConnection(config);
}

/**
 * Add tracks from a ``MediaStream`` to a peer connection.  Returns the
 * array of RTCRtpSender objects so callers can later replace individual
 * tracks without renegotiating the whole connection.
 */
export function addTracksToConnection(
  pc: RTCPeerConnection,
  stream: MediaStream,
): RTCRtpSender[] {
  return stream.getTracks().map((track) => pc.addTrack(track, stream));
}

/**
 * Replace the track of a given kind on a peer connection.  Returns
 * ``true`` if the sender was found and updated, ``false`` otherwise.
 *
 * Useful for camera/microphone hot-swapping — without this the remote
 * peer would continue seeing the old track until renegotiation.
 */
export async function replaceTrack(
  pc: RTCPeerConnection,
  kind: "audio" | "video",
  newTrack: MediaStreamTrack,
): Promise<boolean> {
  const sender = pc.getSenders().find((s) => s.track?.kind === kind);
  if (!sender) return false;
  await sender.replaceTrack(newTrack);
  return true;
}

/**
 * Create an offer and set it as the local description in one step.
 */
export async function createOffer(
  pc: RTCPeerConnection,
): Promise<RTCSessionDescriptionInit> {
  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);
  return offer;
}

/**
 * Create an answer and set it as the local description in one step.
 */
export async function createAnswer(
  pc: RTCPeerConnection,
): Promise<RTCSessionDescriptionInit> {
  const answer = await pc.createAnswer();
  await pc.setLocalDescription(answer);
  return answer;
}

/**
 * Apply a remote description.  Wraps ``RTCSessionDescription``
 * construction so callers can pass a plain ``RTCSessionDescriptionInit``
 * payload.
 */
export async function setRemoteDescription(
  pc: RTCPeerConnection,
  description: RTCSessionDescriptionInit,
): Promise<void> {
  await pc.setRemoteDescription(new RTCSessionDescription(description));
}

/**
 * Add an ICE candidate to a peer connection.  Silently no-ops when the
 * candidate is null (which is the standard ``null`` sentinel that
 * browsers emit at the end of candidate gathering).
 */
export async function addIceCandidate(
  pc: RTCPeerConnection,
  candidate: RTCIceCandidateInit | null,
): Promise<void> {
  if (!candidate) return;
  await pc.addIceCandidate(new RTCIceCandidate(candidate));
}

/**
 * Acquire a local microphone + camera stream.  Pass ``false`` to skip
 * the video or audio track.
 */
export async function getUserMedia(
  video: boolean = true,
  audio: boolean = true,
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
 * Acquire a screen-share stream.  Audio is excluded by default — pass
 * ``audio: true`` if you need to share application audio too.
 */
export async function getDisplayMedia(
  options: { audio?: boolean; cursor?: "always" | "motion" | "never" } = {},
): Promise<MediaStream> {
  return navigator.mediaDevices.getDisplayMedia({
    video: {
      cursor: options.cursor ?? "always",
    },
    audio: options.audio ?? false,
  });
}

/**
 * Enumerate the available input/output devices.  Useful for the
 * "Settings → Devices" panel.
 */
export async function getMediaDevices(): Promise<{
  cameras: MediaDeviceInfo[];
  microphones: MediaDeviceInfo[];
  speakers: MediaDeviceInfo[];
}> {
  if (!navigator.mediaDevices?.enumerateDevices) {
    return { cameras: [], microphones: [], speakers: [] };
  }
  const devices = await navigator.mediaDevices.enumerateDevices();
  return {
    cameras: devices.filter((d) => d.kind === "videoinput"),
    microphones: devices.filter((d) => d.kind === "audioinput"),
    speakers: devices.filter((d) => d.kind === "audiooutput"),
  };
}

/**
 * Build an audio analyser for VU/speaking detection.  Returns an
 * analyser node plus a synchronous ``isSpeaking()`` helper that
 * computes the average byte frequency each time it is called.
 *
 * The caller is responsible for calling ``destroy()`` when the analyser
 * is no longer needed to release the AudioContext.
 */
export function createAudioAnalyzer(
  stream: MediaStream,
  options: { fftSize?: number; speakingThreshold?: number } = {},
): {
  analyser: AnalyserNode;
  isSpeaking: () => boolean;
  destroy: () => void;
} {
  const fftSize = options.fftSize ?? 256;
  const threshold = options.speakingThreshold ?? 30;

  const AudioContextCtor: typeof AudioContext =
    (globalThis as unknown as { AudioContext?: typeof AudioContext })
      .AudioContext ??
    (globalThis as unknown as { webkitAudioContext?: typeof AudioContext })
      .webkitAudioContext;

  if (!AudioContextCtor) {
    throw new Error("AudioContext not supported in this runtime");
  }

  const audioContext = new AudioContextCtor();
  const source = audioContext.createMediaStreamSource(stream);
  const analyser = audioContext.createAnalyser();
  analyser.fftSize = fftSize;
  source.connect(analyser);

  const dataArray = new Uint8Array(analyser.frequencyBinCount);
  let destroyed = false;

  return {
    analyser,
    isSpeaking: () => {
      if (destroyed) return false;
      analyser.getByteFrequencyData(dataArray);
      let total = 0;
      for (let i = 0; i < dataArray.length; i++) total += dataArray[i];
      const average = dataArray.length === 0 ? 0 : total / dataArray.length;
      return average > threshold;
    },
    destroy: () => {
      if (destroyed) return;
      destroyed = true;
      try {
        source.disconnect();
      } catch {
        /* ignore — already disconnected */
      }
      try {
        analyser.disconnect();
      } catch {
        /* ignore */
      }
      audioContext.close().catch(() => {
        /* ignore */
      });
    },
  };
}
