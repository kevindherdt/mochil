import test from "node:test";
import assert from "node:assert/strict";
import { captionText } from "../static/hermes-stage.js";

test("a caption is the end of what was said, cut at a word", () => {
  assert.equal(captionText("  Hello\n there  "), "Hello there");
  assert.equal(captionText(""), "");
  const long = "word ".repeat(100).trim();
  const caption = captionText(long, 40);
  assert.ok(caption.startsWith("…word"), caption);
  assert.ok(caption.length <= 42);
});
