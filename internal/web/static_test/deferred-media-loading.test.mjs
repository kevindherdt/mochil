import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const staticRoot = new URL("../static/", import.meta.url);
const [app, deferredAudio, audioLibrary, deferredVideo, videoLibrary] = await Promise.all([
  readFile(new URL("app.js", staticRoot), "utf8"),
  readFile(new URL("deferred-audio-controller.js", staticRoot), "utf8"),
  readFile(new URL("audio-library.js", staticRoot), "utf8"),
  readFile(new URL("deferred-video-controller.js", staticRoot), "utf8"),
  readFile(new URL("video-library.js", staticRoot), "utf8"),
]);

test("Home owns only the bounded audio continuation bootstrap", () => {
  assert.doesNotMatch(app, /^import .*audio-library\.js/m);
  assert.match(app, /createDeferredAudioController/);
  assert.match(deferredAudio, /import\("\.\/audio-library\.js\?v=164"\)/);
  assert.match(deferredAudio, /createAudioContinueRail/);
  assert.doesNotMatch(audioLibrary, /createAudioContinueRail/);
});

test("Home owns only the bounded video continuation bootstrap", () => {
  assert.doesNotMatch(app, /^import .*video-library\.js/m);
  assert.match(app, /createDeferredVideoController/);
  assert.match(deferredVideo, /import\("\.\/video-library\.js\?v=\d+"\)/);
  assert.match(deferredVideo, /createVideoContinueRail/);
  assert.doesNotMatch(videoLibrary, /createVideoContinueRail/);
});

test("superseded audio routes cannot commit after deferred loading", () => {
  assert.match(deferredAudio, /const requestedGeneration = \+\+generation/);
  assert.match(deferredAudio, /requestedGeneration === generation/);
  assert.match(deferredAudio, /generation \+= 1/);
});

import { lazyOnce } from "../static/lazy-once.js";

test("a feature module that failed to load is tried again, not remembered as broken", async () => {
  let attempts = 0;
  const load = lazyOnce(async () => {
    attempts += 1;
    if (attempts === 1) throw new Error("network");
    return { ready: true };
  });
  await assert.rejects(load(), /network/);
  assert.deepEqual(await load(), { ready: true });
  assert.equal(await load(), await load());
  assert.equal(attempts, 2, "a loaded module is reused");
});
