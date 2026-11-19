import assert from "node:assert/strict";
import test from "node:test";
import { createAlphabetPeek } from "../static/alphabet-peek.js";

test("the A–Z rail wakes on scroll or near the right edge, then rests", async () => {
  const classes = new Set();
  const listeners = {};
  const document = { documentElement: { classList: { add: (n) => classes.add(n), remove: (n) => classes.delete(n) } } };
  const window = { innerWidth: 1000, addEventListener: (type, fn) => (listeners[type] = fn) };
  createAlphabetPeek({ document, window, delay: 5 });
  listeners.pointermove({ clientX: 100 });
  assert.equal(classes.has("rail-awake"), false);
  listeners.pointermove({ clientX: 980 });
  assert.equal(classes.has("rail-awake"), true);
  await new Promise((resolve) => setTimeout(resolve, 20));
  assert.equal(classes.has("rail-awake"), false);
  listeners.scroll();
  assert.equal(classes.has("rail-awake"), true);
});

