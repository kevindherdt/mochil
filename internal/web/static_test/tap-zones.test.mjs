import test from "node:test";
import assert from "node:assert/strict";
import { tapAction, savedTapZones } from "../static/tap-zones.js";

test("edges turn the page at the outer quarters only", () => {
  assert.equal(tapAction("edges", 100, 1000), "back");
  assert.equal(tapAction("edges", 500, 1000), "menu");
  assert.equal(tapAction("edges", 900, 1000), "forward");
});
test("one hand goes back from the left fifth and forward elsewhere", () => {
  assert.equal(tapAction("one hand", 100, 1000), "back");
  assert.equal(tapAction("one hand", 300, 1000), "forward");
  assert.equal(tapAction("one hand", 900, 1000), "forward");
});
test("every choice keeps a way to the menu", () => {
  for (const zones of ["edges", "one hand", "off"]) {
    const seen = new Set(Array.from({ length: 101 }, (_, i) => tapAction(zones, i * 10, 1000)));
    assert.ok(seen.has("menu"), zones);
  }
  assert.equal(tapAction("one hand", 500, 1000), "menu");
});
test("off never turns the page, and right-to-left swaps the sides", () => {
  assert.equal(tapAction("off", 900, 1000), "menu");
  assert.equal(tapAction("edges", 100, 1000, true), "forward");
  assert.equal(tapAction("edges", 900, 1000, true), "back");
  assert.equal(tapAction("edges", 10, 0), "menu");
});
test("the choice is remembered and defaults to edges", () => {
  assert.equal(savedTapZones({ getItem: () => null }), "edges");
  assert.equal(savedTapZones({ getItem: () => "off" }), "off");
  assert.equal(savedTapZones({ getItem: () => "junk" }), "edges");
  assert.equal(savedTapZones({ getItem: () => { throw new Error("blocked"); } }), "edges");
});
