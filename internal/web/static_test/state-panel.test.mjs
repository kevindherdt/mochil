import assert from "node:assert/strict";
import test from "node:test";
import { describeProblem, emptySpec, errorSpec, renderStatePanel, statePanelHTML } from "../static/state-panel.js";

test("an empty shelf explains itself and points to settings", () => {
  const spec = emptySpec({ noun: "books" });
  assert.equal(spec.title, "This shelf is empty");
  assert.deepEqual(spec.actions.map((a) => a.id), ["settings"]);
});

test("a search with no results names the query and offers to clear it", () => {
  const spec = emptySpec({ noun: "series", query: "  dragon " });
  assert.match(spec.title, /No matches for “dragon”/);
  assert.deepEqual(spec.actions.map((a) => a.id), ["clear-search"]);
});

test("an empty letter asks for another letter instead of blaming the library", () => {
  const spec = emptySpec({ noun: "authors", bucket: "Q" });
  assert.match(spec.title, /Nothing under “Q”/);
  assert.equal(spec.actions.length, 0);
});

test("errors are human, announced, and always offer a retry", () => {
  assert.match(describeProblem({ status: 503 }), /server hit a problem/i);
  assert.match(describeProblem({ message: "Failed to fetch" }), /offline/i);
  assert.match(describeProblem({ status: 401 }), /Sign in/);
  const spec = errorSpec({ status: 500 }, "your books");
  assert.equal(spec.tone, "error");
  assert.match(statePanelHTML(spec), /role="alert"/);
  assert.match(statePanelHTML(spec), /data-state-action="retry"/);
});

test("panel text is escaped and actions reach their handlers", () => {
  const html = statePanelHTML({ title: '<img src=x onerror=1>', actions: [{ id: "retry", label: "Go" }] });
  assert.doesNotMatch(html, /<img/);
  let clicked = "";
  let listener;
  const button = { dataset: { stateAction: "retry" } };
  const panel = { addEventListener: (_t, fn) => (listener = fn) };
  const target = { querySelector: () => panel, set innerHTML(v) {} };
  renderStatePanel(target, { title: "x", actions: [{ id: "retry", label: "Go" }] }, { retry: () => (clicked = "retry") });
  listener({ target: { closest: () => button } });
  assert.equal(clicked, "retry");
});

test("a network failure names the server and points at VPN or DNS", () => {
  const text = describeProblem(new Error("Failed to fetch"), "opds.example.net");
  assert.match(text, /opds\.example\.net/);
  assert.match(text, /VPN|DNS/);
});
