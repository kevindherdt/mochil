import assert from "node:assert/strict";
import test from "node:test";
import { fitHeight, heightFor } from "../static/fit-height.js";

test("the box ends a margin above the bottom of the window", () => {
  assert.equal(heightFor(300, 900), 584);
  assert.equal(heightFor(300, 900, { margin: 0 }), 600);
});

test("a short window still shows a few rows", () => {
  assert.equal(heightFor(300, 400), 240);
  assert.equal(heightFor(300, 400, { minimum: 120 }), 120);
});

test("fitHeight sets the height now and again when the window is resized, counting how far the page is scrolled", () => {
  let resize;
  const view = { innerHeight: 800, scrollY: 50, addEventListener: (type, fn) => { if (type === "resize") resize = fn; } };
  const element = { style: {}, getBoundingClientRect: () => ({ top: 250 }) }; // 250 on screen, 300 on the page
  const apply = fitHeight(element, view);
  apply();
  assert.equal(element.style.maxHeight, "484px");
  view.innerHeight = 1000;
  resize();
  assert.equal(element.style.maxHeight, "684px");
});
