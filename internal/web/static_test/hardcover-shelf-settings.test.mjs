import assert from "node:assert/strict";
import test from "node:test";
import { createHardcoverShelf } from "../static/hardcover-shelf-settings.js";

const skippedBook = { library: "light-novels", book: "b1", title: "86--EIGHTY-SIX - Volume 01", authors: ["Asato Asato"] };
const matchedBook = {
  library: "books", book: "a1", title: "Dune", authors: ["Frank Herbert"],
  hardcover_id: "5", hardcover_title: "Dune (Deluxe Edition)", hardcover_authors: ["Frank Herbert"], url: "https://hardcover.app/books/dune", how: "matched",
};

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
  const settle = () => new Promise((resolve) => setTimeout(resolve, 5));
  return { shelf, node, calls, click, settle };
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

test("each matched book shows what it is matched to on Hardcover, and how", async () => {
  const { shelf, node } = fixture({ matches: [matchedBook, { ...matchedBook, book: "a2", title: "Rebuild", hardcover_title: "Rebuild World: Volume 1 Part 2", how: "linked", url: "" }] });
  await shelf.load();
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.equal(node("[data-hardcover-shelf]").hidden, false);
  assert.match(html, /On Hardcover: <strong>Dune \(Deluxe Edition\)<\/strong> by Frank Herbert · matched by name/);
  assert.match(html, /Rebuild World: Volume 1 Part 2<\/strong> by Frank Herbert · linked by you/);
  assert.match(html, /href="https:\/\/hardcover\.app\/books\/dune"/);
  assert.match(html, />Change</);
  assert.match(html, /Nothing needs a link/);
});

test("skipped books wait under Needs a link, and names from the library are escaped", async () => {
  const { shelf, node } = fixture({ skipped: [{ library: "b", book: "1", title: "<img src=x onerror=alert(1)>", authors: ["A & B"] }] });
  await shelf.load();
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.doesNotMatch(html, /<img/);
  assert.match(html, /&lt;img/);
  assert.match(html, /A &amp; B/);
  assert.match(html, /Find on Hardcover/);
  assert.match(html, /No books have been sent to Hardcover yet/);
});

test("changing a wrong match searches by the book's title, and choosing links it and refreshes the lists", async () => {
  const { shelf, node, calls, click, settle } = fixture({
    matches: [matchedBook],
    search: [{ title: "Dune", authors: ["Frank Herbert"], year: 1965, provider_id: "379", source_url: "https://hardcover.app/books/dune-1965" }],
  });
  await shelf.load();
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

test("a doubtful match is explained and listed before the sure ones", async () => {
  const sure = { ...matchedBook, book: "s", title: "Sure" };
  const unsure = { ...matchedBook, book: "u", title: "Unsure", doubt: "Several Hardcover books have this title and author." };
  const { shelf, node } = fixture({ matches: [sure, unsure] });
  await shelf.load();
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.match(html, /Check this one: Several Hardcover books have this title and author\./);
  assert.ok(html.indexOf("Unsure") < html.indexOf("Sure<"), "the doubtful one comes first");
  assert.equal((html.match(/Check this one/g) || []).length, 1, "only the doubtful one is flagged");
});

test("stopping and resuming the sync of a book, and the books that are left alone", async () => {
  const { shelf, node, calls, click, settle } = fixture({ matches: [matchedBook], ignored: [{ library: "books", book: "a9", title: "Deleted on Hardcover", reason: "removed on Hardcover" }] });
  await shelf.load();
  const html = node("#hardcover-shelf-list").innerHTML;
  assert.match(html, />Stop syncing</);
  assert.match(html, /Not syncing/);
  assert.match(html, /Deleted on Hardcover<\/strong><small>removed on Hardcover/);
  assert.match(html, />Sync again</);
  click({ action: "stop", library: "books", book: "a1" });
  await settle();
  const stop = calls.find((c) => c.url.endsWith("/ignore"));
  assert.deepEqual(JSON.parse(stop.options.body), { library: "books", book: "a1", ignore: true });
  click({ action: "resume", library: "books", book: "a9" });
  await settle();
  const resume = calls.filter((c) => c.url.endsWith("/ignore")).pop();
  assert.deepEqual(JSON.parse(resume.options.body), { library: "books", book: "a9", ignore: false });
});
