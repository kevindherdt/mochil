import assert from "node:assert/strict";
import test from "node:test";
import { audioRenderPolicy } from "../static/audio-render-policy.js";

const base = {
  itemCount: 0,
  total: 0,
  visibleItemCount: 0,
  podcast: {},
  audiobook: {},
  music: {},
};

test("music roots distinguish finite destinations from paged catalogues", () => {
  const discovery = audioRenderPolicy({ ...base, kind: "music", browse: "discover" });
  assert.equal(discovery.loadMoreHidden, true);

  const albums = audioRenderPolicy({
    ...base,
    kind: "music",
    browse: "albums",
    music: { groupCount: 20, total: 50 },
  });
  assert.equal(albums.loaded, 20);
  assert.equal(albums.available, 50);
  assert.equal(albums.loadMoreHidden, false);
  assert.equal(albums.loadMoreLabel, "Load more collections");
});

test("recycled track windows count their retained offset", () => {
  const tracks = audioRenderPolicy({
    ...base,
    kind: "music",
    browse: "tracks",
    startOffset: 144,
    itemCount: 72,
    total: 500,
  });
  assert.equal(tracks.loaded, 216);
  assert.equal(tracks.loadMoreHidden, false);
});

test("podcast and audiobook play-all waits for an actionable complete group", () => {
  const podcastRoot = audioRenderPolicy({ ...base, kind: "podcasts", browse: "" });
  assert.equal(podcastRoot.loadMoreHidden, true);

  const podcastShow = audioRenderPolicy({
    ...base,
    kind: "podcasts",
    browse: "",
    visibleItemCount: 4,
    podcast: { show: "A show", episodeCount: 4, total: 8 },
  });
  assert.equal(podcastShow.loadMoreHidden, false);

  const partialAudiobookSeries = audioRenderPolicy({
    ...base,
    kind: "audiobooks",
    browse: "series",
    visibleItemCount: 4,
    audiobook: { group: "A series", bookCount: 4, total: 8 },
  });
  assert.equal(partialAudiobookSeries.loadMoreHidden, false);
});
