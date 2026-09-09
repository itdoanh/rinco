/**
 * Extra unit tests for meeting-ui store and helpers.
 *
 * Run with:
 *     npx tsx frontend/meeting-ui/test/store_extra.test.ts
 */
import {
  useMeetingStore,
  stopMediaStream,
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

function makeStream(opts: { audio?: boolean; video?: boolean; stoppedAudio?: boolean; stoppedVideo?: boolean } = {}) {
  const stopped = { a: false, v: false };
  const tracks = [
    {
      kind: "audio",
      stop: () => {
        stopped.a = true;
      },
    },
    {
      kind: "video",
      stop: () => {
        stopped.v = true;
      },
    },
  ];
  const filtered = tracks.filter((t) => {
    if (t.kind === "audio") return opts.audio !== false;
    if (t.kind === "video") return opts.video !== false;
    return true;
  });
  return {
    s: {
      getTracks: () => filtered,
      getAudioTracks: () => filtered.filter((t) => t.kind === "audio"),
      getVideoTracks: () => filtered.filter((t) => t.kind === "video"),
    },
    stopped,
  };
}

// reset store before each test
const initialState = useMeetingStore.getState();
function reset(): void {
  useMeetingStore.setState({
    roomId: initialState.roomId,
    localStream: initialState.localStream,
    participants: initialState.participants,
    chatMessages: initialState.chatMessages,
    isMuted: initialState.isMuted,
    isVideoOn: initialState.isVideoOn,
    isScreenSharing: initialState.isScreenSharing,
    isRecording: initialState.isRecording,
    isConnected: initialState.isConnected,
  }, false);
}

// ---- stopMediaStream ----
test("stopMediaStream no-op on null", () => {
  // Should not throw
  stopMediaStream(null);
  assert(true, "null safe");
});

test("stopMediaStream stops all tracks", () => {
  const { s, stopped } = makeStream();
  stopMediaStream(s as unknown as MediaStream);
  assert(stopped.a && stopped.v, "both tracks stopped");
});

test("stopMediaStream handles track.stop throwing", () => {
  const s: any = {
    getTracks: () => [{ kind: "audio", stop: () => { throw new Error("err"); } }],
    getAudioTracks: () => [{ kind: "audio", stop: () => { throw new Error("err"); } }],
    getVideoTracks: () => [],
  };
  try {
    stopMediaStream(s);
    assert(true, "did not propagate");
  } catch (e) {
    failed++;
    console.error(`FAIL stopMediaStream threw: ${e}`);
  }
});

// ---- addParticipant ----
test("addParticipant adds new participant", () => {
  reset();
  useMeetingStore.getState().addParticipant({
    id: "p1",
    name: "Alice",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  const participants = useMeetingStore.getState().participants;
  assertEqual(participants.length, 1, "one participant");
});

test("addParticipant ignores duplicates", () => {
  reset();
  const fn = useMeetingStore.getState().addParticipant;
  fn({ id: "p1", name: "A", isMuted: false, isVideoOn: true, isScreenSharing: false, isSpeaking: false });
  fn({ id: "p1", name: "A2", isMuted: true, isVideoOn: true, isScreenSharing: false, isSpeaking: false });
  assertEqual(useMeetingStore.getState().participants.length, 1, "no dup");
});

// ---- removeParticipant ----
test("removeParticipant removes by id", () => {
  reset();
  const fn = useMeetingStore.getState().addParticipant;
  fn({ id: "p1", name: "A", isMuted: false, isVideoOn: true, isScreenSharing: false, isSpeaking: false });
  fn({ id: "p2", name: "B", isMuted: false, isVideoOn: true, isScreenSharing: false, isSpeaking: false });
  useMeetingStore.getState().removeParticipant("p1");
  const ps = useMeetingStore.getState().participants;
  assertEqual(ps.length, 1, "after remove");
  assertEqual(ps[0].id, "p2", "remaining");
});

test("removeParticipant no-op on missing", () => {
  reset();
  useMeetingStore.getState().addParticipant({
    id: "p1",
    name: "A",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  useMeetingStore.getState().removeParticipant("missing");
  assertEqual(useMeetingStore.getState().participants.length, 1, "still 1");
});

// ---- updateParticipant ----
test("updateParticipant applies patch", () => {
  reset();
  useMeetingStore.getState().addParticipant({
    id: "p1",
    name: "A",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  useMeetingStore.getState().updateParticipant("p1", { isMuted: true });
  const p = useMeetingStore.getState().participants[0];
  assertEqual(p.isMuted, true, "muted");
  assertEqual(p.name, "A", "name unchanged");
});

test("updateParticipant no-op on missing", () => {
  reset();
  useMeetingStore.getState().addParticipant({
    id: "p1",
    name: "A",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  useMeetingStore.getState().updateParticipant("missing", { isMuted: true });
  assertEqual(useMeetingStore.getState().participants[0].isMuted, false, "untouched");
});

// ---- toggleXxx actions ----
test("toggleMute flips", () => {
  reset();
  const before = useMeetingStore.getState().isMuted;
  useMeetingStore.getState().toggleMute();
  assertEqual(useMeetingStore.getState().isMuted, !before, "flipped");
});

test("toggleVideo flips", () => {
  reset();
  const before = useMeetingStore.getState().isVideoOn;
  useMeetingStore.getState().toggleVideo();
  assertEqual(useMeetingStore.getState().isVideoOn, !before, "flipped");
});

test("toggleScreenShare flips", () => {
  reset();
  const before = useMeetingStore.getState().isScreenSharing;
  useMeetingStore.getState().toggleScreenShare();
  assertEqual(useMeetingStore.getState().isScreenSharing, !before, "flipped");
});

test("toggleRecording flips", () => {
  reset();
  const before = useMeetingStore.getState().isRecording;
  useMeetingStore.getState().toggleRecording();
  assertEqual(useMeetingStore.getState().isRecording, !before, "flipped");
});

// ---- addChatMessage ----
test("addChatMessage adds a message", () => {
  reset();
  const msg: ChatMessage = {
    id: "m1",
    senderId: "p1",
    senderName: "Alice",
    content: "Hi",
    timestamp: new Date(),
  };
  useMeetingStore.getState().addChatMessage(msg);
  assertEqual(useMeetingStore.getState().chatMessages.length, 1, "1 msg");
});

test("addChatMessage appends multiple", () => {
  reset();
  for (let i = 0; i < 5; i++) {
    useMeetingStore.getState().addChatMessage({
      id: `m${i}`,
      senderId: "p1",
      senderName: "A",
      content: `msg-${i}`,
      timestamp: new Date(),
    });
  }
  assertEqual(useMeetingStore.getState().chatMessages.length, 5, "5 msgs");
});

// ---- setConnected / setRoomId / setLocalStream ----
test("setConnected updates state", () => {
  reset();
  useMeetingStore.getState().setConnected(true);
  assertEqual(useMeetingStore.getState().isConnected, true, "connected");
});

test("setRoomId updates state", () => {
  reset();
  useMeetingStore.getState().setRoomId("room-42");
  assertEqual(useMeetingStore.getState().roomId, "room-42", "room id");
});

test("setLocalStream updates state", () => {
  reset();
  const { s } = makeStream();
  useMeetingStore.getState().setLocalStream(s as unknown as MediaStream);
  assert(useMeetingStore.getState().localStream !== null, "stream set");
});

// ---- reset ----
test("reset clears state", () => {
  useMeetingStore.getState().setRoomId("room-x");
  useMeetingStore.getState().addParticipant({
    id: "p1",
    name: "A",
    isMuted: false,
    isVideoOn: true,
    isScreenSharing: false,
    isSpeaking: false,
  });
  useMeetingStore.getState().reset();
  const s = useMeetingStore.getState();
  // After reset should match initial State.
  assert(s.roomId !== "room-x", "room reset");
});

// ---- runner ----
(async () => {
  for (const t of tests) {
    console.log(`-- ${t.name}`);
    await t.fn();
  }
  console.log(`\nResults: ${passed} passed, ${failed} failed`);
  if (failed > 0) process.exit(1);
})();
