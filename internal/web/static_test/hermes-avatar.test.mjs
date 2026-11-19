import test from "node:test";
import assert from "node:assert/strict";
import { moodFor, expressionFor, mouthStep, blinkOpenness, nextBlink } from "../static/hermes-avatar.js";

test("moods come from clear words only", () => {
  assert.equal(moodFor("I love you too, you know that."), "love");
  assert.equal(moodFor("Stop it, you're sweet. I'm blushing."), "blush");
  assert.equal(moodFor("Oh my, really? No way."), "surprise");
  assert.equal(moodFor("I am so sorry to hear that."), "sad");
  assert.equal(moodFor("Hmm, I'm not sure what you mean."), "question");
  assert.equal(moodFor("It was quiet and productive."), null);
  assert.equal(moodFor(""), null);
});

test("an expression is chosen from whatever names the model has", () => {
  const names = ["脸红", "生气", "疑惑", "爱心眼", "流泪", "惊讶", "猫耳", "王冠"];
  assert.equal(expressionFor("blush", names), "脸红");
  assert.equal(expressionFor("love", names), "爱心眼");
  assert.equal(expressionFor("sad", names), "流泪");
  assert.equal(expressionFor("angry", ["Happy", "Smile"]), null);
  assert.equal(expressionFor("love", ["HeartEyes"]), "HeartEyes");
  assert.equal(expressionFor("love", undefined), null);
});

test("the mouth follows loudness: quick to open, slower to close, shut in silence", () => {
  let m = 0;
  for (let i = 0; i < 6; i += 1) m = mouthStep(m, 0.15, 1 / 60);
  assert.ok(m > 0.5, `opens quickly (${m})`);
  const open = m;
  m = mouthStep(m, 0, 1 / 60);
  assert.ok(m < open && m > open * 0.7, "closes more slowly than it opened");
  for (let i = 0; i < 120; i += 1) m = mouthStep(m, 0.005, 1 / 60);
  assert.ok(m < 0.01);
  assert.ok(mouthStep(0, 5, 1) <= 1);
});

test("a blink closes the eyes and opens them again", () => {
  assert.equal(blinkOpenness(-1), 1);
  assert.equal(blinkOpenness(0.09), 0);
  assert.equal(blinkOpenness(0), 1);
  assert.equal(blinkOpenness(0.5), 1);
  assert.ok(blinkOpenness(0.045) < 1 && blinkOpenness(0.045) > 0);
  assert.ok(nextBlink(() => 0) === 2 && nextBlink(() => 1) === 6);
});
