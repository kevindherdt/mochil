import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { videoHLSURL, videoResumePosition } from "../static/video-player.js";

test("video HLS URLs encode every server-controlled path segment", () => {
  assert.equal(
    videoHLSURL(
      { library_id: "TV & anime", id: "episode/1", revision: "rev ?2" },
      3,
    ),
    "/api/video-hls/TV%20%26%20anime/episode%2F1/3/high/index.m3u8?v=rev%20%3F2-7",
  );
});

test("video resume positions reject the end boundary and invalid values", () => {
  assert.equal(videoResumePosition(3600, 125.5), 125.5);
  assert.equal(videoResumePosition(3600, 3595), 0);
  assert.equal(videoResumePosition(3600, -1), 0);
  assert.equal(videoResumePosition(0, 10), 0);
  assert.equal(videoResumePosition(3600, Number.NaN), 0);
});

test("HLS resumes unfinished video when the catalogue duration is missing", () => {
  assert.equal(videoResumePosition(undefined, 125.5, false), 125.5);
  // attachHLS converts the missing value to NaN before applying the guard again.
  assert.equal(videoResumePosition(Number.NaN, 125.5, false), 125.5);
  assert.equal(videoResumePosition(undefined, 125.5, true), 0);
});

test("finished video starts over even when marked complete at 95 percent", () => {
  assert.equal(videoResumePosition(3600, 3420, true), 0);
  assert.equal(videoResumePosition(3600, 3420, false), 3420);
});

test("closing video relinquishes the active item before pause can save again", async () => {
  const source = await readFile(new URL("../static/video-player.js", import.meta.url), "utf8");
  const closeBody = source.match(/function closePlayer\(\) \{([\s\S]*?)\n  \}/)?.[1] || "";
  const finalSave = closeBody.indexOf("saveProgress(true)");
  const relinquish = closeBody.indexOf("activeItem = null");
  const pause = closeBody.indexOf("video.pause()");
  assert.ok(finalSave >= 0 && relinquish > finalSave && pause > relinquish, closeBody);
});

test("destroying the HLS player also tells the server to stop its encoder", async () => {
  const source = await readFile(new URL("../static/video-player.js", import.meta.url), "utf8");
  const destroyBody = source.match(/function destroyHLS\(\) \{([\s\S]*?)\n  \}/)?.[1] || "";
  assert.match(destroyBody, /stopServerTranscode\(\)/);
});

test("the server stop request is a DELETE against the active HLS source, and clears it so it cannot fire twice", async () => {
  const source = await readFile(new URL("../static/video-player.js", import.meta.url), "utf8");
  const stopBody = source.match(/function stopServerTranscode\(\) \{([\s\S]*?)\n  \}/)?.[1] || "";
  assert.match(stopBody, /if \(!activeHLSSource\) return/);
  assert.match(stopBody, /method:\s*"DELETE"/);
  assert.match(stopBody, /activeHLSSource = undefined/);
});

test("a pagehide (tab close, navigating away) beacons the stop since fetch would not survive it", async () => {
  const source = await readFile(new URL("../static/video-player.js", import.meta.url), "utf8");
  assert.match(source, /addEventListener\("pagehide",\s*\(\)\s*=>\s*\{\s*if \(activeHLSSource\) navigator\.sendBeacon/);
});
