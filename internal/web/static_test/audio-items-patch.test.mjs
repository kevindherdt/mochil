import assert from "node:assert/strict";
import test from "node:test";
import { changedSections } from "../static/audio-items-patch.js";

test("only the sections whose markup changed are listed", () => {
  assert.deepEqual(changedSections(["a", "b", "c"], ["a", "B", "c"]), [1]);
  assert.deepEqual(changedSections(["a", "b"], ["a", "b"]), []);
  assert.deepEqual(changedSections(["a", "b", "c"], ["A", "b", "C"]), [0, 2]);
});

test("a section that was never remembered counts as changed", () => {
  assert.deepEqual(changedSections([undefined, "b"], ["a", "b"]), [0]);
});

test("a different number of sections means replace everything", () => {
  assert.equal(changedSections(["a"], ["a", "b"]), null);
  assert.equal(changedSections(["a", "b"], ["a"]), null);
});

import { morphChildren } from "../static/audio-items-patch.js";

// A tiny stand-in for the DOM: elements with attributes and children, and the few operations the patch uses.
function el(tag, attrs = {}, children = []) {
  const node = {
    tagName: tag.toUpperCase(), parent: null, attrs: new Map(Object.entries(attrs)), children,
    get childNodes() { return this.children; },
    get attributes() { return [...this.attrs].map(([name, value]) => ({ name, value })); },
    hasAttribute(name) { return this.attrs.has(name); }, getAttribute(name) { return this.attrs.get(name); },
    setAttribute(name, value) { this.attrs.set(name, value); }, removeAttribute(name) { this.attrs.delete(name); },
    get outerHTML() { return `<${tag} ${[...this.attrs].map(([k, v]) => `${k}=${v}`).join(" ")}>${this.children.map((c) => c.outerHTML).join("")}</${tag}>`; },
    replaceWith(next) { const list = this.parent.children; next.parent = this.parent; list[list.indexOf(this)] = next; this.parent = null; },
    replaceChildren(...nodes) { this.children = nodes; nodes.forEach((n) => { n.parent = this; }); },
  };
  children.forEach((child) => { child.parent = node; });
  return node;
}
const card = (id, playing = false) => el("article", { "data-item": id, class: playing ? "playing" : "" }, [el("span", { class: "art" }, [el("img", { src: `/art/${id}` })])]);
const grid = (...cards) => el("div", { class: "grid" }, cards);
// Fills the page the way the first render does: whole replace, then remember what each row was built from.
function shelf(...cards) {
  const container = el("section");
  morphChildren(container, el("x", {}, [grid(...cards)]));
  return container;
}

test("playing a song replaces only the row whose markup changed and leaves the other covers alone", () => {
  const container = shelf(card("a"), card("b"), card("c"));
  const wrapper = container.children[0], [a, b, c] = wrapper.children;
  morphChildren(container, el("x", {}, [grid(card("a"), card("b", true), card("c"))]));
  assert.equal(container.children[0], wrapper, "the grid stays");
  assert.equal(wrapper.children[0], a);
  assert.notEqual(wrapper.children[1], b, "the row that changed is replaced");
  assert.equal(wrapper.children[1].getAttribute("class"), "playing");
  assert.equal(wrapper.children[2], c);
});

test("nothing is touched when nothing changed, even if the page has since changed the covers", () => {
  const container = shelf(card("a"), card("b"));
  const wrapper = container.children[0], [a, b] = wrapper.children;
  a.children[0].children[0].setAttribute("class", "loaded"); // the cover finished loading
  morphChildren(container, el("x", {}, [grid(card("a"), card("b"))]));
  assert.deepEqual([wrapper.children[0], wrapper.children[1]], [a, b]);
  assert.equal(a.children[0].children[0].getAttribute("class"), "loaded");
});

test("a different number of rows replaces the list", () => {
  const container = shelf(card("a"), card("b"));
  const old = container.children[0].children[0];
  morphChildren(container, el("x", {}, [grid(card("a"), card("b"), card("c"))]));
  assert.equal(container.children[0].children.length, 3);
  assert.notEqual(container.children[0].children[0], old);
});

test("rows are remembered after a replace too, so the next change still leaves the others alone", () => {
  const container = shelf(card("a"), card("b"));
  morphChildren(container, el("x", {}, [grid(card("a"), card("b"), card("c"))]));
  const [a, b, c] = container.children[0].children;
  morphChildren(container, el("x", {}, [grid(card("a"), card("b"), card("c", true))]));
  const now = container.children[0].children;
  assert.deepEqual([now[0], now[1]], [a, b]);
  assert.notEqual(now[2], c);
});
