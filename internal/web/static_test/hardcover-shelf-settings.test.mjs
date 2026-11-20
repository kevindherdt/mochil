import assert from "node:assert/strict";
import test from "node:test";
import { createHardcoverShelf, PAGE } from "../static/hardcover-shelf-settings.js";

const skippedBook = { library: "light-novels", book: "b1", title: "86--EIGHTY-SIX - Volume 01", authors: ["Asato Asato"] };
const matchedBook = {
  library: "books", book: "a1", title: "Dune", authors: ["Frank Herbert"],
  hardcover_id: "5", hardcover_title: "Dune (Deluxe Edition)", hardcover_authors: ["Frank Herbert"], url: "https://hardcover.app/books/dune", how: "matched",
};
const doubtful = (n) => ({ ...matchedBook, book: `d${n}`, title: `Doubt ${n}`, doubt: "Several Hardcover books have this title and author." });
const sure = (n) => ({ ...matchedBook, book: `s${n}`, title: `Sure ${n}`, url: "" });

// A server that pages: it holds the whole shelf, answers limit/offset/q, and reports an honest total. The panel
// must never receive a whole shelf, however many books it has.
function fixture({ enabled = true, skipped = [], sure: sureBooks = [], doubts = [], ignored = [], search = [], failLink = false } = {}) {
  const nodes = {};
  const node = (selector) => (nodes[selector] ||= { textContent: "", innerHTML: "", value: "", hidden: false, listeners: {}, addEventListener(type, fn) { this.listeners[type] = fn; } });
  const calls = [];
  const document = { querySelector: (selector) => (selector === "[data-hardcover-shelf]" ? node(selector) : { insertAdjacentHTML() {} }) };
  const page = (items, params) => {
    const q = (params.get("q") || "").toLowerCase();
    const found = items.filter((item) => !q || `${item.title} ${(item.authors || []).join(" ")} ${item.hardcover_title || ""}`.toLowerCase().includes(q));
    const offset = Number(params.get("offset") || 0);
    const limit = Number(params.get("limit") || PAGE);
    return { items: found.slice(offset, offset + limit), total: found.length, offset, limit };
  };
  const requestJSON = async (url, options = {}) => {
    calls.push({ url, options });
    const [path, queryString = ""] = url.split("?");
    const params = new URLSearchParams(queryString);
    if (path.endsWith("/skipped")) return { data: { enabled, ...page(skipped, params) } };
    if (path.endsWith("/matches")) {
      const doubt = params.get("doubt");
      const pool = doubt === "1" ? doubts : doubt === "0" ? sureBooks : [...doubts, ...sureBooks];
      return { data: { enabled, ...page(pool, params) } };
    }
    if (path.endsWith("/ignored")) return { data: page(ignored, params) };
    if (path.endsWith("/ignore")) return { data: { ignored: JSON.parse(options.body).ignore } };
    if (path.endsWith("/search")) return { data: search };
    if (path.endsWith("/link")) {
      if (failLink) throw new Error("The link could not be saved");
      return { data: { linked: true } };
    }
    throw new Error("unexpected " + url);
  };
  const shelf = createHardcoverShelf({ document, $: node, requestJSON });
  shelf.bind();
  const click = (dataset) => node("#hardcover-shelf-list").listeners.click({ target: { closest: () => ({ dataset }) } });
  const type = (value) => {
    node("#hardcover-shelf-filter").value = value;
    node("#hardcover-shelf-filter").listeners.input();
  };
  const wait = (ms = 10) => new Promise((resolve) => setTimeout(resolve, ms));
  const settle = () => wait(320); // the field debounces
  const html = () => node("#hardcover-shelf-list").innerHTML;
  const callsFor = (needle) => calls.filter((call) => call.url.includes(needle));
  return { shelf, node, calls, click, type, wait, settle, html, callsFor };
}

test("the screen stays hidden while the module is off, or for someone who may not see it", async () => {
  const off = fixture({ enabled: false });
  await off.shelf.load();
  assert.equal(off.node("[data-hardcover-shelf]").hidden, true);
  const nodes = {};
  const node = (selector) => (nodes[selector] ||= { textContent: "", innerHTML: "", hidden: false, addEventListener() {} });
  const denied = createHardcoverShelf({
    document: { querySelector: (selector) => (selector === "[data-hardcover-shelf]" ? node(selector) : { insertAdjacentHTML() {} }) },
    $: node,
    requestJSON: async () => { throw new Error("Administrator access is required"); },
  });
  await denied.load();
  assert.equal(node("[data-hardcover-shelf]").hidden, true, "a reader who is not an administrator never sees it");
});

// A shelf grows past a thousand books. The panel cannot be a list of everything it knows: it reports the whole
// set as one line and lists only what needs a decision.
test("the panel opens with a count, not a wall of rows", async () => {
  const { shelf, node, html } = fixture({ sure: Array.from({ length: 40 }, (_, i) => sure(i)), doubts: [doubtful(1)], skipped: [skippedBook] });
  await shelf.load();
  assert.equal(node("#hardcover-shelf-summary").textContent, "41 sent to Hardcover · 1 needs a link · 1 to check");
  assert.equal((html().match(/On Hardcover:/g) || []).length, 1, "only the doubtful match is a row");
  assert.match(html(), /40 books look right/);
  assert.ok(html().length < 4000, `the panel stays small with 42 books (${html().length} chars)`);
});

test("no request ever asks for more than a page of the shelf", async () => {
  const { shelf, calls } = fixture({ sure: Array.from({ length: 900 }, (_, i) => sure(i)) });
  await shelf.load();
  assert.ok(calls.length >= 4, "each group is asked for its first page");
  for (const call of calls) {
    const params = new URLSearchParams(call.url.split("?")[1] || "");
    assert.ok(Number(params.get("limit")) <= PAGE, `${call.url} asks for more than a page`);
  }
});

test("Show them opens the matched page, and the rest arrives a page at a time", async () => {
  const { shelf, click, html, callsFor, wait } = fixture({ sure: Array.from({ length: 60 }, (_, i) => sure(i)) });
  await shelf.load();
  click({ action: "more", group: "matched" });
  await wait();
  assert.equal((html().match(/On Hardcover:/g) || []).length, PAGE);
  assert.match(html(), new RegExp(`Show ${PAGE} more of 60`));
  click({ action: "page", group: "matched" });
  await wait();
  assert.equal((html().match(/On Hardcover:/g) || []).length, PAGE * 2);
  const offsets = callsFor("/matches?doubt=0").map((call) => new URLSearchParams(call.url.split("?")[1]).get("offset"));
  assert.deepEqual(offsets.slice(-2), ["0", String(PAGE)], "the next page is asked for by offset");
});

test("a sure match can be inspected, and says what it was matched to", async () => {
  const { shelf, click, html, wait } = fixture({ sure: [matchedBook] });
  await shelf.load();
  click({ action: "more", group: "matched" });
  await wait();
  assert.match(html(), /On Hardcover: <strong>Dune \(Deluxe Edition\)<\/strong> by Frank Herbert · matched by name/);
  assert.match(html(), /href="https:\/\/hardcover\.app\/books\/dune"/);
  assert.match(html(), />Change</);
  assert.match(html(), />Stop syncing</);
});

test("the search field is answered by the server, not by the handful of rows on screen", async () => {
  const { shelf, type, html, callsFor, settle } = fixture({ sure: Array.from({ length: 60 }, (_, i) => sure(i)) });
  await shelf.load();
  type("sure 57");
  await settle();
  const searched = callsFor("q=sure");
  assert.ok(searched.length >= 2, "every group is asked again with the filter");
  assert.equal(new URLSearchParams(searched[0].url.split("?")[1]).get("q"), "sure 57", "the field's words reach the server");
  assert.match(html(), /Sure 57/);
  assert.doesNotMatch(html(), /Sure 1</, "the rest of the shelf is filtered out by the server");
});

test("names from the library are escaped", async () => {
  const { shelf, html } = fixture({ skipped: [{ library: "b", book: "1", title: "<img src=x onerror=alert(1)>", authors: ["A & B"] }] });
  await shelf.load();
  assert.doesNotMatch(html(), /<img/);
  assert.match(html(), /&lt;img/);
  assert.match(html(), /A &amp; B/);
});

test("changing a wrong match searches by the book's title, and choosing links it and refreshes", async () => {
  const { shelf, node, calls, click, settle, wait } = fixture({
    doubts: [doubtful(1)],
    search: [{ title: "Dune", authors: ["Frank Herbert"], year: 1965, provider_id: "379", source_url: "https://hardcover.app/books/dune-1965" }],
  });
  await shelf.load();
  click({ action: "find", library: "books", book: "d1" });
  await wait();
  const lookup = calls.find((call) => call.url.includes("/search"));
  assert.ok(lookup, "searching starts right away");
  assert.equal(new URLSearchParams(lookup.url.split("?")[1]).get("q"), "Doubt 1");
  assert.match(node("#hardcover-shelf-list").innerHTML, /1965/);
  const before = calls.length;
  click({ action: "choose", id: "379" });
  await settle();
  const link = calls.find((call) => call.url.endsWith("/link"));
  assert.deepEqual(JSON.parse(link.options.body), { library: "books", book: "d1", provider_id: "379" });
  assert.match(node("#hardcover-shelf-status").textContent, /Linked/);
  assert.ok(calls.slice(before).some((call) => call.url.includes("/matches")), "the shelf is reloaded to show the new match");
});

test("a failed link keeps everything as it was and says why", async () => {
  const { shelf, node, click, settle, wait } = fixture({
    skipped: [skippedBook],
    search: [{ title: "x", authors: [], provider_id: "9", source_url: "https://hardcover.app/books/x" }],
    failLink: true,
  });
  await shelf.load();
  click({ action: "find", library: "light-novels", book: "b1" });
  await wait();
  click({ action: "choose", id: "9" });
  await settle();
  assert.match(node("#hardcover-shelf-status").textContent, /could not be saved/);
  assert.match(node("#hardcover-shelf-list").innerHTML, /86--EIGHTY-SIX - Volume 01/);
});

test("a doubtful match is explained and says how many are left to check", async () => {
  const { shelf, html } = fixture({ sure: [sure(1)], doubts: [doubtful(1)] });
  await shelf.load();
  assert.match(html(), /Check this one: Several Hardcover books have this title and author\./);
  assert.match(html(), /Check these <small>1<\/small>/);
  assert.equal((html().match(/Check this one/g) || []).length, 1, "only the doubtful one is flagged");
});

test("stopping and resuming the sync of a book, and the books that are left alone", async () => {
  const { shelf, node, calls, click, settle, wait } = fixture({
    doubts: [doubtful(1)],
    ignored: [{ library: "books", book: "a9", title: "Deleted on Hardcover", reason: "removed on Hardcover" }],
  });
  await shelf.load();
  assert.match(node("#hardcover-shelf-list").innerHTML, />Stop syncing</);
  assert.match(node("#hardcover-shelf-list").innerHTML, /1 left alone/);
  click({ action: "stop", library: "books", book: "d1" });
  await settle();
  const stop = calls.find((call) => call.url.endsWith("/ignore"));
  assert.deepEqual(JSON.parse(stop.options.body), { library: "books", book: "d1", ignore: true });

  click({ action: "more", group: "ignored" });
  await wait();
  assert.match(node("#hardcover-shelf-list").innerHTML, /Deleted on Hardcover<\/strong><small>removed on Hardcover/);
  click({ action: "resume", library: "books", book: "a9" });
  await settle();
  const resume = calls.filter((call) => call.url.endsWith("/ignore")).pop();
  assert.deepEqual(JSON.parse(resume.options.body), { library: "books", book: "a9", ignore: false });
});
