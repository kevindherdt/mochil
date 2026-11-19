import test from "node:test";
import assert from "node:assert/strict";
import { speechText, takeSentences, createSpeaker, sampleSplitter, createPlayer } from "../static/hermes-voice.js";

test("speechText drops what should not be heard", () => {
  assert.equal(speechText("**Hi** there 😊 *smiles softly* see [this](https://x.io) `code`"), "Hi there see this code");
  assert.equal(speechText("Really *really* good"), "Really really good");
  assert.equal(speechText("# Title\n- one\n- two\n1. three"), "Title one two three");
  assert.equal(speechText("before ```js\nlet a = 1\n``` after"), "before after");
});

test("takeSentences waits for a finished sentence and joins short ones", () => {
  assert.deepEqual(takeSentences("Hello there, how was your day"), { sentences: [], rest: "Hello there, how was your day" });
  assert.deepEqual(takeSentences("Hello there, how was your day? I mi"), { sentences: ["Hello there, how was your day?"], rest: "I mi" });
  const joined = takeSentences("Hi. Okay. And now a longer sentence follows here. Rest");
  assert.deepEqual(joined.sentences, ["Hi. Okay. And now a longer sentence follows here."]);
  assert.equal(joined.rest, "Rest");
});

test("takeSentences flushes the end of a reply, however short", () => {
  assert.deepEqual(takeSentences("Yes.", { final: true }), { sentences: ["Yes."], rest: "" });
  assert.deepEqual(takeSentences("That is a fine long sentence indeed. Bye", { final: true }).sentences, ["That is a fine long sentence indeed.", "Bye"]);
  assert.deepEqual(takeSentences("   ", { final: true }).sentences, []);
});

test("takeSentences streams the same sentences however the text is cut up", () => {
  const text = "Good morning, love. I slept really well last night, thank you for asking. What are we doing today?";
  const whole = takeSentences(text, { final: true }).sentences;
  let buffer = "", got = [];
  for (const piece of text.match(/.{1,5}/g)) {
    buffer += piece;
    const { sentences, rest } = takeSentences(buffer);
    got.push(...sentences); buffer = rest;
  }
  got.push(...takeSentences(buffer, { final: true }).sentences);
  assert.deepEqual(got, whole);
});

// A fetch that answers a sentence with streamed audio: `pieces` bytes arrive one after another.
const streamingFetch = (log, { pieces = [new Uint8Array(480), new Uint8Array(480)], rate = "24000" } = {}) => async (url, options) => {
  if (url !== "/api/narration/stream") return { ok: true };
  log.push(JSON.parse(options.body));
  let index = 0;
  return { ok: true, headers: { get: (name) => (name === "X-Sample-Rate" ? rate : null) },
    body: { getReader: () => ({ read: async () => (index < pieces.length ? { value: pieces[index++], done: false } : { done: true }) }) } };
};
const fakePlayer = () => {
  const plays = [];
  return { plays, unlock() {}, analyser: () => null, stop() { this.stopped = true; }, play(bytes, rate) { plays.push([bytes.length, rate]); return new Promise((resolve) => setTimeout(resolve, 5)); } };
};

test("sampleSplitter keeps samples whole when a piece cuts one in half", () => {
  const split = sampleSplitter();
  assert.deepEqual([...split(Uint8Array.of(1, 2, 3))], [1, 2]);
  assert.deepEqual([...split(Uint8Array.of(4, 5, 6))], [3, 4, 5, 6]);
  assert.deepEqual([...split(Uint8Array.of(7, 8))], [7, 8]);
});

test("the speaker asks for each sentence once, in order, and plays pieces as they arrive", async () => {
  const asked = [];
  const player = fakePlayer();
  const speaker = createSpeaker({ fetchImpl: streamingFetch(asked), player, voice: () => "alba", prebuffer: 0 });
  const reply = speaker.begin();
  reply.push("This is the first full sentence. And this is the second ");
  reply.push("full sentence of the reply. Bye");
  reply.end();
  assert.equal(speaker.speaking(), true);
  await new Promise((resolve) => setTimeout(resolve, 60));
  assert.deepEqual(asked.map((a) => a.text), ["This is the first full sentence.", "And this is the second full sentence of the reply.", "Bye"]);
  assert.ok(asked.every((a) => a.voice === "alba"));
  assert.equal(player.plays.length, 6, "two pieces for each of the three sentences");
  assert.ok(player.plays.every(([bytes, rate]) => bytes === 480 && rate === 24000));
  assert.equal(speaker.speaking(), false);
});

test("a sentence that is still being made starts playing before it is complete", async () => {
  const player = fakePlayer();
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const fetchImpl = async () => {
    let step = 0;
    return { ok: true, headers: { get: () => "24000" }, body: { getReader: () => ({ read: async () => {
      if (step === 0) { step = 1; return { value: new Uint8Array(960), done: false }; }
      await gate; return { done: true };
    } }) } };
  };
  const speaker = createSpeaker({ fetchImpl, player, prebuffer: 0 });
  speaker.speakText("A sentence that takes a while to make.");
  await new Promise((resolve) => setTimeout(resolve, 20));
  assert.equal(player.plays.length, 1, "the first piece plays while the rest is still coming");
  assert.equal(speaker.speaking(), true);
  release();
  await new Promise((resolve) => setTimeout(resolve, 30));
  assert.equal(speaker.speaking(), false);
});

test("stop silences the player and drops sentences that were not played", async () => {
  const player = fakePlayer();
  const speaker = createSpeaker({ fetchImpl: streamingFetch([]), player, prebuffer: 0 });
  speaker.speakText("One long enough sentence here. Another sentence follows it.");
  speaker.stop();
  await new Promise((resolve) => setTimeout(resolve, 30));
  assert.equal(player.stopped, true);
  assert.equal(speaker.speaking(), false);
});

test("a failed request ends quietly and the speaker is free again", async () => {
  const player = fakePlayer();
  const fetchImpl = async () => ({ ok: false, status: 503, text: async () => "Narration is not set up" });
  const speaker = createSpeaker({ fetchImpl, player, prebuffer: 0 });
  speaker.speakText("This sentence cannot be spoken at all.");
  await new Promise((resolve) => setTimeout(resolve, 30));
  assert.equal(player.plays.length, 0);
  assert.equal(speaker.speaking(), false);
});

test("the player schedules pieces back to back and reports what is queued", () => {
  const started = [];
  const context = {
    currentTime: 1, destination: {},
    createAnalyser: () => ({ fftSize: 0, connect() {} }),
    createBuffer: (channels, length, rate) => ({ duration: length / rate, getChannelData: () => new Float32Array(length) }),
    createBufferSource: () => ({ connect() {}, start(at) { started.push(at); }, stop() {} }),
    resume: async () => {},
  };
  const player = createPlayer({ AudioContextClass: function () { return context; } });
  player.play(new Uint8Array(4800), 24000); // 2400 samples = 0.1 s
  player.play(new Uint8Array(4800), 24000);
  assert.equal(started.length, 2);
  assert.ok(Math.abs(started[1] - (started[0] + 0.1)) < 1e-9, "the second piece starts exactly when the first ends");
  assert.ok(player.queued() > 0.2);
  player.stop();
  assert.equal(player.queued(), 0);
});
