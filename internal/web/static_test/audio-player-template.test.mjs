import test from "node:test";
import assert from "node:assert/strict";
import {
  audioPlayerHTML,
  mountAudioPlayer,
} from "../static/audio-player-template.js";

test("audio player template retains transport and expanded-player contracts", () => {
  for (const id of [
    "audio-element",
    "audio-gapless-element",
    "audio-seek",
    "audio-play",
    "audio-player-art",
    "audio-expand",
    "audio-title",
    "audio-context",
    "audio-elapsed",
    "audio-duration",
    "audio-previous-large",
    "audio-play-large",
    "audio-next-large",
    "audio-chapters",
    "audio-chapters-close",
    "audio-chapter-list",
    "audio-collapse",
    "audio-player-favorite",
    "audio-previous",
    "audio-next",
    "audio-player-shuffle-large",
    "audio-queue-toggle",
    "audio-player-queue",
    "audio-queue-close",
    "audio-queue-items",
    "audio-queue-earlier",
    "audio-queue-later",
    "audio-speed",
    "audio-replaygain",
    "audio-chapters-toggle",
    "audio-sleep",
    "audio-close",
  ]) {
    assert.match(audioPlayerHTML, new RegExp(`id="${id}"`));
  }
  assert.match(audioPlayerHTML, /aria-label="Playback position"/);
  assert.match(audioPlayerHTML, /aria-expanded="false"/);
  assert.match(audioPlayerHTML, /aria-controls="audio-player-queue"/);
});

test("audio player mounts once, hidden and labelled", () => {
  const appended = [];
  const document = {
    getElementById: () => appended.find((node) => node.id === "audio-player"),
    createElement: (tagName) => ({
      tagName,
      setAttribute(name, value) {
        this[name] = value;
      },
    }),
    body: { append: (node) => appended.push(node) },
  };
  mountAudioPlayer(document);
  mountAudioPlayer(document);
  assert.equal(appended.length, 1);
  assert.equal(appended[0].tagName, "section");
  assert.equal(appended[0].hidden, true);
  assert.equal(appended[0]["aria-label"], "Audio player");
});
