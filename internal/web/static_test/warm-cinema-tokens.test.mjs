// Warm cinema: four hues, one per kind of media, and each chip's text has to be readable on it.
// This is the part of the visual direction that is a measurement rather than a taste call.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const css = readFileSync(new URL("../static/card-surface.css", import.meta.url), "utf8");

const token = (name) => css.match(new RegExp(`--kind-${name}:\\s*(#[0-9a-f]{6})`))?.[1];
const ink = (name) => css.match(new RegExp(`--kind-${name}-ink:\\s*(#[0-9a-f]{6})`))?.[1];

function luminance(hex) {
  const channel = (value) => {
    const c = value / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  const [r, g, b] = [1, 3, 5].map((at) => parseInt(hex.slice(at, at + 2), 16));
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

const contrast = (a, b) => {
  const [x, y] = [luminance(a), luminance(b)].sort((m, n) => n - m);
  return (x + 0.05) / (y + 0.05);
};

const KINDS = ["book", "video", "audio", "podcast"];

test("books take the amber of warm cinema, and the other kinds keep their hues", () => {
  assert.equal(token("book"), "#e0a458");
  assert.equal(token("video"), "#b69cfb");
  assert.equal(token("audio"), "#4fd1b8");
  assert.equal(token("podcast"), "#f48fb8");
});

test("every chip reads at 4.5:1 or better", () => {
  for (const kind of KINDS) {
    const ratio = contrast(token(kind), ink(kind));
    assert.ok(ratio >= 4.5, `${kind} chip contrast is ${ratio.toFixed(2)}:1, needs 4.5:1`);
  }
});

test("the book family reads its colour from the token instead of a hardcoded blue", () => {
  const kinds = readFileSync(new URL("../static/kind-colors.css", import.meta.url), "utf8");
  const rule = kinds.match(/\(\[data-kind=books\][^}]*\}/)?.[0] || "";
  assert.match(rule, /--kind:var\(--kind-book\)/, "books should follow the palette token");
  assert.match(rule, /--kind-ink:var\(--kind-book-ink\)/);
  assert.doesNotMatch(kinds, /#7fb2f7/, "the old blue should be gone from the kind colours");
});

test("the read-along slot is as tall as the badge it holds", () => {
  const css = readFileSync(new URL("../static/dashboard.css", import.meta.url), "utf8");
  assert.match(css, /\.jump-card \.jump-badge\{[^}]*min-height:20px/);
});

test("no hue is used as light text on the warm dark background below 4.5:1", () => {
  // --kind-* is also drawn as text on --bg (#171513 on the dark warm palettes).
  for (const kind of KINDS) {
    const ratio = contrast(token(kind), "#171513");
    assert.ok(ratio >= 4.5, `${kind} text on the dark background is ${ratio.toFixed(2)}:1`);
  }
});
