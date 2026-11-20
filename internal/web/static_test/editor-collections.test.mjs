import assert from "node:assert/strict";
import test from "node:test";
import { collectionBody, collectionRequest, fillCollectionFilter } from "../static/editor-collections.js";

const form = (values) => ({ get: (name) => values[name] ?? null });
const ids = ["a", "b"];

test("adding to an existing collection sends its id and the picked books", () => {
  assert.deepEqual(collectionRequest(form({ mode: "add", collection: "c1" }), "books", ids).body, { library: "books", collection: "c1", ids, remove: false });
});

test("taking books out sends remove", () => {
  assert.equal(collectionRequest(form({ mode: "remove", collection: "c1" }), "books", ids).body.remove, true);
});

test("a new collection needs a name, and cannot be a take-out", () => {
  assert.match(collectionRequest(form({ mode: "add", collection: "__new__", name: "  " }), "books", ids).problem, /name/);
  assert.deepEqual(collectionRequest(form({ mode: "add", collection: "__new__", name: " Reread " }), "books", ids).body, { library: "books", name: "Reread", ids });
  assert.ok(collectionRequest(form({ mode: "remove", collection: "__new__", name: "x" }), "books", ids).problem);
});

test("the dialog lists only collections that can be changed by hand, and says why the others are missing", () => {
  const html = collectionBody([{ id: "1", name: "Mine", smart: false, count: 3 }, { id: "2", name: "Auto", smart: true, count: 9 }]);
  assert.match(html, /Mine \(3\)/);
  assert.doesNotMatch(html, /Auto \(/);
  assert.match(html, /fill themselves/);
  assert.match(collectionBody([{ id: "2", name: "Auto", smart: true, count: 9 }]), /value="remove" disabled/);
});

test("the collection list is hidden when there are none, and marks the chosen one", () => {
  const bar = { hidden: false };
  const select = { parentElement: bar, innerHTML: "" };
  fillCollectionFilter(select, [], "");
  assert.equal(bar.hidden, true);
  fillCollectionFilter(select, [{ id: "1", name: "A <b>", smart: true, count: 2 }], "1");
  assert.equal(bar.hidden, false);
  assert.match(select.innerHTML, /selected>A &lt;b&gt; \(smart\) · 2/);
});

import { changesFrom, manageBody, manageCollections } from "../static/editor-collections-manage.js";

const smart = { id: "s1", name: "Herbert", count: 2, rule: { author: "Frank Herbert", added_days: 0 } };
const picked = { id: "p1", name: "Mine", count: 3 };
const submitted = (values) => ({ get: (name) => (name in values ? values[name] : null) });
const untouched = { "name:s1": "Herbert", "rule:s1:author": "Frank Herbert", "rule:s1:status": "", "name:p1": "Mine" };

test("saving the manage dialog without touching anything changes nothing", () => {
  assert.deepEqual(changesFrom(submitted(untouched), [smart, picked]), []);
});

test("a rename, a changed search and a delete each become one request", () => {
  const changes = changesFrom(submitted({ ...untouched, "name:p1": " Reread ", "rule:s1:author": "", "rule:s1:query": "messiah", "rule:s1:added_days": "30", "rule:s1:status": "unread" }), [smart, picked]);
  assert.deepEqual(changes, [
    { method: "PATCH", id: "s1", body: { rule: { query: "messiah", author: "", series: "", tag: "", format: "", status: "unread", added_days: 30, missing_volumes: false } } },
    { method: "PATCH", id: "p1", body: { name: "Reread" } },
  ]);
  assert.deepEqual(changesFrom(submitted({ ...untouched, "delete:p1": "on", "name:p1": "Other" }), [smart, picked]), [{ method: "DELETE", id: "p1" }], "a deleted collection is not also renamed");
});

test("only smart collections show a search, and names are escaped", () => {
  const html = manageBody([smart, { ...picked, name: "<b>" }]);
  assert.equal((html.match(/class="ed-rule"/g) || []).length, 1);
  assert.match(html, /value="&lt;b&gt;"/);
  assert.match(html, /value="Frank Herbert"/);
});

import { syncBody, syncMessage, syncRequest } from "../static/editor-hardcover.js";

test("the Hardcover sync dialog leaves books out by default and puts them back on request", () => {
  assert.deepEqual(syncRequest(form({ mode: "out" }), "books", ids), { library: "books", ids, ignore: true });
  assert.deepEqual(syncRequest(form({ mode: "in" }), "books", ids), { library: "books", ids, ignore: false });
  assert.match(syncBody(1), /This book's reading is sent .* while it is in the sync/);
  assert.match(syncBody(3), /These 3 books' reading is sent .* while they are in the sync/);
});

test("the Hardcover sync message says what happened, including when nothing did", () => {
  assert.equal(syncMessage(1, true), "1 book left out of the Hardcover sync.");
  assert.equal(syncMessage(4, false), "4 books put back in the Hardcover sync.");
  assert.match(syncMessage(0, true), /already left out/);
  assert.match(syncMessage(0, false), /already in the sync/);
});

// The manage dialog reads the collections before it can offer them. A request
// that fails is not an empty library: it used to be reported as "There are no
// collections in this library yet", and a reply that was not JSON (a sign-in
// page, a proxy error) threw out of the click handler, so the button did nothing
// at all -- no dialog, no message.
const failingFetch = (body, status = 500) => async () => ({
  ok: false,
  status,
  json: async () => {
    if (typeof body === "string") throw new SyntaxError("Unexpected token '<'");
    return body;
  },
  text: async () => (typeof body === "string" ? body : JSON.stringify(body)),
});

const withFetch = async (stub, run) => {
  const original = globalThis.fetch;
  globalThis.fetch = stub;
  try { return await run(); } finally { globalThis.fetch = original; }
};

const manageRun = async (library = "books") => {
  const said = [];
  const dialogs = [];
  await manageCollections({
    state: { library },
    ask: (...args) => { dialogs.push(args); return Promise.resolve(false); },
    status: (text, kind) => said.push([text, kind]),
    reload: async () => {},
  });
  return { said, dialogs };
};

test("a collection request that fails says so instead of claiming the library has none", async () => {
  const { said, dialogs } = await withFetch(failingFetch({ error: "The catalogue could not be read." }), () => manageRun());
  assert.equal(dialogs.length, 0, "a failed load opens no dialog");
  assert.equal(said.length, 1, "one message");
  assert.doesNotMatch(said[0][0], /no collections in this library/, "a failure is not an empty library");
  assert.match(said[0][0], /could not be read/i);
  assert.equal(said[0][1], "problem");
});

test("a collection reply that is not JSON is reported, not thrown out of the click", async () => {
  const { said, dialogs } = await withFetch(failingFetch("<!doctype html><html>Sign in</html>", 401), () => manageRun());
  assert.equal(dialogs.length, 0);
  assert.equal(said.length, 1);
  assert.match(said[0][0], /could not read the collections/i);
  assert.equal(said[0][1], "problem");
});

test("a save that cannot reach the server is shown in the dialog instead of vanishing", async () => {
  const collections = [{ id: "p1", name: "Mine", count: 3 }];
  const said = [];
  let problem;
  await withFetch(async (url, options) => {
    if (!options?.method) return { ok: true, status: 200, json: async () => collections, text: async () => JSON.stringify(collections) };
    throw new TypeError("Failed to fetch");
  }, async () => {
    await manageCollections({
      state: { library: "books" },
      ask: (_title, _body, _label, onOk) => {
        return onOk(submitted({ "name:p1": "Reread", "delete:p1": null })).then((reply) => { problem = reply; return false; });
      },
      status: (text, kind) => said.push([text, kind]),
      reload: async () => {},
    });
  });
  assert.match(String(problem), /could not save/i);
  assert.equal(said.length, 0, "the dialog keeps the message; the status line is not used for it");
});
