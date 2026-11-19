import test from "node:test";
import assert from "node:assert/strict";
import { createVad, encodeWav } from "../static/hermes-listen.js";

const run = (vad, readings) => readings.flatMap(([rms, count]) => Array.from({ length: count }, () => vad.feed(rms, 20))).filter(Boolean);

test("speech starts after a moment of loudness and ends after a short silence", () => {
  const vad = createVad();
  assert.deepEqual(run(vad, [[0.004, 50], [0.1, 30], [0.004, 60]]), ["start", "end"]);
});

test("a click or cough shorter than the minimum is not speech", () => {
  assert.deepEqual(run(createVad(), [[0.004, 50], [0.1, 6], [0.004, 80]]), []);
});

test("short pauses between words do not end the turn, a long one does", () => {
  const vad = createVad();
  assert.deepEqual(run(vad, [[0.004, 20], [0.1, 20], [0.004, 25], [0.1, 20], [0.004, 60]]), ["start", "end"]);
});

test("steady room noise below the bar is ignored, speech above it is heard", () => {
  const vad = createVad();
  assert.deepEqual(run(vad, [[0.015, 300]]), []);
  assert.deepEqual(run(vad, [[0.1, 30], [0.015, 60]]), ["start", "end"]);
});

test("while she speaks, the detector is pickier", () => {
  const vad = createVad();
  vad.sensitivity({ factor: 5, minSpeech: 420 });
  assert.deepEqual(run(vad, [[0.004, 50], [0.1, 15]]), []);
  assert.deepEqual(run(vad, [[0.1, 10]]), ["start"]);
});

test("a very long turn is cut off", () => {
  const vad = createVad({ maxMs: 2000 });
  assert.deepEqual(run(vad, [[0.1, 120]]).slice(0, 2), ["start", "end"]);
});

test("encodeWav makes a 16 kHz mono 16-bit file", async () => {
  const rate = 48000;
  const block = new Float32Array(rate / 2).fill(0.5); // half a second
  const wav = encodeWav([block], rate);
  const bytes = new DataView(await wav.arrayBuffer());
  const tag = (offset) => String.fromCharCode(...[0, 1, 2, 3].map((i) => bytes.getUint8(offset + i)));
  assert.equal(wav.type, "audio/wav");
  assert.deepEqual([tag(0), tag(8), tag(12), tag(36)], ["RIFF", "WAVE", "fmt ", "data"]);
  assert.equal(bytes.getUint32(24, true), 16000);
  assert.equal(bytes.getUint16(22, true), 1);
  assert.equal(bytes.getUint32(40, true), 8000 * 2); // 0.5 s at 16 kHz, 2 bytes each
  assert.ok(Math.abs(bytes.getInt16(44, true) - 0x3fff) < 4);
});
