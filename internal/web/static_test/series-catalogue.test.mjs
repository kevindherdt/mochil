import test from "node:test";
import assert from "node:assert/strict";
import {
  createSeriesCatalogue,
  seriesContext,
} from "../static/series-catalogue.js";

function deferred() {
  let resolve;
  const promise = new Promise((done) => (resolve = done));
  return { promise, resolve };
}

function windowedElement() {
  return { innerHTML: "", insertAdjacentHTML(_where, markup) { this.innerHTML += markup } };
}

function fixture(requestJSON, { createBoundedWindow, getSeriesLayout = () => "covers", releaseCovers = () => {}, libraryState = { selected: () => "books" } } = {}) {
  const seriesCoverGrid = windowedElement();
  const seriesList = windowedElement();
  const topSpacer = windowedElement();
  const bottomSpacer = windowedElement();
  const index = {
    hidden: true,
    innerHTML: "",
    querySelector(selector) {
      if (selector === ".series-cover-grid") return seriesCoverGrid;
      if (selector === ".series-list") return seriesList;
      if (selector === "[data-top-spacer]") return topSpacer;
      if (selector === "[data-bottom-spacer]") return bottomSpacer;
      return null;
    },
    querySelectorAll: () => [],
  };
  const elements = {
    "#series-index": index,
    "#book-catalog": {},
    "#status": {},
    "#search": { value: "Robot" },
    "#series-sentinel": {},
    "#series-load-more": {},
  };
  let bookAborts = 0;
  const defaultBoundedWindow = () => ({
    reset() {},
    begin() { return {}; },
    finish() { return { start: 0, end: 0 }; },
  });
  const catalogue = createSeriesCatalogue({
    $: (selector) => elements[selector],
    document: { createElement() { return { classList: { add() {} }, querySelector() { return null } } } },
    escapeHTML: String,
    requestJSON,
    createBoundedWindow: createBoundedWindow || defaultBoundedWindow,
    createContinuousPager: () => ({ refresh() {} }),
    cards: {
      markup() {},
      bookmarkIcon: "bookmark",
      installCoverFallbacks() {},
      releaseCovers,
    },
    collections: { installActions() {} },
    libraryState,
    getBrowseMode: () => "series",
    getSeriesLayout,
    getItemLayout: () => "covers",
    getActiveBucket: () => "R",
    abortBooks: () => bookAborts++,
    AbortController: class {
      constructor() {
        this.signal = {};
      }
      abort() {
        this.signal.aborted = true;
      }
    },
  });
  catalogue.bind();
  return { catalogue, elements, getBookAborts: () => bookAborts, seriesCoverGrid, seriesList, topSpacer, bottomSpacer };
}

const response = (data, total = data.length) => ({
  data,
  response: { headers: { get: () => String(total) } },
});

test("series context omits unknown authors and includes shelf in all-shelves mode", () => {
  const item = {
    authors: ["Unknown", "Ada Lovelace"],
    library_name: "Classics",
  };
  assert.equal(seriesContext(item, true), "Ada Lovelace · Classics");
  assert.equal(seriesContext(item, false), "Ada Lovelace");
});

test("series catalogue owns its filters and empty state", async () => {
  let requested;
  const { catalogue, elements, getBookAborts } = fixture(async (url) => {
    requested = url;
    return response([]);
  });
  assert.equal(await catalogue.load(), true);
  assert.match(requested, /library=books/);
  assert.match(requested, /bucket=R/);
  assert.match(requested, /q=Robot/);
  assert.match(elements["#series-index"].innerHTML, /No matches for “Robot”/);
  assert.equal(getBookAborts(), 1);
});

// Two callers run loadCurrentView per navigation (the route and the library
// loader), 8 ms apart. The second used to abort the first /api/series and fetch
// the same page again.
test("a second non-append load for the same view joins the request in flight", async () => {
  const requests = [];
  const { catalogue, elements } = fixture(() => {
    const request = deferred();
    requests.push(request);
    return request.promise;
  });
  const first = catalogue.load();
  const second = catalogue.load();
  assert.equal(requests.length, 1, "the duplicate load did not start a second /api/series");
  requests[0].resolve(response([], 2));
  assert.equal(await first, true);
  assert.equal(await second, true);
  assert.equal(elements["#status"].textContent, "Series under R");
});

test("a load for a different library replaces the one in flight", async () => {
  const requests = [];
  let library = "books";
  const { catalogue } = fixture(() => {
    const request = deferred();
    requests.push(request);
    return request.promise;
  }, { libraryState: { selected: () => library } });
  const first = catalogue.load();
  library = "manga";
  const second = catalogue.load();
  assert.equal(requests.length, 2, "a changed library still fetches");
  requests[1].resolve(response([], 2));
  assert.equal(await second, true);
  requests[0].resolve(response([], 99));
  assert.equal(await first, false, "the stale library cannot replace the new one");
});

test("the series grid is wrapped in a bounded window that releases trimmed covers", async () => {
  const boundedWindowCalls = [];
  const releasedNodes = [];
  let onRemove;
  const windowStub = {
    reset() {},
    begin(mode) { return { mode }; },
    finish() {
      onRemove(["trimmed-series-card"]);
      return { start: 1, end: 0 };
    },
  };
  const { catalogue, seriesCoverGrid, topSpacer, bottomSpacer } = fixture(
    async (url) => {
      if (url.startsWith("/api/series?")) return response([{ name: "Discworld", count: 3, library_id: "books", bucket: "D" }]);
      throw new Error(`Unexpected request: ${url}`);
    },
    {
      createBoundedWindow: (options) => {
        boundedWindowCalls.push(options);
        onRemove = options.onRemove;
        return windowStub;
      },
      releaseCovers: (node) => releasedNodes.push(node),
    },
  );
  assert.equal(await catalogue.load(), true);
  assert.equal(boundedWindowCalls.length, 1, "the series catalogue hands its grid to createBoundedWindow");
  assert.equal(boundedWindowCalls[0].container, seriesCoverGrid);
  assert.equal(boundedWindowCalls[0].topSpacer, topSpacer);
  assert.equal(boundedWindowCalls[0].bottomSpacer, bottomSpacer);
  assert.match(seriesCoverGrid.innerHTML, /Discworld/);
  assert.ok(releasedNodes.includes("trimmed-series-card"), "nodes trimmed by the window release their cover resources");
});
