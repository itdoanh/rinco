/**
 * Unit tests for the meeting-ui store.
 *
 * Run from frontend/landing with:
 *     npx tsx ../../frontend/meeting-ui/test/store.test.ts
 */
import {
  useMeetingStore,
  stopMediaStream,
  type Participant,
  type ChatMessage,
} from "../lib/store";

// ---------- tiny test runner ----------
let passed = 0;
let failed = 0;
function assert(cond: unknown, label: string): void {
  if (cond) {
    passed++;
  } else {
    failed++;
    console.error(`FAIL ${label}`);
  }
}
function assertEqual<T>(actual: T, expected: T, label: string): void {
  if (actual === expected) {
    passed++;
  } else {
    failed++;
    console.error(
      `FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(
        expected,
      )}`,
    );
  }
}
const tests: Array<{ name: string; fn: () => void | Promise<void> }> = [];
function test(name: string, fn: () => void | Promise<void>): void {
  tests.push({ name, fn });
}

/** Build a fake MediaStream with two tracks. */
function fakeStream(): MediaStream & {
  __tracks: { stop: ReturnType<typeof vi> };
} {
  const tracks = [
    { kind: "audio", stop: () => ((tracks[0] as any).__stopped = true) },
    { kind: "video", stop: () => ((tracks[1] as any).__stopped = true) },
  ];
  const stream: any = {
    getTracks: () => tracks,
    getAudioTracks: () => tracks.filter((t) => t.kind === "audio"),
    getVideoTracks: () => tracks.filter((t) => t.kind === "video"),
  };
  return stream;
}
// Mock helper (just for the type).
function vi(): () => void {
  return () => undefined;
}

// ---------- tests ----------

test("initial state", () => {
  const s = useMeetingStore.getState();
  assertEqual(s.isMuted, false, "isMuted");
  assertEqual(s.isVideoOn, true, "isVideoOn default true");
  assertEqual(s.participants.length, 0, "no participants");
  assertEqual(s.chatMessages.length, 0, "no messages");
  assertEqual(s.localStream, null, "no stream");
});

test("stopMediaStream no-op on null", () => {
  // Should not throw.
  stopMediaStream(null);
  passed++;
});

test("stopMediaStream stops every track", () => {
  const stream = fakeStream();
  stopMediaStream(stream as unknown as MediaStream);
  // Tracks should have been mutated.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  assertEqual((stream.getTracks()[0] as any).__stopped, true, "audio stopped");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  assertEqual((stream.getTracks()[1] as any).__stopped, true, "video stopped");
});

test("toggleMute flips isMuted and flips track.enabled", () => {
  // Reset to known state
  useMeetingStore.getState().reset();
  const stream = fakeStream();
  // Pre-arm the audio track with enabled = true
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (stream.getAudioTracks()[0] as any).enabled = true;
  useMeetingStore.getState().setLocalStream(stream as unknown as MediaStream);
  useMeetingStore.getState().toggleMute();
  assertEqual(useMeetingStore.getState().isMuted, true, "muted");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  assertEqual(
    (stream.getAudioTracks()[0] as any).enabled,
    false,
    "track disabled",
  );
  useMeetingStore.getState().toggleMute();
  assertEqual(useMeetingStore.getState().isMuted, false, "unmuted");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  assertEqual(
    (stream.getAudioTracks()[0] as any).enabled,
    true,
    "track enabled",
  );
});

test("toggleVideo flips isVideoOn and flips track.enabled", () => {
  useMeetingStore.getState().reset();
  const stream = fakeStream();
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (stream.getVideoTracks()[0] as any).enabled = true;
  useMeetingStore.getState().setLocalStream(stream as unknown as MediaStream);
  useMeetingStore.getState().toggleVideo();
  assertEqual(useMeetingStore.getState().isVideoOn, false, "video off");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  assertEqual(
    (stream.getVideoTracks()[0] as any).enabled,
    false,
    "video track disabled",
  );
});

test("toggleMute is a no-op when no stream is set", () => {
  useMeetingStore.getState().reset();
  useMeetingStore.getState().toggleMute();
  assertEqual(useMeetingStore.getState().isMuted, true, "still toggles state");
});

test("addParticipant dedupes by id", () => {
  useMeetingStore.getState().reset();
  const p: Participant = {
    id: "u1",
    name: "Alice",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  };
  useMeetingStore.getState().addParticipant(p);
  useMeetingStore.getState().addParticipant(p);
  assertEqual(useMeetingStore.getState().participants.length, 1, "1 unique");
});

test("updateParticipant partial-merge", () => {
  useMeetingStore.getState().reset();
  useMeetingStore.getState().addParticipant({
    id: "u1",
    name: "Alice",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  useMeetingStore.getState().updateParticipant("u1", { isMuted: true });
  const p = useMeetingStore.getState().participants[0];
  assertEqual(p.isMuted, true, "muted");
  assertEqual(p.name, "Alice", "name unchanged");
});

test("removeParticipant filters by id", () => {
  useMeetingStore.getState().reset();
  useMeetingStore.getState().addParticipant({
    id: "u1",
    name: "Alice",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  useMeetingStore.getState().addParticipant({
    id: "u2",
    name: "Bob",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  useMeetingStore.getState().removeParticipant("u1");
  const remaining = useMeetingStore.getState().participants;
  assertEqual(remaining.length, 1, "1 left");
  assertEqual(remaining[0].id, "u2", "u2 remains");
});

test("addChatMessage dedupes by id", () => {
  useMeetingStore.getState().reset();
  const msg: ChatMessage = {
    id: "m1",
    senderId: "u1",
    senderName: "Alice",
    content: "Hi",
    timestamp: new Date(),
  };
  useMeetingStore.getState().addChatMessage(msg);
  useMeetingStore.getState().addChatMessage(msg);
  assertEqual(useMeetingStore.getState().chatMessages.length, 1, "1 unique");
});

test("reset clears state and stops stream tracks", () => {
  const stream = fakeStream();
  useMeetingStore.getState().setLocalStream(stream as unknown as MediaStream);
  useMeetingStore.getState().setRoomId("r1");
  useMeetingStore.getState().reset();
  assertEqual(useMeetingStore.getState().roomId, null, "room cleared");
  assertEqual(useMeetingStore.getState().localStream, null, "stream cleared");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  assertEqual((stream.getTracks()[0] as any).__stopped, true, "stopped");
});

// ---------- runner ----------
(async () => {
  for (const t of tests) {
    console.log(`-- ${t.name}`);
    await t.fn();
  }
  console.log(`\nResults: ${passed} passed, ${failed} failed`);
  if (failed > 0) process.exit(1);
})();
