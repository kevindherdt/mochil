import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const staticRoot = new URL("../static/", import.meta.url);

// Decorative box-shadow declarations written with hardcoded colors (not the --shadow-N
// tokens) are NOT covered by design-tokens.css'/theme-tokens.css' eink zeroing, and keep
// rendering their blur/glow on the e-ink palette. Each one fixed here is split out into its
// own `:root:not([data-palette=eink]) <selector>{box-shadow:...}` rule — the same technique
// already used by book-list.css's .reading-card:hover, dashboard.css's .jump-card:hover and
// music-library.css's .music-track-card:hover — so the shadow still renders for every other
// palette but is suppressed for eink. The original rule keeps every other property.

function escapeRegExp(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

async function load(file) {
  return readFile(new URL(file, staticRoot), "utf8");
}

// Asserts a `:root:not([data-palette=eink]) <selector>{...box-shadow...}` rule exists.
function assertEinkExcluded(css, selector, label) {
  const pattern = new RegExp(
    `:root:not\\(\\[data-palette=eink\\]\\)\\s*${escapeRegExp(selector)}\\s*\\{[^}]*box-shadow`,
  );
  assert.match(css, pattern, `${label}: expected a :not([data-palette=eink]) box-shadow rule for ${selector}`);
}

const cases = [
  ["activity-panel.css", ".activity-toggle.active .activity-dot"],
  ["activity-panel.css", ".activity-panel"],
  ["audio-player.css", ".audio-player"],
  ["audio-player.css", ".audio-player.expanded"],
  ["audio-player.css", ".audio-player.expanded .audio-art-play"],
  ["audio-player.css", ".audio-player.expanded .audio-expanded-transport #audio-play-large"],
  ["audio-player.css", ".audio-player:not(.expanded) .audio-bar-skip .audio-bar-play"],
  ["audio-player-wide.css", ".audio-player.expanded:has(.audio-player-queue:not([hidden])) .audio-art-play"],
  ["audio-player-wide.css", ".audio-player.expanded:has(.audio-player-queue:not([hidden])) .audio-vinyl"],
  ["audio-player-wide.css", ".audio-player.expanded:has(.audio-player-queue:not([hidden]))>.audio-seek::-webkit-slider-thumb"],
  ["audio-player-wide.css", ".audio-player.expanded:has(.audio-player-queue:not([hidden])) .audio-expanded-transport #audio-play-large"],
  ["book-actions.css", ".book-check"],
  ["book-list.css", ".series-list .series-row>summary>.series-row-cover"],
  ["book-list.css", ".book-list .book-list-cover"],
  ["book-list.css", ".reading-cover"],
  ["card-surface.css", ".cover-tag"],
  ["collection-rows.css", ".collection-row-cover"],
  ["document-library.css", ".document-paper"],
  ["document-library.css", ".document-open:hover .document-paper"],
  ["editor.css", "dialog"],
  ["editor.css", ".ed-columns-list"],
  ["editor.css", ".ed-grid td[data-field]:hover"],
  ["hermes.css", ".hm-voice"],
  ["hermes.css", ".hm-side.open"],
  ["hermes.css", ".hm-stage"],
  ["hermes.css", ".hm-vm-mic"],
  ["hermes.css", ".hm-vm-mic.on"],
  ["hermes.css", ".hm-quickmenu"],
  ["hermes.css", ".hm-memory"],
  ["login.css", "section"],
  ["login.css", ".wordmark span"],
  ["login.css", "button"],
  ["publication-destinations.css", ".layout-toggle button[aria-pressed=true]"],
  ["series-hub.css", ".series-hub-cover"],
  ["series-hub.css", ".series-hub-primary"],
  ["series-stack.css", ".series-cover-grid .series-card.is-stack .cover-frame"],
  ["settings-unified.css", "#settings-view .nv-more .nv-actions"],
  ["ui-polish.css", "#status.catalog-status.toast-visible"],
  ["vault.css", ".vault-workspace>aside"],
  ["vault-live.css", ".cm-tooltip.cm-tooltip-autocomplete"],
  ["vault-nav.css", ".vault-switcher-box"],
  ["vault-nav.css", ".vault-graph-modes button[aria-pressed=true]"],
  ["video-controls.css", ".video-controls"],
  ["video-controls.css", ".video-seek::-webkit-slider-thumb"],
  ["video-controls.css", "#video-play"],
  ["video-controls.css", ".video-centre-play"],
  ["video-player.css", ".video-preview"],
];

for (const [file, selector] of cases) {
  test(`${file} ${selector} decorative box-shadow is suppressed for the e-ink palette`, async () => {
    const css = await load(file);
    assertEinkExcluded(css, selector, file);
  });
}

// .hm-vm-mic.on's box-shadow fuses two layers: a crisp, non-blurred 6px ring (the only visual
// signal that voice input is actively listening) and a blurred 22px drop shadow (pure
// decoration, same glow already excluded on .hm-vm-mic itself). Unlike the cases above, this
// selector must have an UNCONDITIONAL rule carrying the ring so the "listening" state is never
// lost on e-ink, in addition to the eink-excluded rule carrying both layers for every other palette.
test("hermes.css .hm-vm-mic.on keeps its listening ring unconditionally, even on the e-ink palette", async () => {
  const css = await load("hermes.css");
  const unconditional = css.match(/(?<!:not\(\[data-palette=eink\]\)\s)\.hm-vm-mic\.on\s*\{([^}]*)\}/);
  assert.ok(unconditional, "expected an unconditional .hm-vm-mic.on rule");
  assert.match(
    unconditional[1],
    /box-shadow:\s*0 0 0 6px/,
    ".hm-vm-mic.on's unconditional rule must still set the crisp listening ring regardless of palette",
  );
  assert.doesNotMatch(
    unconditional[1],
    /22px/,
    ".hm-vm-mic.on's unconditional rule must not carry the blurred decorative glow (that belongs only in the eink-excluded rule)",
  );
});
