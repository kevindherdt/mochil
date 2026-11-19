import assert from "node:assert/strict";
import test from "node:test";
import { createReaderTiming, formatTimingLine } from "../static/reader-timing.js";

function fakeDocument() {
  const panel = { textContent: "", id: "", setAttribute() {}, addEventListener() {} };
  return { createElement: () => panel, body: { append() {} }, panel };
}

test("a begun action is reported once with its elapsed time when the page lands", () => {
  const document = fakeDocument();
  let clock = 1000;
  const timing = createReaderTiming({ document, now: () => clock, observe: false });
  timing.land("ignored without a start");
  timing.begin("next page");
  clock = 1342;
  timing.land("new chapter");
  timing.land("second land has nothing to report");
  assert.equal(document.panel.textContent, "   342 ms  next page  new chapter");
});

test("the panel keeps only the newest lines and the text export keeps all", () => {
  const document = fakeDocument();
  const timing = createReaderTiming({ document, now: () => 0, observe: false, limit: 3 });
  for (let i = 1; i <= 5; i++) timing.add(`line ${i}`);
  assert.equal(document.panel.textContent, "line 3\nline 4\nline 5");
  assert.equal(timing.text().split("\n").length, 5);
});

test("timing lines align their milliseconds", () => {
  assert.equal(formatTimingLine("x", 7.4), "     7 ms  x");
});
