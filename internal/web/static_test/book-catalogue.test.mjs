import test from "node:test";
import assert from "node:assert/strict";
import { createBookCatalogue } from "../static/book-catalogue.js";

function deferred() {
  let resolve;
  const promise = new Promise((done) => (resolve = done));
  return { promise, resolve };
}

function fixture(requestJSON, { isActive = () => true } = {}) {
  const elements = {
    "#books": {
      children: [],
      innerHTML: "",
      classList: { toggle() {} },
      insertAdjacentHTML(_where, markup) {
        this.innerHTML += markup;
      },
    },
    "#status": {},
    "#book-top-spacer": {},
    "#book-bottom-spacer": {},
    "#book-sentinel": {},
    "#book-start-sentinel": {},
    "#book-load-previous": {},
    "#load-more": {},
    "#search": { value: "Ada" },
    "#book-catalog": { hidden: false, scrollIntoView() {} },
  };
  const navigationChanges = [];
  const pagers = [];
  const catalogue = createBookCatalogue({
    $: (selector) => elements[selector],
    requestJSON,
    createBoundedWindow: () => ({
      reset() {},
      begin() {
        return {};
      },
      finish() {
        return { start: 0, end: 0 };
      },
    }),
    createContinuousPager: (options) => {
      pagers.push(options);
      return { refresh() {} };
    },
    cards: {
      markup: (book) => `<article>${book.title}</article>`,
      installCoverFallbacks() {},
      releaseCovers() {},
    },
    collections: { installActions() {} },
    libraryState: { selected: () => "books" },
    getItemLayout: () => "covers",
    abortSeries() {},
    onNavigationChange: () => navigationChanges.push(true),
    setCurrentBucket() {},
    isActive,
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
  return { catalogue, elements, navigationChanges, pagers };
}

const response = (data, total = data.length) => ({
  data,
  response: { headers: { get: () => String(total) } },
});

test("book catalogue owns filtering, rendering, totals, and navigation", async () => {
  const urls = [];
  const { catalogue, elements } = fixture(async (url) => {
    urls.push(url);
    if (url.startsWith("/api/book-navigation"))
      return { data: [{ bucket: "A", offset: 0 }] };
    return response([{ title: "Analytical Engine" }]);
  });
  assert.equal(await catalogue.load(true), true);
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.match(
    urls.find((url) => url.startsWith("/api/books")),
    /library=books/,
  );
  assert.match(
    urls.find((url) => url.startsWith("/api/books")),
    /q=Ada/,
  );
  assert.match(elements["#books"].innerHTML, /Analytical Engine/);
  assert.equal(elements["#status"].textContent, "");
  assert.deepEqual(catalogue.navigation(), [{ bucket: "A", offset: 0 }]);
});

test("a superseded request cannot replace the newest catalogue", async () => {
  const pages = [];
  const { catalogue, elements } = fixture((url) => {
    if (url.startsWith("/api/book-navigation"))
      return Promise.resolve({ data: [] });
    const page = deferred();
    pages.push(page);
    return page.promise;
  });
  const first = catalogue.load(true);
  const second = catalogue.load(true);
  pages[1].resolve(response([{ title: "Newest" }]));
  assert.equal(await second, true);
  pages[0].resolve(response([{ title: "Stale" }]));
  assert.equal(await first, false);
  assert.match(elements["#books"].innerHTML, /Newest/);
  assert.doesNotMatch(elements["#books"].innerHTML, /Stale/);
});

test("book paging stays dormant while another destination owns the shared surface", () => {
  let active = false;
  const { pagers } = fixture(async () => response([]), {
    isActive: () => active,
  });
  assert.equal(pagers[0].canLoad(), false);
  assert.equal(pagers[1].canLoad(), false);
  active = true;
  assert.equal(pagers[0].canLoad(), false, "an empty active catalogue still has no next page");
});
