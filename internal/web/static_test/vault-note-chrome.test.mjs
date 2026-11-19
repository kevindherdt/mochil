// The note chrome has to stay put and stay whole: the tab strip names the document you are in,
// and the breadcrumb row under it must never be covered by it.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const nav = readFileSync(new URL("../static/vault-nav.css", import.meta.url), "utf8");
const vault = readFileSync(new URL("../static/vault.css", import.meta.url), "utf8");

const rule = (css, selector) => css.match(new RegExp(selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&") + "\\{[^}]*\\}"))?.[0] || "";

test("the note bar is offset by the tab strip, at the specificity that can win", () => {
  // The pane rule sets top:0 with four classes; a three-class selector loses to it, which is what
  // let the tab strip draw over the breadcrumb.
  const offset = rule(nav, ".vault-workspace .vault-pane.vault-document.has-tabs>header.vault-notebar");
  assert.match(offset, /top:var\(--vault-tabs-h,38px\)/);
  assert.doesNotMatch(nav, /^\.vault-pane\.has-tabs>header\.vault-notebar\{top:38px\}$/m);
});

test("the tab strip and the note bar read one height", () => {
  assert.match(rule(nav, ".vault-pane.vault-document"), /--vault-tabs-h:38px/);
  assert.match(rule(nav, ".vault-tabs"), /height:var\(--vault-tabs-h,38px\)/);
});

test("the note pane fills its box instead of the window", () => {
  // A note pane is a <main>, so an unscoped body.vault-reading main rule sized it to the window and
  // its bottom hung past the workspace, clipped by overflow:hidden.
  assert.doesNotMatch(vault, /^body\.vault-reading main\{/m);
  assert.match(vault, /body\.vault-reading>main\{height:calc\(/);
  assert.match(rule(nav, ".vault-pane.vault-document"), /height:100%;min-height:0/);
});
