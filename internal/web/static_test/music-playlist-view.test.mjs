import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  musicPlaylistCardsHTML,
  musicPlaylistDetailHTML,
  musicPlaylistOverviewHTML,
  playlistArtworkURLs,
} from "../static/music-playlist-view.js";

const escapeHTML = value =>
  String(value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;");

test("playlist overview stays compact and escapes account data", () => {
  const html = musicPlaylistOverviewHTML(
    [{ name: '<Late "night">', items: [{ artwork_url: "/cover?a=1&b=2" }] }],
    escapeHTML,
  );
  assert.match(html, /music-playlist-grid/);
  assert.match(html, /data-new-playlist/);
  assert.match(html, /&lt;Late &quot;night&quot;>/);
  assert.match(html, /1 track/);
  assert.match(html, /class="music-playlist-art" data-art="\/cover\?a=1&amp;b=2"/);
  assert.doesNotMatch(html, /music-playlist-mosaic/);
  assert.doesNotMatch(html, /data-delete-playlist/);
});

test("playlist overview follows the shared music cover and list preference", () => {
  const playlists = [{ name: "Focus", items: [{ artwork_url: "/focus" }] }];
  assert.doesNotMatch(musicPlaylistOverviewHTML(playlists, escapeHTML, "covers"), /music-playlist-grid list/);
  assert.match(musicPlaylistOverviewHTML(playlists, escapeHTML, "list"), /music-playlist-grid list/);

  const css = readFileSync(new URL("../static/music-library.css", import.meta.url), "utf8");
  assert.match(css, /\.music-playlist-grid\.list\{grid-template-columns:1fr;gap:0/);
  assert.match(css, /\.music-playlist-grid\.list \.music-playlist-art\{[^}]*width:54px;height:54px/);
});

test("playlist views tolerate legacy null item collections", () => {
  const playlist = { name: "Empty", items: null };
  const overview = musicPlaylistOverviewHTML([playlist], escapeHTML);
  const detail = musicPlaylistDetailHTML(playlist, () => "unexpected row", escapeHTML);
  assert.match(overview, />0 tracks</);
  assert.match(detail, /data-play-playlist disabled/);
  assert.match(detail, /Add tracks from their options menu/);
  assert.doesNotMatch(detail, /unexpected row/);
  assert.match(musicPlaylistOverviewHTML(null, escapeHTML), /Create a playlist/);
});

test("playlist artwork is unique, ordered, and bounded to four covers", () => {
  const playlist = { items: [
    { artwork_url: "/one" }, { artwork_url: "/one" },
    { player_artwork_url: "/two" }, { artwork_url: "" },
    { artwork_url: "/three" }, { artwork_url: "/four" },
    { artwork_url: "/five" },
  ] };
  assert.deepEqual(playlistArtworkURLs(playlist), ["/one", "/two", "/three", "/four"]);
  const html = musicPlaylistOverviewHTML([{ name: "Mix", ...playlist }], escapeHTML);
  assert.match(html, /music-playlist-mosaic/);
  assert.match(html, /data-art-count="4"/);
  assert.doesNotMatch(html, /\/five/);
});

test("discovery playlist cards are bounded and open without duplicating the overview", () => {
  const playlists = Array.from({ length: 10 }, (_, index) => ({
    name: `Mix ${index + 1}`,
    items: [{ artwork_url: `/cover-${index + 1}` }],
  }));
  const html = musicPlaylistCardsHTML(playlists, escapeHTML, { limit: 8, discovery: true });
  assert.match(html, /music-discovery-playlists/);
  assert.match(html, /data-discovery-playlist="Mix 1"/);
  assert.match(html, /data-discovery-playlist="Mix 8"/);
  assert.doesNotMatch(html, /Mix 9/);
  assert.doesNotMatch(html, /data-new-playlist/);
});

test("playlist detail owns playback and guarded destructive actions", () => {
  const html = musicPlaylistDetailHTML(
    { name: "Road trip", items: [{ id: "one", artwork_url: "/one" }, { id: "two", artwork_url: "/two" }] },
    item => `<article>${item.id}</article>`,
    escapeHTML,
  );
  assert.match(html, /data-playlist-back/);
  assert.match(html, /data-play-playlist/);
  assert.match(html, /<details>/);
  assert.match(html, /data-rename-playlist/);
  assert.match(html, /data-playlist-rename-form/);
  assert.match(html, /data-delete-playlist/);
  assert.match(html, /\/api\/audio-playlists\/Road%20trip\?download=1/);
  assert.match(html, /2 tracks/);
  assert.match(html, /music-playlist-mosaic/);
  assert.match(html, /aria-label="Back to playlists"><svg/);
  assert.match(html, /aria-label="Playlist options"><svg/);
  assert.doesNotMatch(html, />•••</);
});

test("focused playlist identity keeps its complete responsive title", () => {
  const css = readFileSync(new URL("../static/music-library.css", import.meta.url), "utf8");
  assert.match(css, /\.music-playlist-heading h2\{[^}]*overflow:visible;[^}]*text-overflow:clip;[^}]*white-space:normal/);
  assert.match(css, /@media\(max-width:900px\)\{\.music-playlist-heading\{grid-template-columns:44px 88px minmax\(0,1fr\)\}\.music-playlist-actions\{grid-column:3\/4/);
  assert.match(css, /@media\(max-width:560px\)\{[\s\S]{0,220}\.music-playlist-heading\{grid-template-columns:44px 62px minmax\(0,1fr\)/);
});
