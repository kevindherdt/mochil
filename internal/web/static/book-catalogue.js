import { emptySpec, errorSpec, renderStatePanel } from "./state-panel.js?v=2";
import { createBucketTracker } from "./catalog-scroll-position.js?v=2";

const stateHandlers = (retry) => ({
  retry,
  settings: () => { location.hash = "#/settings"; },
  "clear-search": () => {
    const input = document.querySelector("#search");
    if (!input) return;
    input.value = "";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.focus();
  },
});

const defaultPageSize = 120;

export function createBookCatalogue({
  $,
  requestJSON,
  createBoundedWindow,
  createContinuousPager,
  cards,
  collections,
  libraryState,
  getItemLayout,
  abortSeries,
  onNavigationChange,
  setCurrentBucket,
  isActive = () => true,
  AbortController: AbortControllerClass = AbortController,
  pageSize = defaultPageSize,
  getSort = () => '',
  getActiveBucket = () => '',
}) {
  let books = [];
  let total = 0;
  let startOffset = 0;
  let navigation = [];
  let request;
  let previousRequest;
  // Bumped whenever the catalogue is replaced, so a late earlier-page response
  // cannot prepend the previous library's books into the new shelf.
  let generation = 0;
  let navigationRequest;
  let continuation;
  let prepending;
  let nextPager;
  let previousPager;

  const container = $("#books");
  const status = $("#status");
  const bookWindow = createBoundedWindow({
    container,
    topSpacer: $("#book-top-spacer"),
    bottomSpacer: $("#book-bottom-spacer"),
    onRemove: (nodes) => nodes.forEach((node) => cards.releaseCovers(node)),
  });
  const bucketTracker = createBucketTracker();

  function queryFor(offset, limit = pageSize, { bucket = true } = {}) {
    const query = new URLSearchParams({ limit, offset });
    if (!offset) query.delete("offset");
    if (libraryState.selected()) query.set("library", libraryState.selected());
    const search = $("#search").value.trim();
    if (search) query.set("q", search);
    if (getSort()) query.set("sort", getSort());
    // A letter picked while a non-title sort is on filters the shelf: the
    // server's letter offsets only describe its title order.
    if (bucket && getActiveBucket()) query.set("bucket", getActiveBucket());
    return query;
  }

  function render(newBooks, mode = "replace") {
    const layout = getItemLayout();
    const markup = newBooks.map((book) => cards.markup(book, layout)).join("");
    container.classList.toggle("book-list", layout === "list");
    if (mode === "replace") {
      bookWindow.reset();
      cards.releaseCovers(container);
      container.innerHTML = markup;
    } else {
      const insertion = bookWindow.begin(mode);
      container.insertAdjacentHTML(
        mode === "append" ? "beforeend" : "afterbegin",
        markup,
      );
      const removed = bookWindow.finish(insertion);
      if (removed.start) {
        books.splice(0, removed.start);
        startOffset += removed.start;
      }
      if (removed.end) books.splice(-removed.end, removed.end);
      // A trimmed or prepended window shifts every child's index.
      if (mode !== "append" || removed.start || removed.end) bucketTracker.reset();
    }
    cards.installCoverFallbacks(container);
    collections.installActions(container);
    if (!books.length) {
      renderStatePanel(container, emptySpec({ noun: "books", query: $("#search")?.value || "" }), stateHandlers(() => load(true)));
      $("#load-more").hidden = true;
      status.textContent = "";
      collections.syncSaveSearch?.(false);
      return;
    }
    const end = startOffset + books.length;
    status.textContent = "";
    collections.syncSaveSearch?.(true);
    $("#load-more").hidden = end >= total;
    $("#load-more").disabled = false;
    $("#load-more").textContent = "Load more books";
    nextPager?.refresh();
    previousPager?.refresh();
  }

  async function loadNavigation() {
    navigationRequest?.abort();
    const currentRequest = new AbortControllerClass();
    navigationRequest = currentRequest;
    // /api/book-navigation reports offsets in its own (title) order and takes
    // no sort, so the rail's letters come from the unfiltered shelf. When a
    // non-title sort is on, the rail filters by letter instead of jumping to
    // an offset that would name the wrong book.
    const query = new URLSearchParams();
    if (libraryState.selected()) query.set("library", libraryState.selected());
    const search = $("#search").value.trim();
    if (search) query.set("q", search);
    try {
      const result = await requestJSON(`/api/book-navigation?${query}`, {
        signal: currentRequest.signal,
      });
      if (navigationRequest !== currentRequest) return;
      navigation = result.data;
    } catch (error) {
      if (error.name !== "AbortError" && navigationRequest === currentRequest)
        navigation = [];
    }
    if (navigationRequest === currentRequest) onNavigationChange();
  }


  // The page names itself and counts itself, the way the design does: a title, then "120 items · Title (A–Z)".
  function updateShelfHead() {
    // The catalogue is also driven headless in the tests, where there is no page to write on.
    if (typeof document === "undefined") return;
    const title = document.getElementById('shelf-title');
    const meta = document.getElementById('shelf-meta');
    if (!title || !meta) return;
    const select = document.querySelector('.shelf-switcher select');
    const name = (select?.selectedOptions?.[0]?.textContent || '').trim();
    title.textContent = name || 'Library';
    // The sort control sits on this line and already names the order, so the count is all this says.
    meta.textContent = total ? `${total} ${total === 1 ? 'item' : 'items'}` : '';
  }

  async function load(reset = false, requestedOffset = 0) {
    if (reset) {
      abortSeries();
      request?.abort();
      // A different library, search or sort means any earlier-page request still
      // in flight belongs to a catalogue that no longer exists.
      generation += 1;
      previousRequest?.abort();
      previousRequest = null;
    }
    const currentRequest = new AbortControllerClass();
    request = currentRequest;
    if (reset) {
      startOffset = Math.max(0, requestedOffset);
      books = [];
      total = 0;
      navigation = [];
      onNavigationChange();
      // Every sort orders the shelf, so the rail has to come back with each
      // reload; it was being dropped for every sort but the default one.
      void loadNavigation();
      bucketTracker.reset();
      cards.releaseCovers(container);
      container.innerHTML = "";
      status.textContent = "Opening the shelves…";
      updateShelfHead();
    }
    const offset = reset ? startOffset : startOffset + books.length;
    try {
      if (!reset) {
        $("#load-more").disabled = true;
        $("#load-more").textContent = "Loading…";
      }
      const { data, response } = await requestJSON(
        `/api/books?${queryFor(offset)}`,
        { signal: currentRequest.signal },
      );
      if (request !== currentRequest) return false;
      total = Number(response.headers.get("X-Total-Count")) || data.length;
      books.push(...data);
      render(data, reset ? "replace" : "append");
      updateShelfHead();
      return true;
    } catch (error) {
      if (error.name !== "AbortError" && request === currentRequest) {
        if (reset) {
          status.textContent = "";
          renderStatePanel(container, errorSpec(error, "your books"), stateHandlers(() => load(true, requestedOffset)));
        } else status.textContent = `Could not load more books: ${error.message}`;
      }
      return false;
    } finally {
      nextPager?.refresh();
    }
  }

  async function loadMore() {
    if (continuation) return continuation;
    continuation = load(false);
    try {
      return await continuation;
    } finally {
      continuation = null;
      nextPager?.refresh();
    }
  }

  async function loadPrevious() {
    if (prepending || startOffset <= 0) return prepending;
    const currentPrevious = new AbortControllerClass();
    const requestedGeneration = generation;
    previousRequest = currentPrevious;
    prepending = (async () => {
      const offset = Math.max(0, startOffset - pageSize);
      try {
        const { data } = await requestJSON(
          `/api/books?${queryFor(offset, startOffset - offset)}`,
          { signal: currentPrevious.signal },
        );
        // Late pages used to be prepended into the next catalogue with the wrong
        // offset; the response only counts for the catalogue that asked for it.
        if (requestedGeneration !== generation || previousRequest !== currentPrevious) return false;
        if (!data.length) return false;
        startOffset = offset;
        books.unshift(...data);
        render(data, "prepend");
        return true;
      } catch (error) {
        if (error.name !== "AbortError" && requestedGeneration === generation)
          status.textContent = `Could not load earlier books: ${error.message}`;
        return false;
      }
    })();
    try {
      return await prepending;
    } finally {
      if (previousRequest === currentPrevious) previousRequest = null;
      prepending = null;
      previousPager?.refresh();
    }
  }

  async function jump(bucket) {
    if (!bucket) {
      request?.abort();
      await load(true, 0);
      $("#book-catalog").scrollIntoView({ block: "start" });
      setCurrentBucket("");
      return;
    }
    const target = navigation.find((item) => item.bucket === bucket);
    if (!target) return;
    request?.abort();
    await load(true, Math.max(0, target.offset - 40));
    const card = [...container.children].find(
      (item) => item.dataset.bucket === bucket,
    );
    card?.scrollIntoView({ block: "start" });
    setCurrentBucket(bucket);
  }

  function bind() {
    nextPager = createContinuousPager({
      sentinel: $("#book-sentinel"),
      button: $("#load-more"),
      label: "books",
      canLoad: () =>
        isActive() &&
        !$("#book-catalog").hidden &&
        !continuation &&
        !prepending &&
        startOffset + books.length < total,
      load: loadMore,
    });
    previousPager = createContinuousPager({
      sentinel: $("#book-start-sentinel"),
      button: $("#book-load-previous"),
      label: "previous books",
      canLoad: () =>
        isActive() &&
        !$("#book-catalog").hidden &&
        !continuation &&
        !prepending &&
        startOffset > 0,
      load: loadPrevious,
      rootMargin: "500px",
    });
  }

  function updateCurrentFromScroll(top) {
    const bucket = bucketTracker.find(container.children, top);
    if (bucket) setCurrentBucket(bucket);
  }

  function abort() {
    request?.abort();
    navigationRequest?.abort();
  }

  return {
    abort,
    bind,
    jump,
    load,
    navigation: () => navigation,
    updateCurrentFromScroll,
  };
}
