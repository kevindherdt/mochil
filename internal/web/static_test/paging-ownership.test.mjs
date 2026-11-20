import assert from "node:assert/strict";
import test from "node:test";
import { createBookCatalogue } from "../static/book-catalogue.js";
import { createContinuousPager } from "../static/continuous-pager.js";
import { createRequestCache } from "../static/request-cache.js";

class Observer {
  static latest;
  constructor(callback) {
    this.callback = callback;
    Observer.latest = this;
  }
  observe(target) {
    this.target = target;
  }
  disconnect() {}
  intersect() {
    this.callback([{ isIntersecting: true, target: this.target }]);
  }
}

// A page that failed used to be requested again by the next animation frame, the
// next intersection and the 500 ms interval, so an outage turned into a request
// storm instead of one visible Retry.
test("a failed page waits for the reader instead of retrying itself", async () => {
  globalThis.window = { IntersectionObserver: Observer, addEventListener() {}, removeEventListener() {} };
  globalThis.IntersectionObserver = Observer;
  const button = {};
  let calls = 0;
  const sentinel = { getBoundingClientRect: () => ({ top: 10, bottom: 20 }) };
  const pager = createContinuousPager({
    sentinel,
    button,
    canLoad: () => true,
    load: async () => {
      calls += 1;
      return false;
    },
  });
  Observer.latest.intersect();
  await new Promise((resolve) => setTimeout(resolve, 20));
  assert.equal(calls, 1);
  assert.equal(button.hidden, false, "the visible Retry must come back");

  Observer.latest.intersect();
  await new Promise((resolve) => setTimeout(resolve, 650));
  assert.equal(calls, 1, "nothing may retry on its own after a failure");

  await button.onclick();
  assert.equal(calls, 2, "the reader's Retry still works");
  pager.disconnect();
});

// Every earlier-page request used to keep running with no owner, so a response
// from the previous library could prepend its books into the new catalogue and
// move its offset.
test("a late earlier page cannot prepend into a new catalogue", async () => {
  const pages = [];
  const deferred = () => {
    let resolve;
    const promise = new Promise((done) => (resolve = done));
    return { promise, resolve };
  };
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
    "#search": { value: "" },
    "#book-catalog": { hidden: false, scrollIntoView() {} },
  };
  const pagers = [];
  const catalogue = createBookCatalogue({
    $: (selector) => elements[selector],
    requestJSON: (url) => {
      if (url.startsWith("/api/book-navigation")) return Promise.resolve({ data: [] });
      const page = deferred();
      pages.push(page);
      return page.promise;
    },
    createBoundedWindow: () => ({ reset() {}, begin: () => ({}), finish: () => ({ start: 0, end: 0 }) }),
    createContinuousPager: (options) => {
      pagers.push(options);
      return { refresh() {} };
    },
    cards: { markup: (book) => `<article>${book.title}</article>`, installCoverFallbacks() {}, releaseCovers() {} },
    collections: { installActions() {} },
    libraryState: { selected: () => "books" },
    getItemLayout: () => "covers",
    abortSeries() {},
    onNavigationChange() {},
    setCurrentBucket() {},
    isActive: () => true,
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
  const response = (data, total = data.length) => ({
    data,
    response: { headers: { get: () => String(total) } },
  });

  const first = catalogue.load(true, 40);
  pages[0].resolve(response([{ title: "Forty" }], 500));
  await first;

  const previous = pagers[1].load();
  const replacement = catalogue.load(true, 0);
  pages[2].resolve(response([{ title: "New shelf" }], 1));
  await replacement;
  pages[1].resolve(response([{ title: "Old shelf page" }], 500));
  assert.equal(await previous, false);
  assert.doesNotMatch(elements["#books"].innerHTML, /Old shelf page/);
});

// Two rails asking for the same list shared one promise owned by the first
// caller's AbortSignal, so one rail's abort cancelled the other's load.
test("sharing a request never couples two callers' cancellation", async () => {
  const seen = [];
  const cache = createRequestCache((url, options) => {
    seen.push(options?.signal ?? null);
    return Promise.resolve({ data: [] });
  });
  const first = { name: "first" };
  const second = { name: "second" };
  cache.json("/api/reading", { signal: first });
  cache.json("/api/reading", { signal: second });
  assert.equal(seen.length, 2);
  assert.equal(seen[0], first);
  assert.equal(seen[1], second);
});

test("a plain GET is still shared between rails", async () => {
  let calls = 0;
  const cache = createRequestCache(() => {
    calls += 1;
    return Promise.resolve({ data: [] });
  });
  cache.json("/api/reading");
  cache.json("/api/reading");
  assert.equal(calls, 1);
});
