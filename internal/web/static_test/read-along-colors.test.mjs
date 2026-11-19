import assert from "node:assert/strict";
import test from "node:test";
import { highlightColor, highlightColors, highlightCSS, highlightKey } from "../static/read-along-colors.js";

const memory = (initial = {}) => ({ getItem: (key) => initial[key] ?? null });

test("the highlight defaults to amber and ignores unknown or unreadable choices", () => {
  assert.equal(highlightColor(memory()), "amber");
  assert.equal(highlightColor(memory({ [highlightKey]: "nope" })), "amber");
  assert.equal(highlightColor(memory({ [highlightKey]: "toString" })), "amber");
  assert.equal(highlightColor({ getItem() { throw new Error("blocked"); } }), "amber");
  assert.equal(highlightColor(memory({ [highlightKey]: "green" })), "green");
});

test("every colour makes a highlight that survives the reader's own background rules", () => {
  for (const name of Object.keys(highlightColors)) {
    const css = highlightCSS("active", name);
    assert.match(css, /\.active\{background-image:linear-gradient\(rgba\(\d+,\d+,\d+,\.38\)/);
    assert.match(css, /!important/);
  }
});
