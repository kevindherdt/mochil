import assert from "node:assert/strict";
import test from "node:test";
import { createHardcoverShelf, PAGE } from "../static/hardcover-shelf-settings.js";

// A shelf is thousands of books, so the panel asks for one page of one group at a time. These pin the parts
// that broke when that paging was introduced: the card must actually be shown, one failing request must not
// take it away, and a slow lookup must not answer under a different book.
const matched = (n) => ({
  library: "books", book: `b${n}`, title: `Book ${n}`, authors: ["An Author"],
  hardcover_id: String(n), hardcover_title: `Book ${n}`, hardcover_authors: ["An Author"], url: "", how: "matched",
});

function fixture({ enabled = true, failGroups = [], search = [], onSearch } = {}) {
  const nodes = {};
  const node = (selector) => (nodes[selector] ||= { textContent: "", innerHTML: "", value: "", hidden: false, open: true, listeners: {}, addEventListener(type, fn) { this.listeners[type] = fn; } });
  const calls = [];
  const document = { querySelector: (selector) => (selector === "[data-hardcover-shelf]" ? node(selector) : { insertAdjacentHTML() {} }) };
  // two books that need a link, so "find" has rows to act on
  const skipped = [
    { library: "books", book: "b1", title: "Book 1", authors: ["An Author"] },
    { library: "books", book: "b2", title: "Book 2", authors: ["An Author"] },
  ];
  const requestJSON = async (url) => {
    calls.push(url);
    const [path] = url.split("?");
    if (failGroups.some((group) => path.endsWith(group))) throw new Error("The server had a problem");
    if (path.endsWith("/search")) {
      if (onSearch) await onSearch();
      // each answer is labelled with the book that asked, so a stale one is visible
      const asked = new URLSearchParams(url.split("?")[1]).get("q");
      return { data: search.map((item) => ({ ...item, title: `result for ${asked}`, provider_id: `id-for-${asked}` })) };
    }
    if (path.endsWith("/skipped")) return { data: { enabled, items: skipped, total: skipped.length, offset: 0, limit: PAGE } };
    return { data: { enabled, items: [], total: 0, offset: 0, limit: PAGE } };
  };
  const shelf = createHardcoverShelf({ document, $: node, requestJSON });
  shelf.bind();
  return { shelf, node, calls };
}

test("the card is shown when the module is on — it is mounted hidden", async () => {
  const { shelf, node } = fixture();
  await shelf.load();
  assert.equal(node("[data-hardcover-shelf]").hidden, false, "a hidden card means nobody can reach the shelf");
});

test("the card stays hidden while the module is off", async () => {
  const { shelf, node } = fixture({ enabled: false });
  await shelf.load();
  assert.equal(node("[data-hardcover-shelf]").hidden, true);
});

test("one failing group does not take the card away or claim the shelf is empty", async () => {
  const { shelf, node } = fixture({ failGroups: ["/ignored"] });
  await shelf.load();
  assert.equal(node("[data-hardcover-shelf]").hidden, false, "the shelf is still there");
  assert.match(node("#hardcover-shelf-status").textContent, /could not be loaded/i);
});

test("a lookup answers for the book that asked, even when the reader moved on", async () => {
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const { shelf, node, calls } = fixture({
    search: [{ authors: [], source_url: "https://hardcover.app/books/one" }],
    onSearch: () => gate,
  });
  await shelf.load();

  // book one asks; book two asks before the first answer comes back
  node("#hardcover-shelf-list").listeners.click({ target: { closest: () => ({ dataset: { action: "find", library: "books", book: "b1" } }) } });
  await new Promise((resolve) => setTimeout(resolve, 5));
  node("#hardcover-shelf-list").listeners.click({ target: { closest: () => ({ dataset: { action: "find", library: "books", book: "b2" } }) } });
  await new Promise((resolve) => setTimeout(resolve, 5));
  release();
  await new Promise((resolve) => setTimeout(resolve, 20));

  const html = node("#hardcover-shelf-list").innerHTML;
  assert.doesNotMatch(html, /id-for-Book 1/, "the first book's result must not be offered under the second");
  assert.match(html, /id-for-Book 2/, "the second book's own result is shown");
  assert.ok(calls.some((url) => url.includes("/search")), "both lookups were attempted");
});

test("a row inserted while the reader pages does not appear twice", async () => {
  // The store is prepended to: between page one and "Show more", a book that was just recorded lands at the
  // front and every later row shifts down by one. Paging by offset without care shows the last row twice.
  const rows = Array.from({ length: 30 }, (_, i) => ({ library: "books", book: `b${i}`, title: `Book ${i}` }));
  const nodes = {};
  const node = (selector) => (nodes[selector] ||= { textContent: "", innerHTML: "", value: "", hidden: false, open: true, listeners: {}, addEventListener(type, fn) { this.listeners[type] = fn; } });
  const document = { querySelector: (selector) => (selector === "[data-hardcover-shelf]" ? node(selector) : { insertAdjacentHTML() {} }) };
  let prepended = false;
  const requestJSON = async (url) => {
    const [path, queryString = ""] = url.split("?");
    const params = new URLSearchParams(queryString);
    if (!path.endsWith("/matches")) return { data: { enabled: true, items: [], total: 0, offset: 0, limit: 25 } };
    // only the sure matches come from this fixture; "check these" is a different list
    if (params.get("doubt") === "1") return { data: { enabled: true, items: [], total: 0, offset: 0, limit: 25 } };
    const all = prepended ? [{ library: "books", book: "new", title: "Just recorded" }, ...rows] : rows;
    const offset = Number(params.get("offset") || 0);
    const limit = Number(params.get("limit") || 25);
    return { data: { enabled: true, items: all.slice(offset, offset + limit), total: all.length, offset, limit } };
  };
  const shelf = createHardcoverShelf({ document, $: node, requestJSON });
  shelf.bind();
  await shelf.load();
  await new Promise((resolve) => setTimeout(resolve, 10));
  prepended = true;
  node("#hardcover-shelf-list").listeners.click({ target: { closest: () => ({ dataset: { action: "more", group: "matched" } }) } });
  await new Promise((resolve) => setTimeout(resolve, 10));
  node("#hardcover-shelf-list").listeners.click({ target: { closest: () => ({ dataset: { action: "page", group: "matched" } }) } });
  await new Promise((resolve) => setTimeout(resolve, 20));
  const html = node("#hardcover-shelf-list").innerHTML;
  const keys = [...html.matchAll(/data-key="([^"]+)"/g)].map((match) => match[1]);
  assert.equal(new Set(keys).size, keys.length, `a row was listed twice: ${keys.join(", ")}`);
});
