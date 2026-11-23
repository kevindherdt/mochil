// A card takes its colour from its own artwork, so a shelf reads as a row of coloured objects.
// The measurement and the sampling rules are what matter: only on-screen cards are read, a missing
// or unreadable image leaves the card's plain surface, and the colour is mixed in at a strength that
// cannot fight the text.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const read = (name) => readFileSync(new URL(`../static/${name}`, import.meta.url), "utf8");
const { createCardTint } = await import("../static/card-tint.js");
const { dominantColour } = await import("../static/audio-player-tint.js");

const card = () => {
  const element = {
    "data-card": true,
    style: { values: {}, setProperty: (name, value) => { element.style.values[name] = value } },
    matches: () => true,
    querySelector: () => element.image,
    querySelectorAll: () => [],
    image: null,
  };
  return element;
};

const fakeImage = (pixels) => ({ complete: true, naturalWidth: 16, src: "cover.jpg", addEventListener() {}, pixels });

// jsdom is not available here, so the canvas is stubbed to return the pixels the test chose.
globalThis.document = {
  createElement: () => ({
    width: 0,
    height: 0,
    getContext: () => ({ drawImage() {}, getImageData: () => ({ data: globalThis.__pixels || [] }) }),
  }),
};

test("a colourful cover gives the card a tint, greys and blacks do not", () => {
  const red = new Uint8ClampedArray([220, 40, 30, 255, 225, 45, 35, 255]);
  assert.match(dominantColour(red), /^rgb\(22\d /);
  const grey = new Uint8ClampedArray([128, 128, 128, 255, 240, 240, 240, 255]);
  assert.equal(dominantColour(grey), "");
});

test("a card that is on screen gets --art-tint from its image", () => {
  globalThis.__pixels = [200, 60, 40, 255, 210, 70, 50, 255];
  const element = card();
  element.image = fakeImage(globalThis.__pixels);
  const tint = createCardTint({ root: element, Observer: null, MutationObserverClass: null });
  tint.scan({ nodeType: 1, matches: () => false, querySelectorAll: () => [element] });
  assert.match(element.style.values["--art-tint"], /^rgb\(/);
});

test("a shelf is painted at once, a large grid waits for the screen", () => {
  globalThis.__pixels = [200, 60, 40, 255, 210, 70, 50, 255];
  let observed = 0, unobserved = 0, entryHandler = null;
  const Observer = class {
    constructor(handler) { entryHandler = handler }
    observe() { observed += 1 }
    unobserve() { unobserved += 1 }
  };
  const grid = Array.from({ length: 40 }, () => Object.assign(card(), { image: fakeImage(globalThis.__pixels) }));
  const tint = createCardTint({ root: grid[0], Observer, MutationObserverClass: null });

  tint.scan({ nodeType: 1, matches: () => false, querySelectorAll: () => grid });
  assert.equal(observed, 40, "a big grid is watched, not sampled at once");
  assert.equal(grid[0].style.values["--art-tint"], undefined);

  const shelf = [card(), card()];
  shelf.forEach((one) => { one.image = fakeImage(globalThis.__pixels) });
  tint.scan({ nodeType: 1, matches: () => false, querySelectorAll: () => shelf });
  assert.equal(observed, 40, "a shelf of a few cards is painted instead of watched");
  assert.match(shelf[0].style.values["--art-tint"], /^rgb\(/);

  entryHandler([{ isIntersecting: true, target: grid[0] }]);
  assert.match(grid[0].style.values["--art-tint"], /^rgb\(/);
  assert.equal(unobserved, 1, "once sampled, a card is no longer watched");
});

test("a card without a readable image keeps its plain surface", () => {
  globalThis.__pixels = [];
  const element = card();
  element.image = fakeImage([]);
  const tint = createCardTint({ root: element, Observer: null, MutationObserverClass: null });
  tint.scan(element);
  assert.equal(element.style.values["--art-tint"], undefined);
});

test("a card is a flat surface, and its edge cannot fight the text", () => {
  const css = read("card-surface.css");
  const rule = css.match(/:is\([\s\S]*?\)\s*\{[\s\S]*?\}/)?.[0] || "";
  // The artwork wash over every card was a second gradient on the page; the sheet's cards are flat.
  assert.doesNotMatch(rule, /radial-gradient/);
  assert.match(rule, /background: var\(--card-bg\)/);
  assert.match(css, /--card-bg: color-mix\(in srgb, var\(--surface\) 90%, var\(--text\) 5%\)/);
  assert.match(css, /--card-edge: inset 0 1px 0/);
  assert.match(read("dashboard.css"), /\.jump-card\{[^}]*var\(--art-tint/);
  assert.match(read("app.js"), /createCardTint\(\{ root: document\.body \}\)/);
});

test("a cover swapped in after the card was drawn is picked up", () => {
  globalThis.__pixels = [30, 120, 200, 255, 40, 130, 210, 255];
  let notify = null;
  const MutationObserverClass = class { constructor(handler) { notify = handler } observe() {} };
  const element = card();
  element.image = null;                       // the card was painted before its cover existed
  createCardTint({ root: { nodeType: 1, matches: () => false, querySelectorAll: () => [] }, Observer: null, MutationObserverClass });

  // The cover loader inserts a sized <img> that has not decoded yet.
  let onLoad = null;
  const image = {
    nodeType: 1, tagName: "IMG", complete: false, naturalWidth: 0, src: "sized-cover.jpg",
    closest: () => element,
    addEventListener: (_type, handler) => { onLoad = handler },
  };
  element.image = image;
  notify([{ addedNodes: [image] }]);
  assert.equal(element.style.values["--art-tint"], undefined, "an undecoded cover is not sampled");

  image.complete = true; image.naturalWidth = 16;
  onLoad();
  assert.match(element.style.values["--art-tint"], /^rgb\(/);
});

test("every card that wears the shared card look also takes its artwork colour", () => {
  const surface = read("card-surface.css");
  const list = surface.match(/:is\(([\s\S]*?)\)\s*\{/)?.[1] || "";
  const classes = [...list.matchAll(/\.([a-z-]+(?:-card|-group-card|book|track-card))\b/g)].map((m) => m[1]);
  const unique = [...new Set(classes)];
  assert.ok(unique.length >= 5, `expected the card list to name its cards, found ${unique.length}`);
  const tint = read("card-tint.js");
  for (const name of unique) {
    assert.ok(tint.includes(`.${name}`), `${name} wears the card look but is never tinted`);
  }
});

test("small text is half a step up from the 11px it used to be", () => {
  const tokens = read("design-tokens.css");
  assert.match(tokens, /--fs-11:11\.5px/);
  assert.match(tokens, /--fs-12:12\.5px/);
});
