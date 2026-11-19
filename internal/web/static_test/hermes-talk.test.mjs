import test from "node:test";
import assert from "node:assert/strict";
import { classify, normalize, createTalk } from "../static/hermes-talk.js";

test("classify ignores silence and Whisper phantoms, honours a bare stop", () => {
  assert.equal(classify("   "), "empty");
  assert.equal(classify("[BLANK_AUDIO]"), "empty");
  assert.equal(classify("Thanks for watching!", 900), "phantom");
  assert.equal(classify("Thanks for watching my friend, did you see it", 900), "say");
  assert.equal(classify("Stop."), "stop");
  assert.equal(classify("stop doing that and try this instead"), "say");
  assert.equal(normalize("  Hello,   World! "), "hello world");
});

function rig({ handsFree = true, bargeIn = true, speaking = false, text = "hello there" } = {}) {
  const log = { sent: [], interrupts: 0, modes: [], muted: [], stopped: 0, quiet: [] };
  let handlers;
  const listener = { start: async () => {}, stop() { log.stopped += 1; }, setMuted: (v) => log.muted.push(v), hearAbove: (v) => log.quiet.push(v) };
  const state = { speaking };
  const talk = createTalk({
    makeListener: (h) => { handlers = h; return listener; },
    speaker: { speaking: () => state.speaking },
    transcribe: async () => text,
    send: async (said, extra) => { log.sent.push([said, extra.interrupted]); },
    interrupt: () => { log.interrupts += 1; },
    options: () => ({ handsFree, bargeIn }),
    onMode: (m) => log.modes.push(m),
  });
  return { talk, log, state, say: () => handlers.onUtterance(new Blob(["x"]), 3000), start: () => handlers.onSpeechStart() };
}

test("a hands-free turn: listen, understand, send, wait, listen again", async () => {
  const { talk, log, say } = rig();
  await talk.toggle();
  await say();
  assert.deepEqual(log.sent, [["hello there", false]]);
  assert.equal(talk.mode(), "waiting");
  talk.replyDone();
  assert.equal(talk.mode(), "listening");
  assert.deepEqual(log.modes, ["listening", "transcribing", "waiting", "listening"]);
});

test("one tap, one turn when it is not hands-free: the microphone closes while she answers", async () => {
  const { talk, log, say } = rig({ handsFree: false });
  await talk.toggle();
  await say();
  assert.equal(log.stopped, 1);
  talk.replyDone();
  assert.equal(talk.mode(), "off");
});

test("talking over her stops her and tells the next message it was an interruption", async () => {
  const { talk, log, say, start } = rig({ speaking: true });
  await talk.toggle();
  start();
  assert.equal(log.interrupts, 1);
  await say();
  assert.deepEqual(log.sent, [["hello there", true]]);
});

test("without barge-in the microphone is muted while she speaks", async () => {
  const { talk, log, state } = rig({ bargeIn: false });
  await talk.toggle();
  state.speaking = true;
  talk.refresh();
  assert.equal(log.muted.at(-1), true);
  state.speaking = false;
  talk.refresh();
  assert.equal(log.muted.at(-1), false);
});

test("saying stop ends the conversation without sending it", async () => {
  const { talk, log, say } = rig({ text: "Stop." });
  await talk.toggle();
  await say();
  assert.deepEqual(log.sent, []);
  assert.equal(talk.mode(), "off");
});

test("silence or a phantom goes back to listening and sends nothing", async () => {
  const { talk, log, say } = rig({ text: "" });
  await talk.toggle();
  await say();
  assert.deepEqual(log.sent, []);
  assert.equal(talk.mode(), "listening");
});

test("a second tap on the microphone ends listening", async () => {
  const { talk } = rig();
  await talk.toggle();
  await talk.toggle();
  assert.equal(talk.mode(), "off");
});
