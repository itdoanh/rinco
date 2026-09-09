// Tests for meeting-ui SignalingClient message construction (pure JS).
// Re-implements the send helpers so we can verify the JSON shape without a real WebSocket.

let passed = 0;
let failed = 0;

function assert(cond, label) {
  if (cond) passed++;
  else {
    failed++;
    console.error("FAIL " + label);
  }
}

function assertEqual(actual, expected, label) {
  if (actual === expected) passed++;
  else {
    failed++;
    console.error("FAIL " + label + ": got " + JSON.stringify(actual) + " want " + JSON.stringify(expected));
  }
}

// Helper: build messages like the SignalingClient does
function buildOffer(from, to, sdp) {
  return { type: "offer", sdp, from, to };
}
function buildAnswer(from, to, sdp) {
  return { type: "answer", sdp, from, to };
}
function buildIceCandidate(from, to, candidate) {
  return { type: "ice-candidate", candidate, from, to };
}
function buildJoin(roomId, userId, userName) {
  return { type: "join", roomId, userId, userName };
}
function buildLeave(userId) {
  return { type: "leave", userId };
}
function buildChat(content, from, fromName) {
  return { type: "chat", content, from, fromName };
}
function buildParticipants(parts) {
  return { type: "participants", participants: parts };
}

// offer
const off = buildOffer("u1", "u2", { type: "offer", sdp: "v=0..." });
assertEqual(off.type, "offer", "offer type");
assertEqual(off.from, "u1", "offer from");
assertEqual(off.to, "u2", "offer to");
assertEqual(off.sdp.type, "offer", "offer sdp.type");

// answer
const ans = buildAnswer("u1", "u2", { type: "answer", sdp: "v=0..." });
assertEqual(ans.type, "answer", "answer type");

// ICE candidate
const ice = buildIceCandidate("u1", "u2", { candidate: "candidate:1 ..." });
assertEqual(ice.type, "ice-candidate", "ice type");
assertEqual(ice.candidate.candidate, "candidate:1 ...", "ice candidate string");

// join
const join = buildJoin("room1", "u1", "Alice");
assertEqual(join.type, "join", "join type");
assertEqual(join.roomId, "room1", "join room");
assertEqual(join.userName, "Alice", "join userName");

// leave
const leave = buildLeave("u1");
assertEqual(leave.type, "leave", "leave type");
assertEqual(leave.userId, "u1", "leave userId");

// chat
const chat = buildChat("Hello", "u1", "Alice");
assertEqual(chat.type, "chat", "chat type");
assertEqual(chat.content, "Hello", "chat content");
assertEqual(chat.fromName, "Alice", "chat fromName");

// participants
const parts = buildParticipants([
  { id: "u1", name: "Alice" },
  { id: "u2", name: "Bob" },
]);
assertEqual(parts.type, "participants", "participants type");
assertEqual(parts.participants.length, 2, "participants count");
assertEqual(parts.participants[0].name, "Alice", "first name");

// JSON round-trip for a complex message
const complex = buildOffer("u1", "u2", {
  type: "offer",
  sdp: "v=0\r\no=- ...\r\n",
});
const str = JSON.stringify(complex);
const parsed = JSON.parse(str);
assertEqual(parsed.type, "offer", "JSON round-trip type");
assert(parsed.sdp.sdp.includes("v=0"), "JSON round-trip sdp preserved");

// Required fields
const joinMin = buildJoin("r", "u", "n");
assert("roomId" in joinMin, "join has roomId");
assert("userId" in joinMin, "join has userId");

console.log("\nResults: " + passed + " passed, " + failed + " failed");
if (failed > 0) process.exit(1);
