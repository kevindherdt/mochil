import assert from "node:assert/strict";
import test from "node:test";
import { nearestPosition, positionFromLocator, resolveRemotePosition } from "../static/readium-sync.js";

const readingOrder = [
  { href: "OEBPS/one.xhtml", type: "application/xhtml+xml" },
  { href: "OEBPS/two %281%29.xhtml", type: "application/xhtml+xml" },
  { href: "OEBPS/three.xhtml", type: "application/xhtml+xml" },
];
const positions = readingOrder.flatMap((link, chapter) =>
  [0, 1, 2, 3].map((page) => ({ href: link.href, locations: { position: chapter * 4 + page + 1, progression: page / 4, totalProgression: (chapter * 4 + page) / 12 } })));

test("a saved position carries the fraction and this engine's locator, never a CFI", () => {
  const locator = { href: "a", locations: { totalProgression: 0.25 }, serialize() { return { href: "a", locations: { totalProgression: 0.25 } }; } };
  const position = positionFromLocator(locator);
  assert.equal(position.fraction, 0.25);
  assert.equal(position.cfi, "");
  assert.deepEqual(JSON.parse(position.locators.readium), { href: "a", locations: { totalProgression: 0.25 } });
  assert.equal(positionFromLocator({ locations: {} }), null);
});

test("the nearest position follows the whole-book fraction", () => {
  assert.equal(nearestPosition(positions, 0.5).locations.position, 7);
  assert.equal(nearestPosition([], 0.5), null);
});

test("an Android or Readium locator wins when its chapter is in this book, even when encoded differently", async () => {
  const remote = { locators: { readium: JSON.stringify({ href: "OEBPS/two%20%281%29.xhtml", locations: { progression: 0.5 } }) }, web_anchor: { spine_index: 0, exact: "x".repeat(30) }, fraction: 0.9 };
  const result = await resolveRemotePosition({ remote, readingOrder, positions, findText: async () => 0.1 });
  assert.equal(result.accuracy, "exact");
  assert.equal(result.locator.href, "OEBPS/two %281%29.xhtml");
});

test("a locator for a chapter this book lacks falls back to the quoted text, searching nearby chapters first", async () => {
  const asked = [];
  const remote = { locators: { readium: JSON.stringify({ href: "OEBPS/other.xhtml" }) }, web_anchor: { spine_index: 1, exact: "y".repeat(30) }, fraction: 0.9 };
  const result = await resolveRemotePosition({
    remote, readingOrder, positions,
    findText: async (href) => { asked.push(href); return href === "OEBPS/three.xhtml" ? 0.5 : null; },
  });
  assert.deepEqual(asked, ["OEBPS/two %281%29.xhtml", "OEBPS/one.xhtml", "OEBPS/three.xhtml"]);
  assert.equal(result.accuracy, "text");
  assert.equal(result.locator.href, "OEBPS/three.xhtml");
  assert.equal(result.locator.locations.progression, 0.5);
  assert.equal(result.locator.locations.totalProgression, positions[10].locations.totalProgression);
});

test("generated image pages skip text matching and use the fraction", async () => {
  const pages = [{ href: "_bobo/page/1-1.xhtml" }, { href: "_bobo/page/1-2.xhtml" }];
  const pagePositions = pages.map((link, i) => ({ href: link.href, locations: { totalProgression: i / 2 } }));
  let searched = false;
  const result = await resolveRemotePosition({ remote: { web_anchor: { spine_index: 0, exact: "z".repeat(30) }, fraction: 0.5 }, readingOrder: pages, positions: pagePositions, findText: async () => { searched = true; return 0.2; } });
  assert.equal(searched, false);
  assert.equal(result.accuracy, "approximate");
  assert.equal(result.locator.href, "_bobo/page/1-2.xhtml");
});

test("with nothing usable there is nothing to resolve", async () => {
  assert.equal(await resolveRemotePosition({ remote: {}, readingOrder, positions, findText: async () => null }), null);
});

test("a locator another reader has since moved past is ignored in favour of the quoted text", async () => {
  const remote = {
    fraction: 0.8,
    locators: { readium: JSON.stringify({ href: "OEBPS/one.xhtml", locations: { totalProgression: 0.1 } }) },
    web_anchor: { spine_index: 2, exact: "w".repeat(30) },
  };
  const result = await resolveRemotePosition({ remote, readingOrder, positions, findText: async (href) => (href === "OEBPS/three.xhtml" ? 0.25 : null) });
  assert.equal(result.accuracy, "text");
  assert.equal(result.locator.href, "OEBPS/three.xhtml");
  const current = await resolveRemotePosition({ remote: { ...remote, fraction: 0.1002 }, readingOrder, positions, findText: async () => null });
  assert.equal(current.accuracy, "exact");
});
