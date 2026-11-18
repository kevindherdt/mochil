import test from "node:test";
import assert from "node:assert/strict";
import { createMemory, relativeDate } from "../static/hermes-memory.js";

const NOW = Date.parse("2026-11-03T22:00:00Z");

test("dates read as today, yesterday, days ago and then a plain date", () => {
  assert.equal(relativeDate("2026-11-03T08:00:00Z", NOW), "Today");
  assert.equal(relativeDate("2026-11-02T08:00:00Z", NOW), "Yesterday");
  assert.equal(relativeDate("2026-10-28T08:00:00Z", NOW), "6 days ago");
  assert.match(relativeDate("2025-01-05T08:00:00Z", NOW), /2025/);
  assert.equal(relativeDate("not a date", NOW), "");
  assert.equal(relativeDate("", NOW), "");
});

// A tiny stand-in for the few elements the module touches.
function fakeDom() {
  const make = () => ({ hidden: false, textContent: "", children: [], attrs: {}, onclick: null,
    setAttribute(k, v) { this.attrs[k] = v; }, replaceChildren() { this.children = []; }, append(...n) { this.children.push(...n); } });
  const els = { "#hm-mem-open": make(), "#hm-memory": make(), "#hm-mem-list": make(), "#hm-mem-note": make(), "#hm-mem-close": make(), "#hm-mem-refresh": make() };
  els["#hm-memory"].hidden = true;
  const document = { createElement: (tag) => ({ tag, textContent: "", className: "", children: [], append(...n) { this.children.push(...n); }, set innerHTML(v) { this._html = v; this.lastChild = { textContent: "Remember" }; } }) };
  return { els, document, $: (s) => els[s] };
}

test("the sidebar link only shows when memory is available, and opening lists what she remembers", async () => {
  const { els, document, $ } = fakeDom();
  const calls = [];
  const api = async (path) => { calls.push(path); return { total: 2, items: [{ id: "a", text: "Kevin lives in Belgium", date: "2026-11-03T08:00:00Z" }, { id: "b", text: "Likes green" }] }; };
  let opened = 0;
  const memory = createMemory({ document, $, api, name: () => "Lyana", onOpen: () => { opened += 1; } });
  memory.setEnabled(false);
  assert.equal(els["#hm-mem-open"].hidden, true);
  memory.setEnabled(true);
  assert.equal(els["#hm-mem-open"].hidden, false);
  els["#hm-mem-open"].onclick();
  await new Promise((r) => setImmediate(r));
  assert.deepEqual(calls, ["/api/hermes/memories"]);
  assert.equal(els["#hm-memory"].hidden, false);
  assert.equal(els["#hm-mem-list"].children.length, 2);
  assert.match(els["#hm-mem-note"].textContent, /2 things Lyana has picked up/);
  assert.equal(opened, 1, "opening the panel lets the page close the phone sidebar");
  els["#hm-mem-close"].onclick();
  assert.equal(els["#hm-memory"].hidden, true);
});

test("an empty list and a failure each say so in plain words", async () => {
  const { els, document, $ } = fakeDom();
  let mode = "empty";
  const api = async () => { if (mode === "fail") throw new Error("The memory server is not reachable"); return { items: [] }; };
  const memory = createMemory({ document, $, api, name: () => "Lyana" });
  memory.setEnabled(true);
  els["#hm-mem-open"].onclick();
  await new Promise((r) => setImmediate(r));
  assert.match(els["#hm-mem-note"].textContent, /has not remembered anything yet/);
  mode = "fail";
  els["#hm-mem-refresh"].onclick();
  await new Promise((r) => setImmediate(r));
  assert.equal(els["#hm-mem-note"].textContent, "The memory server is not reachable");
});

test("Remember sends the message text once and reports the result", async () => {
  const { document, $ } = fakeDom();
  const sent = [];
  let fail = false;
  const api = async (path, options) => { if (fail) throw new Error("x"); sent.push([path, JSON.parse(options.body)]); return { ok: true }; };
  const memory = createMemory({ document, $, api });
  const button = memory.button(() => "  My sister is Anna ");
  await button.onclick();
  assert.deepEqual(sent, [["/api/hermes/remember", { text: "My sister is Anna" }]]);
  assert.equal(button.lastChild.textContent, "Sent to memory");
  assert.equal(button.disabled, true);
  await button.onclick(); // already saved: no second request
  assert.equal(sent.length, 1);
  const again = memory.button(() => "   ");
  await again.onclick();
  assert.equal(sent.length, 1, "blank text is never sent");
  fail = true;
  const broken = memory.button(() => "Hello");
  await broken.onclick();
  assert.equal(broken.lastChild.textContent, "Could not save");
  assert.equal(broken.disabled, false);
});
