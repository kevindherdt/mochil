import test from "node:test";
import assert from "node:assert/strict";
import {
  mountVideoPlayer,
  videoPlayerHTML,
} from "../static/video-player-template.js";

test("video player template retains every controller element contract", () => {
  for (const id of [
    "video-title",
    "video-runtime",
    "track-menu",
    "audio-track-field",
    "audio-track",
    "subtitle-track-field",
    "subtitle-track",
    "video-close",
    "video-element",
    "video-preview",
    "video-preview-image",
    "video-preview-time",
    "video-message",
  ]) {
    assert.match(videoPlayerHTML, new RegExp(`id="${id}"`));
  }
  assert.match(videoPlayerHTML, /playsinline preload="metadata"/);
  assert.match(videoPlayerHTML, /role="status"/);
});

test("video player mounts once with dialog semantics", () => {
  const appended = [];
  const document = {
    getElementById: () => appended.find((node) => node.id === "video-player"),
    createElement: (tagName) => ({ tagName }),
    body: { append: (node) => appended.push(node) },
  };
  mountVideoPlayer(document);
  mountVideoPlayer(document);
  assert.equal(appended.length, 1);
  assert.equal(appended[0].tagName, "dialog");
  assert.equal(appended[0].className, "video-player");
});
