import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const staticRoot = new URL("../static/", import.meta.url);

test("music discovery styles live outside the shared application stylesheet", async () => {
  const [shared, discovery] = await Promise.all([
    readFile(new URL("app.css", staticRoot), "utf8"),
    readFile(new URL("music-discovery.css", staticRoot), "utf8"),
  ]);

  assert.doesNotMatch(shared, /\.music-discovery\b/);
  assert.match(discovery, /\.music-discovery\b/);
  assert.match(discovery, /scroll-snap-type:\s*x proximity/);
});

test("settings navigation uses the workspace width independently of narrow form panels", async () => {
  const [navigation, deferred] = await Promise.all([
    readFile(new URL("settings-navigation.css", staticRoot), "utf8"),
    readFile(new URL("deferred-route-controllers.js", staticRoot), "utf8"),
  ]);

  assert.match(navigation, /\.settings-nav\s*\{[^}]*width:\s*100%/);
  assert.match(navigation, /\.settings-nav\s*\{[^}]*max-width:\s*none/);
  assert.match(navigation, /scroll-snap-type:\s*x proximity/);
  assert.match(navigation, /scroll-snap-align:\s*start/);
  assert.match(deferred, /settings-navigation\.css/);
});

test("audio templates own their controls instead of the catalogue coordinator", async () => {
  const coordinator = await readFile(new URL("audio-library.js", staticRoot), "utf8");
  assert.doesNotMatch(coordinator, /<label id="podcast-state-field"/);
  assert.doesNotMatch(coordinator, /<option value="recently-played"/);
  assert.doesNotMatch(coordinator, /Series \/ volume<\/option>/);
  assert.doesNotMatch(coordinator, /<button id="audio-chapters-toggle"/);
});

test("mobile music navigation keeps a clear action gutter without shrinking touch targets", async () => {
  const [music, rail] = await Promise.all([
    readFile(new URL("music-library.css", staticRoot), "utf8"),
    readFile(new URL("destination-rail.css", staticRoot), "utf8"),
  ]);
  assert.match(music, /@media\(max-width:760px\)\{\.audio-browse-head[^}]*gap:6px/);
  assert.match(music, /\.music-destinations\{gap:0;margin:0\}/);
  assert.match(rail, /\.destination-rail button\{[^}]*min-height:44px/);
  assert.match(rail, /scroll-snap-type:x proximity/);
  assert.match(rail, /touch-action:pan-x/);
});

test("mobile music navigation and actions share one compact touch row", async () => {
  const music = await readFile(new URL("music-library.css", staticRoot), "utf8");
  assert.match(music, /grid-template-areas:\s*"destinations actions"/);
  assert.match(music, /grid-template-columns:\s*minmax\(0,1fr\) auto/);
  assert.match(music, /grid-template-areas:\s*"destinations actions"/);
});

test("desktop publication and music rails share one centered 44px navigation primitive", async () => {
  const [music, publications, rail, shell] = await Promise.all([
    readFile(new URL("music-library.css", staticRoot), "utf8"),
    readFile(new URL("publication-destinations.css", staticRoot), "utf8"),
    readFile(new URL("destination-rail.css", staticRoot), "utf8"),
    readFile(new URL("index.html", staticRoot), "utf8"),
  ]);
  assert.match(rail, /destination-rail button\{[^}]*min-height:44px/);
  assert.match(rail, /destination-rail button\.active::after/);
  assert.match(rail, /destination-rail\.has-destinations-after/);
  assert.doesNotMatch(rail, /destination-rail button:hover\{[^}]*background:/);
  assert.match(shell, /destination-rail\.css/);
  assert.match(music, /grid-template-areas:"balance destinations actions"/);
  assert.match(publications, /publication-destinations\{justify-self:center/);
  assert.doesNotMatch(music, /music-destinations button\.active/);
  assert.doesNotMatch(publications, /publication-destinations button\.active/);
  assert.doesNotMatch(music, /\.music-destinations\{[^}]*border-bottom/);
  assert.match(music, /#audio-index\[data-kind="music"\] \.audio-item\{border-bottom:0\}/);
});

test("music item actions use pointer-aware progressive disclosure", async () => {
  const music = await readFile(new URL("music-library.css", staticRoot), "utf8");
  const desktop = music.match(/@media\(hover:hover\) and \(pointer:fine\)\{[\s\S]*?#audio-index\[data-kind="music"\] \.audio-item>\.audio-more\{opacity:0;[\s\S]*?\.audio-more\[open\]\{opacity:1\}[\s\S]*?\}/)?.[0] || "";

  assert.match(desktop, /\.audio-item:hover>\.audio-more/);
  assert.match(desktop, /\.audio-item:focus-within>\.audio-more/);
  assert.match(desktop, /\.audio-more\[open\]\{opacity:1\}/);
  assert.doesNotMatch(music, /@media\(hover:none\)[^{]*\{[^}]*\.audio-more[^}]*opacity:0/);
});

test("music track information owns its compact dialog stylesheet", async () => {
  const [shared, details, lazy] = await Promise.all([
    readFile(new URL("app.css", staticRoot), "utf8"),
    readFile(new URL("music-track-info.css", staticRoot), "utf8"),
    readFile(new URL("lazy-music-track-info.js", staticRoot), "utf8"),
  ]);
  for (const selector of [".music-track-info", ".music-track-info-card", ".music-track-info-art"]) {
    assert.doesNotMatch(shared, new RegExp(`\\${selector}\\b`));
    assert.match(details, new RegExp(`\\${selector}\\b`));
  }
  assert.match(details, /prefers-reduced-motion/);
  assert.match(details, /100dvh/);
  assert.match(lazy, /music-track-info\.css/);
});

test("video player styles live outside the shared application stylesheet", async () => {
  const [shared, player] = await Promise.all([
    readFile(new URL("app.css", staticRoot), "utf8"),
    readFile(new URL("video-player.css", staticRoot), "utf8"),
  ]);

  for (const selector of [".video-player", ".video-stage", ".video-preview", ".track-menu"]) {
    assert.doesNotMatch(shared, new RegExp(`\\${selector}\\b`));
    assert.match(player, new RegExp(`\\${selector}\\b`));
  }
  assert.match(player, /safe-area-inset-bottom/);
  assert.match(player, /100dvh/);
});

test("activity monitor styles live outside the shared application stylesheet", async () => {
  const [shared, activity] = await Promise.all([
    readFile(new URL("app.css", staticRoot), "utf8"),
    readFile(new URL("activity-panel.css", staticRoot), "utf8"),
  ]);

  for (const selector of [".activity-panel", ".activity-row", ".activity-toggle"]) {
    assert.doesNotMatch(shared, new RegExp(`\\${selector}\\b`));
    assert.match(activity, new RegExp(`\\${selector}\\b`));
  }
  assert.match(activity, /prefers-reduced-motion/);
  assert.match(activity, /safe-area-inset-right/);
});

test("audio player styles live outside the shared application stylesheet", async () => {
  const [shared, player] = await Promise.all([
    readFile(new URL("app.css", staticRoot), "utf8"),
    readFile(new URL("audio-player.css", staticRoot), "utf8"),
  ]);

  for (const selector of [
    ".audio-player",
    ".audio-art-play",
    ".audio-player-more",
    ".audio-chapters",
  ]) {
    assert.doesNotMatch(shared, new RegExp(`\\${selector}\\b`));
    assert.match(player, new RegExp(`\\${selector}\\b`));
  }
  assert.match(player, /safe-area-inset-bottom/);
  assert.match(player, /prefers-reduced-motion/);
  assert.match(player, /100dvh/);
  assert.doesNotMatch(player, /\.reading-item\b|\.video-reading-item\b/);
});

test("Documents catalogue and reader styles own their stylesheet", async () => {
  const [shared, documents] = await Promise.all([
    readFile(new URL("app.css", staticRoot), "utf8"),
    readFile(new URL("document-library.css", staticRoot), "utf8"),
  ]);

  for (const selector of [
    ".document-toolbar",
    ".document-grid",
    ".document-card",
    ".document-row",
    ".document-reader",
    ".document-office-text",
  ]) {
    assert.doesNotMatch(shared, new RegExp(`\\${selector}\\b`));
    assert.match(documents, new RegExp(`\\${selector}\\b`));
  }
  assert.match(documents, /safe-area-inset-bottom/);
  assert.match(documents, /max-width:\s*78ch/);
  assert.match(documents, /text-underline-offset:\s*3px/);
});

test("compact feature controls preserve the shared browser interaction floor", async () => {
  const [deferred, settings, music, player, documents] = await Promise.all([
    readFile(new URL("deferred-route-controllers.js", staticRoot), "utf8"),
    readFile(new URL("settings-controls.css", staticRoot), "utf8"),
    readFile(new URL("music-library.css", staticRoot), "utf8"),
    readFile(new URL("audio-player.css", staticRoot), "utf8"),
    readFile(new URL("document-library.css", staticRoot), "utf8"),
  ]);

  assert.match(deferred, /settings-controls\.css/);
  assert.match(settings, /\.scan-library[\s\S]*?min-height:\s*44px/);
  assert.match(settings, /\.audio-analysis-action button[\s\S]*?min-height:\s*44px/);
  assert.match(music, /\.podcast-storage button[\s\S]*?min-height:\s*44px/);
  assert.match(player, /\.audio-player-favorite,\.audio-player-more>summary\{[^}]*width:44px;height:44px/);
  assert.match(documents, /\.document-more>summary\{width:44px;height:44px/);
});
