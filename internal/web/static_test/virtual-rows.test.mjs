import assert from "node:assert/strict";
import test from "node:test";
import { scrollTopFor, stillCovers, windowFor } from "../static/virtual-rows.js";

const base = { viewport: 300, rowHeight: 30, count: 1000 };

test("only the rows in view and a few either side are drawn, with space standing in for the rest", () => {
  const part = windowFor({ ...base, scrollTop: 3000, overscan: 5 });
  assert.deepEqual([part.start, part.end], [95, 115]);
  assert.equal(part.before, 95 * 30);
  assert.equal(part.after, (1000 - 115) * 30);
  assert.equal(part.before + (part.end - part.start) * 30 + part.after, 1000 * 30, "the scrollbar is the length of the whole list");
});

test("the start and the end of the list are not overrun", () => {
  assert.deepEqual(windowFor({ ...base, scrollTop: 0, overscan: 5 }), { start: 0, end: 15, before: 0, after: 985 * 30 });
  const last = windowFor({ ...base, scrollTop: 1000 * 30 - 300, overscan: 5 });
  assert.equal(last.end, 1000);
  assert.equal(last.after, 0);
  assert.deepEqual(windowFor({ ...base, count: 0, scrollTop: 0 }), { start: 0, end: 0, before: 0, after: 0 });
});

test("a small scroll redraws nothing, a big one does", () => {
  const drawn = windowFor({ ...base, scrollTop: 3000 });
  assert.ok(stillCovers(drawn, { ...base, scrollTop: 3040 }));
  assert.ok(!stillCovers(drawn, { ...base, scrollTop: 6000 }));
  assert.ok(!stillCovers(drawn, { ...base, scrollTop: 0 }));
  assert.ok(stillCovers(windowFor({ ...base, scrollTop: 0 }), { ...base, scrollTop: 10 }), "at the top there is nothing more above");
});

test("moving to a row scrolls only as far as needed, below a pinned heading", () => {
  const view = { scrollTop: 300, viewport: 300, rowHeight: 30, header: 30 };
  assert.equal(scrollTopFor(12, view), 300, "already visible");
  assert.equal(scrollTopFor(2, view), 60, "above: its top meets the top");
  assert.equal(scrollTopFor(25, view), 25 * 30 + 30 - 300 + 30, "below: its bottom meets the bottom");
});
