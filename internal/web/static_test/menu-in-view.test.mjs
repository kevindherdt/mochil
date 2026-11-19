import assert from "node:assert/strict";
import test from "node:test";
import { shiftToFit } from "../static/menu-in-view.js";

test("a menu already inside the window stays where it is", () => {
  assert.equal(shiftToFit(100, 360, 900), 0);
});

test("a menu off the left edge slides right to the margin", () => {
  assert.equal(shiftToFit(-112, 98, 420), 120);
});

test("a menu off the right edge slides left to the margin", () => {
  assert.equal(shiftToFit(344, 604, 420), -192);
});

test("a menu wider than the window starts at the left edge", () => {
  assert.equal(shiftToFit(300, 700, 380), -292);
});
