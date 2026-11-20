import assert from "node:assert/strict";
import test from "node:test";
import { createHardcoverShelf } from "../static/hardcover-shelf-settings.js";

const skippedBook = { library: "light-novels", book: "b1", title: "86--EIGHTY-SIX - Volume 01", authors: ["Asato Asato"] };
const matchedBook = {
  library: "books", book: "a1", title: "Dune", authors: ["Frank Herbert"],
  hardcover_id: "5", hardcover_title: "Dune (Deluxe Edition)", hardcover_authors: ["Frank Herbert"], url: "https://hardcover.app/books/dune", how: "matched",
};
const doubtful = (n) => ({ ...matchedBook, book: `d${n}`, title: `Doubt ${n}`, doubt: "Several Hardcover books have this title and author." });
const sure = (n) => ({ ...matchedBook, book: `s${n}`, title: `Sure ${n}`, url: "" });

function fixture({ enabled = true, skipped = [], matches = [], search = [], failLink = false, ignored = [] } = {}) {
  const nodes = {};
  const node = (selector) => (nodes[selector] ||= { textContent: "", innerHTML: "", value: "", hidden: false, listeners: {}, addEventListener(type, fn) { this.listeners[type] = fn; } });
  const calls = [];
  const document = { querySelector: (selector) => (selector === "[data-hardcover-shelf]" ? node(selector) : { insertAdjacentHTML() {} }) };
  const requestJSON = async (url, options = {}) => {
    calls.push({ url, options });
    if (url.endsWith("/skipped")) return { data: { enabled, items: skipped } };
    if (url.endsWith("/matches")) return { data: { enabled, items: matches } };
    if (url.endsWith("/ignored")) return { data: { items: ignored } };
    if (url.endsWith("/ignore")) return { data: { ignored: JSON.parse(options.body).ignore } };
    if (url.includes("/search")) return { data: search };
    if (url.endsWith("/link")) {
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
  const settle = () => new Promise((resolve) => setTimeout(resolve, 5));
  return { shelf, node, calls, click, type, settle };
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

// A library grows past a few hundred books. The panel therefore cannot be a list of everything it knows:
// it reports the whole set as one line and lists only what needs a decision.
test("the panel opens with a count, not a wall of rows", async () => {
  const matches = [...Array.from({ length: 40 }, (_, i) => sure(i)), doubtful(1)];
  const { shelf, node } = fixture({ matches, skipped: [skippedBook] });
  await shelf.load();
  assert.equal(node("#hardcover-shelf-summary").textContent, "41 sent to Hardcover · 1 needs a link · 1 to check");
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.equal((html.match(/On Hardcover:/g) || []).length, 1, "only the doubtful match is a row");
  assert.match(html, /40 more books look right/);
  assert.doesNotMatch(html, /Sure 7</, "a match that looks right is not rendered as a row");
  assert.ok(html.length < 3000, `the panel stays small with 42 books (${html.length} chars)`);
});

test("a sure match can still be inspected, and shows what it was matched to", async () => {
  const { shelf, node, click } = fixture({ matches: [matchedBook] });
  await shelf.load();
  click({ action: "more", group: "Matched" });
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.match(html, /On Hardcover: <strong>Dune \(Deluxe Edition\)<\/strong> by Frank Herbert · matched by name/);
  assert.match(html, /href="https:\/\/hardcover\.app\/books\/dune"/);
  assert.match(html, />Change</);
  assert.match(html, />Stop syncing</);
});

test("a long Needs a link list is capped until the reader asks for all of it", async () => {
  const skipped = Array.from({ length: 12 }, (_, i) => ({ library: "books", book: `b${i}`, title: `Unmatched ${i}`, authors: ["A"] }));
  const { shelf, node, click } = fixture({ skipped });
  await shelf.load();
  const first = node("#hardcover-shelf-list").innerHTML;
  assert.equal((first.match(/Find on Hardcover/g) || []).length, 8, "eight rows, not twelve");
  assert.match(first, /Show all 12/);
  click({ action: "more", group: "Needs a link" });
  assert.equal((node("#hardcover-shelf-list").innerHTML.match(/Find on Hardcover/g) || []).length, 12);
});

test("one field finds a book among all the matches", async () => {
  const { shelf, node, type } = fixture({ matches: [sure(1), sure(2), sure(3)] });
  await shelf.load();
  type("sure 3");
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.match(html, /Sure 3</);
  assert.doesNotMatch(html, /Sure 1</, "the others are filtered out");
  assert.equal(node("#hardcover-shelf-summary").textContent, "3 sent to Hardcover · 0 need a link", "the count still describes the whole set");
  type("");
  assert.match(node("#hardcover-shelf-list").innerHTML, /3 more books look right/);
});

test("skipped books wait under Needs a link, and names from the library are escaped", async () => {
  const { shelf, node } = fixture({ skipped: [{ library: "b", book: "1", title: "<img src=x onerror=alert(1)>", authors: ["A & B"] }] });
  await shelf.load();
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.doesNotMatch(html, /<img/);
  assert.match(html, /&lt;img/);
  assert.match(html, /A &amp; B/);
  assert.match(html, /Find on Hardcover/);
});

test("changing a wrong match searches by the book's title, and choosing links it and refreshes the lists", async () => {
  const { shelf, node, calls, click, settle } = fixture({
    matches: [matchedBook],
    search: [{ title: "Dune", authors: ["Frank Herbert"], year: 1965, provider_id: "379", source_url: "https://hardcover.app/books/dune-1965" }],
  });
  await shelf.load();
  click({ action: "more", group: "Matched" });
  node("#hardcover-search-input").value = "Dune"; // what the box shows once drawn: the book's title
  click({ action: "find", library: "books", book: "a1" });
  await settle();
  const search = calls.find((c) => c.url.includes("/search"));
  assert.ok(search, "searching starts right away");
  assert.match(decodeURIComponent(search.url), /q=Dune$/);
  assert.match(node("#hardcover-shelf-list").innerHTML, /1965/);
  const before = calls.length;
  click({ action: "choose", id: "379" });
  await settle();
  const link = calls.find((c) => c.url.endsWith("/link"));
  assert.deepEqual(JSON.parse(link.options.body), { library: "books", book: "a1", provider_id: "379" });
  assert.match(node("#hardcover-shelf-status").textContent, /Linked/);
  assert.ok(calls.slice(before).some((c) => c.url.endsWith("/matches")), "the lists are reloaded to show the new match");
});

test("a failed link keeps everything as it was and says why", async () => {
  const { shelf, node, click, settle } = fixture({
    skipped: [skippedBook],
    search: [{ title: "x", authors: [], provider_id: "9", source_url: "https://hardcover.app/books/x" }],
    failLink: true,
  });
  await shelf.load();
  node("#hardcover-search-input").value = "86";
  click({ action: "find", library: "light-novels", book: "b1" });
  await settle();
  click({ action: "choose", id: "9" });
  await settle();
  assert.match(node("#hardcover-shelf-status").textContent, /could not be saved/);
  assert.match(node("#hardcover-shelf-list").innerHTML, /86--EIGHTY-SIX - Volume 01/);
});

test("a doubtful match is explained, and sits above the count of the sure ones", async () => {
  const { shelf, node } = fixture({ matches: [sure(1), doubtful(1)] });
  await shelf.load();
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.match(html, /Check this one: Several Hardcover books have this title and author\./);
  assert.ok(html.indexOf("Doubt 1") < html.indexOf("more book looks right"), "the doubtful one comes first");
  assert.equal((html.match(/Check this one/g) || []).length, 1, "only the doubtful one is flagged");
});

test("stopping and resuming the sync of a book, and the books that are left alone", async () => {
  const { shelf, node, calls, click, settle } = fixture({
    matches: [doubtful(1)],
    ignored: [{ library: "books", book: "a9", title: "Deleted on Hardcover", reason: "removed on Hardcover" }],
  });
  await shelf.load();
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.match(html, />Stop syncing</);
  assert.match(html, /1 left alone/);
  click({ action: "stop", library: "books", book: "d1" });
  await settle();
  const stop = calls.find((c) => c.url.endsWith("/ignore"));
  assert.deepEqual(JSON.parse(stop.options.body), { library: "books", book: "d1", ignore: true });

  click({ action: "more", group: "Not syncing" });
  assert.match(node("#hardcover-shelf-list").innerHTML, /Deleted on Hardcover<\/strong><small>removed on Hardcover/);
  click({ action: "resume", library: "books", book: "a9" });
  await settle();
  const resume = calls.filter((c) => c.url.endsWith("/ignore")).pop();
  assert.deepEqual(JSON.parse(resume.options.body), { library: "books", book: "a9", ignore: false });
});
