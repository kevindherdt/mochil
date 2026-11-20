import assert from "node:assert/strict";
import test from "node:test";
import {
  audioLibraryHTML,
  mountAudioLibrary,
} from "../static/audio-library-template.js";

test("audio library template retains browse, podcast, and paging contracts", () => {
  for (const id of [
    "audio-index",
    "music-browse-field",
    "music-browse",
    "podcast-state-field",
    "podcast-state",
    "music-destinations",
    "audiobook-browse",
    "audiobook-destinations",
    "podcast-layout-field",
    "podcast-layout",
    "audiobook-series-layout-field",
    "audiobook-series-layout",
    "audiobook-item-layout-field",
    "audiobook-item-layout",
    "audio-sort",
    "audio-analysis-action",
    "audio-analyze",
    "audio-analysis-status",
    "podcast-manage-toggle",
    "playlist-create",
    "playlist-name",
    "playlist-status",
    "podcast-subscriptions",
    "podcast-manage",
    "podcast-search",
    "podcast-query",
    "podcast-results",
    "podcast-add",
    "podcast-url",
    "podcast-status",
    "podcast-feeds",
    "audio-top-spacer",
    "audio-start-sentinel",
    "audio-load-previous",
    "audio-items",
    "audio-sentinel",
    "audio-load-more",
    "audio-bottom-spacer",
  ])
    assert.match(audioLibraryHTML, new RegExp(`id="${id}"`));
  assert.match(audioLibraryHTML, /aria-label="Sorting and options"/);
  assert.match(audioLibraryHTML, /M4 5h6v6H4zM14 5h6v6h-6z/);
  assert.match(audioLibraryHTML, /id="audio-layout-toggle"/);
  assert.match(audioLibraryHTML, /aria-label="Search podcast directory"/);
  assert.match(audioLibraryHTML, /value="discover">Discover/);
  assert.match(audioLibraryHTML, /value="recently-played">Recently played/);
  assert.match(audioLibraryHTML, /value="queue">Queue/);
  assert.match(audioLibraryHTML, /value="series">Series \/ volume/);
  assert.doesNotMatch(audioLibraryHTML, /id="audio-shuffle"/);
});

test("audio library mounts once before the shared collection anchor", () => {
  const insertions = [];
  let mounted = false;
  const documents = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
    append() {},
  };
  const document = {
    querySelector(selector) {
      return selector === "#audio-index" ? (mounted ? {} : null) : documents;
    },
  };
  mountAudioLibrary(document);
  mountAudioLibrary(document);
  assert.deepEqual(insertions, [
    { position: "beforebegin", markup: audioLibraryHTML },
  ]);
});
