import assert from "node:assert/strict";
import test from "node:test";
import { musicLoadingHTML } from "../static/music-loading.js";
import { readFile } from "node:fs/promises";

const dashboardCSS = await readFile(
  new URL("../static/dashboard.css", import.meta.url),
  "utf8",
);

test("music loading feedback is bounded, accessible, and animation-free", () => {
  const markup = musicLoadingHTML();
  assert.match(markup, /role="status"/);
  assert.match(markup, /class="sr-only">Opening music/);
  assert.equal((markup.match(/music-loading-row/g) || []).length, 8);
  assert.doesNotMatch(markup, /style=|script|img/);
});

test("dashboard listening rows omit library-style separators", () => {
  assert.match(dashboardCSS, /\.audio-reading \.audio-item\{border-bottom:0\}/);
});

test("every resumable media type shares one card size and one text scale", () => {
  assert.match(dashboardCSS, /\.dashboard-reading\{--resume-card:136px;--resume-art:120px\}/);
  assert.match(dashboardCSS, /\.dashboard-reading \.reading-track\{grid-auto-columns:clamp\(280px,86vw,340px\)/);
  for (const surface of ["reading-item>a", "video-reading-item", "audio-reading .audio-open"])
    assert.ok(dashboardCSS.includes(surface), `${surface} is part of the shared resume card`);
  assert.match(dashboardCSS, /min-height:var\(--resume-card\)/);
  assert.match(dashboardCSS, /\.dashboard-reading\.audio-reading \.podcast-episode-description\{[^}]*-webkit-line-clamp:2/s);
  assert.match(dashboardCSS, /font:600 15px\/1\.25/);
});

test("wide dashboards put books on one side and watching and listening on the other", () => {
  assert.match(dashboardCSS, /@media\(min-width:1180px\)/);
  assert.match(dashboardCSS, />\.dashboard-media-lane\{grid-column:1\}/);
  assert.doesNotMatch(dashboardCSS, /#(?:video|audio)-reading-rail\{grid-column:/);
});
