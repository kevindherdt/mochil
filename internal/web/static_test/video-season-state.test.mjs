import assert from "node:assert/strict";
import test from "node:test";
import {
  applySeasonItems,
  isCurrentVideoSeasonView,
} from "../static/video-season-state.js";

test("appends season items once across overlapping season results", () => {
  const items = [{ id: "already-loaded" }];
  const knownIDs = new Set(["already-loaded"]);
  const request = {};
  const target = { isConnected: true };
  const view = { request, items, knownIDs, target };
  const applied = applySeasonItems(view, request, items, knownIDs, [
    { id: "already-loaded" },
    { id: "episode-1" },
    { id: "episode-1" },
    { id: "episode-2" },
  ]);

  assert.equal(applied, true);
  assert.deepEqual(items.map((item) => item.id), [
    "already-loaded",
    "episode-1",
    "episode-2",
  ]);
  assert.deepEqual([...knownIDs], ["already-loaded", "episode-1", "episode-2"]);
});

test("rejects season responses after the active view changes or is detached", () => {
  const request = {};
  const items = [];
  const knownIDs = new Set();
  const target = { isConnected: true };
  const view = { request, items, knownIDs, target };

  assert.equal(isCurrentVideoSeasonView(view, request, items, knownIDs), true);
  assert.equal(applySeasonItems(view, {}, items, knownIDs, [{ id: "stale" }]), false);
  assert.deepEqual(items, []);
  assert.equal(isCurrentVideoSeasonView(view, {}, items, knownIDs), false);
  assert.equal(isCurrentVideoSeasonView(view, request, [], knownIDs), false);
  assert.equal(isCurrentVideoSeasonView(view, request, items, new Set()), false);
  target.isConnected = false;
  assert.equal(isCurrentVideoSeasonView(view, request, items, knownIDs), false);
});
