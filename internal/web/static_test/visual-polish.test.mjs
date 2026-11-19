// Three small visual promises, each one a thing the eye notices before any code does:
// a row of "Jump back in" cards shares one height, an episode title can be read in full,
// and an empty shelf centres its panel instead of hugging the top.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const read = (name) => readFileSync(new URL(`../static/${name}`, import.meta.url), "utf8");

test("every jump card carries the read-along slot, so one badge cannot make a row uneven", () => {
  const source = read("jump-back-in.js");
  // The slot exists for every card, and only books with a recording put anything in it.
  assert.match(source, /<span class="jump-badge">\$\{type === 'book' \? readAlongMarkup\(item, 'label'\) : ''\}<\/span>/);
  assert.match(read("dashboard.css"), /\.jump-card \.jump-badge\{display:flex;align-items:center;min-height:20px\}/);
});

test("an episode title carries its full text and wraps to two lines", () => {
  const source = read("video-library.js");
  assert.ok(
    (source.match(/<strong title="\$\{escapeHTML\(displayTitle\)\}">/g) || []).length >= 2,
    "both the episode card and the standard card title need hover text",
  );
  const css = read("video-polish.css");
  assert.match(css, /\.video-episode-card \.video-copy strong\{[^}]*-webkit-line-clamp:2[^}]*white-space:normal/);
});

test("an empty shelf centres its panel and states it in readable text", () => {
  const css = read("ui-polish.css");
  const rule = css.match(/\.state-panel\{[^}]*\}/)?.[0] || "";
  assert.match(rule, /align-content:center/, "the panel should centre inside the height it takes");
  assert.match(rule, /min-height:min\(56vh,420px\)/, "it needs a height to centre in");
  assert.doesNotMatch(rule, /margin:14vh auto 0/, "the old top-hugging margin should be gone");
  const body = css.match(/\.state-panel p\{[^}]*\}/)?.[0] || "";
  assert.doesNotMatch(body, /color:var\(--muted\)/, "body copy should not be the faintest tone on the page");
  assert.match(body, /color-mix\(in srgb,var\(--text\) 76%,transparent\)/);
});
