import assert from "node:assert/strict";
import test from "node:test";
import { TOOLS, mountTools } from "../static/editor-tools.js";
import { COLUMNS, countText } from "../static/editor-model.js";

test("the Tools menu lists every tool and runs the one clicked, then closes", async () => {
  let handler;
  const list = { innerHTML: "", addEventListener: (type, fn) => (handler = fn) };
  const details = { open: true };
  const ran = [];
  TOOLS.push({ id: "probe", label: "Probe", hint: "test", run: (context) => ran.push(context.tag) });
  try {
    await mountTools({ list, details, context: () => ({ tag: "ctx" }), hermesEnabled: true });
    assert.match(list.innerHTML, /data-tool="count"/);
    assert.match(list.innerHTML, /data-tool="probe"/);
    handler({ target: { closest: () => ({ dataset: { tool: "probe" } }) } });
    assert.deepEqual(ran, ["ctx"]);
    assert.equal(details.open, false);
  } finally {
    TOOLS.pop();
  }
});

test("the two Ask Hermes tools are left out unless hermesEnabled resolves true", async () => {
  const list = { innerHTML: "", addEventListener: () => {} };
  const details = { open: false };
  await mountTools({ list, details, context: () => ({}) }); // hermesEnabled omitted: fails closed
  assert.doesNotMatch(list.innerHTML, /data-tool="isbn-ask"/);
  assert.doesNotMatch(list.innerHTML, /data-tool="match-ask"/);
  assert.match(list.innerHTML, /data-tool="isbn"/, "the non-AI tool stays");

  await mountTools({ list, details, context: () => ({}), hermesEnabled: false });
  assert.doesNotMatch(list.innerHTML, /data-tool="isbn-ask"/);

  await mountTools({ list, details, context: () => ({}), hermesEnabled: Promise.resolve(true) });
  assert.match(list.innerHTML, /data-tool="isbn-ask"/);
  assert.match(list.innerHTML, /data-tool="match-ask"/);
});

test("a click can only reach a tool that was actually rendered", async () => {
  let handler;
  const list = { innerHTML: "", addEventListener: (type, fn) => (handler = fn) };
  const ran = [];
  const context = () => ({ noted: true });
  const tool = TOOLS.find((entry) => entry.id === "isbn-ask");
  const original = tool.run;
  try {
    await mountTools({ list, details: { open: false }, context, hermesEnabled: false });
    tool.run = () => ran.push("should not run");
    handler({ target: { closest: () => ({ dataset: { tool: "isbn-ask" } }) } });
  } finally {
    tool.run = original;
  }
  assert.deepEqual(ran, [], "the hidden tool's button does not exist, so nothing it maps to can run");
});

test("Words and Pages are read-only, sortable, numeric columns that start hidden", () => {
  for (const id of ["words", "pages"]) {
    const column = COLUMNS.find((c) => c.id === id);
    assert.ok(column, `${id} column exists`);
    assert.equal(column.field, undefined, "not editable");
    assert.equal(column.sort, id);
    assert.equal(column.num, true);
    assert.notEqual(column.on, true, "off until chosen");
  }
  assert.equal(countText(0), "");
  assert.equal(countText(80250), "80,250");
});

test("Check quality starts the check on the selected books, watches it, then reloads and reports", async () => {
  const check = TOOLS.find((tool) => tool.id === "check");
  assert.ok(check);
  const calls = [], said = [];
  const replies = [{ status: "running", current: 1, total: 3 }, { status: "ready", message: "Checked 3 books; no missing covers or unreadable files" }];
  const realTimeout = globalThis.setTimeout;
  globalThis.setTimeout = (fn) => realTimeout(fn, 0);
  try {
    await check.run({
      state: { library: "books", selected: new Set(["a", "b"]) },
      api: async (path, options) => { calls.push([path, options?.method, options?.body]); return options?.method === "POST" ? {} : replies.shift(); },
      status: (text) => said.push(text),
      reload: async () => said.push("reloaded"),
    });
  } finally { globalThis.setTimeout = realTimeout; }
  assert.deepEqual(calls[0], ["/api/metadata/editor/check", "POST", { library: "books", ids: ["a", "b"] }]);
  assert.match(said[0], /Checking 2 selected books/);
  assert.match(said.at(-2), /reloaded/);
  assert.match(said.at(-1), /^Checked 3 books/);
});

function jobContext(replies, calls, said, stop) {
  return {
    state: { library: "books", selected: new Set() },
    api: async (path, options) => { calls.push([options?.method || "GET", path]); return options?.method ? {} : replies.length > 1 ? replies.shift() : replies[0]; },
    status: (text) => said.push(text),
    reload: async () => said.push("reloaded"),
    stop,
  };
}

test("Stop is offered while a tool runs, asks the server to stop, and the page reports it was stopped", async () => {
  const calls = [], said = [], shown = [];
  let press;
  const stop = { show: (fn) => { shown.push("show"); press = fn; }, hide: () => shown.push("hide") };
  const realTimeout = globalThis.setTimeout;
  globalThis.setTimeout = (fn) => realTimeout(fn, 0);
  const replies = [{ status: "running", current: 10, total: 100 }];
  try {
    const context = jobContext(replies, calls, said, stop);
    const ran = TOOLS.find((tool) => tool.id === "check").run(context);
    await new Promise((resolve) => realTimeout(resolve, 5));
    assert.deepEqual(shown, ["show"], "Stop appears");
    await press();
    replies.push({ status: "canceled", current: 12, total: 100 });
    await ran;
  } finally { globalThis.setTimeout = realTimeout; }
  assert.ok(calls.some(([method, path]) => method === "DELETE" && path === "/api/metadata/editor/check?library=books"));
  assert.deepEqual(shown, ["show", "hide"], "Stop goes away afterwards");
  assert.match(said.at(-1), /^Stopped after 12 of 100/);
});

test("a page that opens while a job is running picks up watching it, and does not start another", async () => {
  const { resumeJobs } = await import("../static/editor-tools.js");
  const calls = [], said = [], shown = [];
  const realTimeout = globalThis.setTimeout;
  globalThis.setTimeout = (fn) => realTimeout(fn, 0);
  const replies = [{ status: "idle" }, { status: "running", current: 5, total: 9 }, { status: "ready", message: "Checked 9 books; no missing covers or unreadable files" }];
  try {
    await resumeJobs(jobContext(replies, calls, said, { show: () => shown.push("show"), hide: () => shown.push("hide") }));
    await new Promise((resolve) => realTimeout(resolve, 30));
  } finally { globalThis.setTimeout = realTimeout; }
  assert.equal(calls.filter(([method]) => method === "POST").length, 0, "nothing is started");
  assert.deepEqual(shown, ["show", "hide"]);
  assert.match(said.at(-1), /^Checked 9 books/);
});

test("Find ISBNs is a tool that starts the ISBN job and can be stopped like the others", async () => {
  const tool = TOOLS.find((entry) => entry.id === "isbn");
  assert.ok(tool);
  const calls = [], shown = [];
  const realTimeout = globalThis.setTimeout;
  globalThis.setTimeout = (fn) => realTimeout(fn, 0);
  try {
    await tool.run({
      state: { library: "books", selected: new Set() },
      api: async (path, options) => { calls.push([options?.method || "GET", path]); return options?.method ? {} : { status: "ready", message: "Searched 4 books: 3 with a clear ISBN, 1 unsure, 0 with none" }; },
      status: () => {}, reload: async () => {}, stop: { show: () => shown.push("show"), hide: () => shown.push("hide") },
    });
  } finally { globalThis.setTimeout = realTimeout; }
  assert.deepEqual(calls[0], ["POST", "/api/metadata/editor/isbn"]);
  assert.deepEqual(shown, ["show", "hide"]);
});

test("Find Hardcover matches is a stoppable tool like the others", async () => {
  const tool = TOOLS.find((entry) => entry.id === "matches");
  assert.ok(tool);
  const calls = [];
  const realTimeout = globalThis.setTimeout;
  globalThis.setTimeout = (fn) => realTimeout(fn, 0);
  try {
    await tool.run({ state: { library: "books", selected: new Set(["a"]) }, api: async (path, options) => { calls.push([options?.method || "GET", path, options?.body]); return options?.method ? {} : { status: "ready", message: "Looked up 1 books" }; }, status: () => {}, reload: async () => {}, stop: { show: () => {}, hide: () => {} } });
  } finally { globalThis.setTimeout = realTimeout; }
  assert.deepEqual(calls[0], ["POST", "/api/metadata/editor/matches", { library: "books", ids: ["a"] }]);
});

test("Asking Hermes shows which book it is on while the slow answer is awaited", async () => {
  const said = [];
  const realTimeout = globalThis.setTimeout;
  globalThis.setTimeout = (fn) => realTimeout(fn, 0);
  const replies = [{ status: "running", current: 4, total: 60, message: "Asking Hermes about “Dune” (question 2)" }, { status: "ready", message: "Asked." }];
  try {
    await TOOLS.find((tool) => tool.id === "isbn-ask").run({
      state: { library: "books", selected: new Set() },
      api: async (path, options) => (options?.method ? {} : replies.shift()),
      status: (text) => said.push(text),
      reload: async () => {},
    });
  } finally { globalThis.setTimeout = realTimeout; }
  assert.ok(said.some((text) => /4 of 60 · Asking Hermes about “Dune” \(question 2\)/.test(text)), said.join("|"));
});

test("A tool the server refuses to start says why, marked as a problem", async () => {
  const said = [];
  await TOOLS.find((tool) => tool.id === "isbn-ask").run({
    state: { library: "books", selected: new Set() },
    api: async () => { throw new Error("Asking needs a helper model and the ISBN ability switched on in Settings, Hermes."); },
    status: (text, kind) => said.push([text, kind]),
    reload: async () => {},
  });
  assert.equal(said.at(-1)[1], "problem");
  assert.match(said.at(-1)[0], /needs a helper model/);
});
