import { createBucketTracker } from './catalog-scroll-position.js?v=2'
import { bindSeriesHub, seriesHubHTML } from './series-hub.js?v=6'
import { createListReturn } from './catalogue-return.js?v=1'
import { createDetailRestore } from './detail-restore.js?v=3'
import { emptySpec, errorSpec, renderStatePanel } from './state-panel.js?v=2'

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

export function seriesContext(item, showLibrary) {
  const authors = (item.authors || []).filter(
    (name) => name && name.toLowerCase() !== "unknown",
  );
  const details = authors.length ? [authors.join(", ")] : [];
  if (showLibrary && item.library_name) details.push(item.library_name);
  return details.join(" · ");
}

export function createSeriesCatalogue({
  $,
  document,
  escapeHTML,
  requestJSON,
  createBoundedWindow,
  createContinuousPager,
  cards,
  collections,
  libraryState,
  getBrowseMode,
  getSeriesLayout,
  getItemLayout,
  getActiveBucket,
  abortBooks,
  AbortController: AbortControllerClass = AbortController,
}) {
  let request;
  let groups = [];
  // Offset of the first series row in the list; trimming advances it so paging
  // back can ask for exactly the rows the window dropped.
  let windowStart = 0;
  let total = 0;
  let loading = false;
  let loadingEarlier = false;
  let earlierPager;
  let pager;
  let groupWindow;
  let inflight = null;
  const bucketTracker = createBucketTracker();
  const detailPagers = new Set();
  const restore = createDetailRestore();
  const index = $("#series-index");

  const context = (item) => seriesContext(item, !libraryState.selected());

  function disconnectDetailPagers() {
    detailPagers.forEach((detailPager) => detailPager.disconnect());
    detailPagers.clear();
  }

  function installDetailPager(target, loaded, itemTotal, load) {
    target._seriesPager?.disconnect();
    if (target._seriesPager) detailPagers.delete(target._seriesPager);
    target.querySelector(".series-continuation")?.remove();
    if (loaded >= itemTotal) {
      target._seriesPager = null;
      return;
    }
    const continuation = document.createElement("div");
    continuation.className = "series-continuation";
    continuation.innerHTML =
      '<div class="pagination-sentinel" aria-hidden="true"></div><button class="load-more series-more" hidden>Load more books</button>';
    target.append(continuation);
    const disclosure = target.closest("details");
    const detailPager = createContinuousPager({
      sentinel: continuation.firstElementChild,
      button: continuation.lastElementChild,
      label: "books",
      canLoad: () =>
        target.isConnected &&
        (!disclosure || disclosure.open) &&
        loaded < itemTotal,
      load,
    });
    target._seriesPager = detailPager;
    detailPagers.add(detailPager);
  }

  async function loadBooksPage(item, target, search, offset = 0, cursor = "") {
    const params = new URLSearchParams({
      library: item.library_id,
      series: item.unsorted ? "" : item.name,
      limit: 500,
    });
    cursor ? params.set("cursor", cursor) : params.set("offset", offset);
    if (search) params.set("q", search);
    const result = await requestJSON(`/api/books?${params}`);
    let grid = target.querySelector(".grid");
    if (!grid) {
      target.innerHTML = '<div class="grid"></div>';
      grid = target.querySelector(".grid");
    }
    const layout = getItemLayout();
    grid.classList.toggle("book-list", layout === "list");
    grid.insertAdjacentHTML(
      "beforeend",
      result.data.map((book) => cards.markup(book, layout, false)).join(""),
    );
    cards.installCoverFallbacks(grid);
    collections.installActions(grid);
    target.dispatchEvent(new CustomEvent("series-books", { detail: result.data }));
    const itemTotal =
      Number(result.response.headers.get("X-Total-Count")) ||
      result.data.length;
    const loaded = offset + result.data.length;
    const next = result.response.headers.get("X-Next-Cursor") || "";
    installDetailPager(target, loaded, itemTotal, () =>
      loadBooksPage(item, target, search, loaded, next),
    );
  }

  const listReturn = createListReturn({ count: () => groups.length, load: () => load(), loadMore: () => load(true) });

  async function open(item, search) {
    if (!item) return;
    restore.set(item.name);
    listReturn.remember();
    disconnectDetailPagers();
    cards.releaseCovers(index);
    index.innerHTML = `${seriesHubHTML(item, { escapeHTML, authors: seriesContext(item, false), saveIcon: cards.bookmarkIcon })}<div class="series-focus-books"><p>Loading books…</p></div>`;
    const feedHub = bindSeriesHub(index, item.count);
    index.querySelector(".series-focus-books").addEventListener("series-books", (event) => feedHub(event.detail));
    cards.installCoverFallbacks(index);
    collections.installActions(index);
    index.querySelector(".back-button").onclick = () => listReturn.restore();
    const target = index.querySelector(".series-focus-books");
    try {
      target.innerHTML = "";
      await loadBooksPage(item, target, search);
    } catch {
      target.innerHTML =
        '<p role="alert">Could not load this series. Please go back and try again.</p>';
    }
  }

  function bindCoverActions(search) {
    cards.installCoverFallbacks(index);
    collections.installActions(index);
    index
      .querySelectorAll(".series-card button[data-index]:not([data-ready])")
      .forEach((button) => {
        button.dataset.ready = "true";
        button.onclick = () =>
          open(groups[Number(button.dataset.index)], search);
      });
  }

  function bindListActions(search) {
    cards.installCoverFallbacks(index);
    collections.installActions(index);
    index
      .querySelectorAll(".series-row:not([data-ready])")
      .forEach((details) => {
        details.dataset.ready = "true";
        details.addEventListener("toggle", async () => {
          if (!details.open || details.dataset.loaded) return;
          details.dataset.loaded = "true";
          const item = groups[Number(details.dataset.index)];
          const target = details.querySelector(".series-books");
          try {
            await loadBooksPage(item, target, search);
          } catch {
            details.dataset.loaded = "";
            target.innerHTML =
              '<p role="alert">Could not load this series. Collapse and reopen it to retry.</p>';
          }
        });
      });
  }

  function onNodesRemoved(nodes) {
    nodes.forEach((node) => {
      cards.releaseCovers(node);
      const pager = node.querySelector?.(".series-books")?._seriesPager;
      if (pager) {
        pager.disconnect();
        detailPagers.delete(pager);
      }
    });
  }

  // The rows of one page, numbered from `start` so the click handlers resolve the
  // right series. Redrawing the retained window reuses this after a trim, when
  // every remaining index has shifted.
  function seriesMarkup(series, covers, start) {
      const rows = covers
      ? series
          .map((item, position) => {
            const detail = context(item);
            const countLabel = `${item.count.toLocaleString()} ${item.count === 1 ? "book" : "books"}`;
            const stack = item.count > 1 && item.cover_url ? ` is-stack" style="--stack-cover:url('${escapeHTML(encodeURI(item.cover_url))}')` : "";
            // The visible count and authors must survive into the accessible
            // name; naming the button "Open <title>" alone dropped both.
            const openLabel = `Open ${item.name}, ${countLabel}${detail ? `, ${detail}` : ""}`;
            return `<article class="series-card${stack}" data-bucket="${escapeHTML(item.bucket || '#')}"><button data-index="${start + position}" aria-label="${escapeHTML(openLabel)}"><span class="cover-frame"><img class="cover" decoding="async" data-src="${escapeHTML(item.cover_url)}" data-title="${escapeHTML(item.name)}" alt="Representative cover for ${escapeHTML(item.name)}"><span class="series-count-badge" aria-label="${countLabel}">${item.count.toLocaleString()}</span></span><h2>${escapeHTML(item.name)}</h2>${detail ? `<p>${escapeHTML(detail)}</p>` : ""}</button><button class="collection-save series-save" data-type="series" data-library="${escapeHTML(item.library_id)}" data-series="${escapeHTML(item.name)}" aria-label="Save ${escapeHTML(item.name)} to collection">${cards.bookmarkIcon}</button></article>`;
          })
          .join("")
      : series
          .map((item, position) => {
            const detail = context(item);
            return `<details class="series-row" data-index="${start + position}" data-bucket="${escapeHTML(item.bucket || '#')}"><summary><span><strong>${escapeHTML(item.name)}</strong>${detail ? `<small>${escapeHTML(detail)}</small>` : ""}</span><span class="series-count">${item.count.toLocaleString()} ${item.count === 1 ? "book" : "books"}</span><span class="series-row-cover"><img class="cover" decoding="async" data-src="${escapeHTML(item.cover_url)}" data-title="${escapeHTML(item.name)}" alt=""></span><button class="collection-save series-row-save" data-type="series" data-library="${escapeHTML(item.library_id)}" data-series="${escapeHTML(item.name)}" aria-label="Save ${escapeHTML(item.name)} to collection">${cards.bookmarkIcon}</button></summary><div class="series-books"><p>Loading books…</p></div></details>`;
          })
          .join("");
    return rows;
  }

  function renderPage(series, append, search) {
    const start = groups.length;
    groups.push(...series);
    const bucket = getActiveBucket();
    $("#status").textContent = bucket ? `Series under ${bucket}` : "";
    collections.syncSaveSearch?.(groups.length > 0);
    if (!groups.length) {
      renderStatePanel(index, emptySpec({ noun: "series", query: $("#search")?.value || "", bucket }), stateHandlers(() => load(false)));
      return;
    }
    const covers = getSeriesLayout() === "covers";
    const wrapperClass = covers ? "series-cover-grid" : "series-list";
    const wrapperSelector = `.${wrapperClass}`;
    if (!append) {
      index.innerHTML = `<div class="pagination-sentinel" data-earlier-sentinel aria-hidden="true"></div><button type="button" class="load-more" data-earlier-more hidden>Load earlier series</button><div class="virtual-spacer" data-top-spacer aria-hidden="true"></div><div class="${wrapperClass}"></div><div class="virtual-spacer" data-bottom-spacer aria-hidden="true"></div>`;
      groupWindow = createBoundedWindow({
        container: index.querySelector(wrapperSelector),
        topSpacer: index.querySelector("[data-top-spacer]"),
        bottomSpacer: index.querySelector("[data-bottom-spacer]"),
        onRemove: onNodesRemoved,
      });
    }
    const wrapper = index.querySelector(wrapperSelector);
    // A late page must not clobber a series that was opened meanwhile.
    if (!wrapper) return;
    if (!append) installEarlierPager();
    const insertion = groupWindow.begin("append");
    wrapper.insertAdjacentHTML("beforeend", seriesMarkup(series, covers, start + windowStart));
    const removed = groupWindow.finish(insertion);
    // `start` is how many rows were on screen before this page, so the window can
    // only have trimmed those. Redraw them all with fresh indices: a stale index
    // opens the wrong series.
    const trimmed = Math.min(removed.start, start);
    if (trimmed > 0) {
      groups.splice(0, trimmed);
      windowStart += trimmed;
      wrapper.innerHTML = seriesMarkup(groups, covers, windowStart);
    }
    covers ? bindCoverActions(search) : bindListActions(search);
  }

  function loadKey() {
    return [libraryState.selected(), getActiveBucket(), $("#search").value.trim()].join("\u0000");
  }

  // Two callers run loadCurrentView per navigation (the route and the library
  // loader), which used to abort the first /api/series and fetch it again 8 ms
  // later. A second non-append load for the same library, bucket and query is
  // now the same request; a changed library still starts a fresh one.
  function load(append = false) {
    if (!append) {
      const key = loadKey();
      if (inflight && inflight.key === key) return inflight.promise;
    }
    const promise = runLoad(append);
    if (!append) {
      const entry = { key: loadKey(), promise };
      inflight = entry;
      const settle = () => { if (inflight === entry) inflight = null; };
      promise.then(settle, settle);
    }
    return promise;
  }

  async function runLoad(append = false) {
    if (loading && append) return;
    abortBooks();
    if (!append) request?.abort();
    const currentRequest = new AbortControllerClass();
    request = currentRequest;
    loading = true;
    $("#book-catalog").hidden = true;
    index.hidden = false;
    if (!append) {
      disconnectDetailPagers();
      cards.releaseCovers(index);
      index.innerHTML = "";
      groups = [];
      windowStart = 0;
      total = 0;
      bucketTracker.reset();
      $("#status").textContent = "Opening the series index…";
    }
    const wanted = append ? "" : restore.take();
    if (!append) restore.set("");
    const query = new URLSearchParams({ limit: 80 });
    if (libraryState.selected()) query.set("library", libraryState.selected());
    if (getActiveBucket()) query.set("bucket", getActiveBucket());
    const search = $("#search").value.trim();
    if (search) query.set("q", search);
    if (append) query.set("offset", String(windowStart + groups.length));
    if (wanted) query.set("q", wanted);
    try {
      const result = await requestJSON(`/api/series?${query}`, {
        signal: currentRequest.signal,
      });
      if (getBrowseMode() !== "series" || request !== currentRequest)
        return false;
      if (wanted) {
        open(result.data.find((item) => item.name === wanted) || { library_id: libraryState.selected(), name: wanted, count: 0, authors: [] }, "");
        return true;
      }
      total =
        Number(result.response.headers.get("X-Total-Count")) ||
        result.data.length;
      renderPage(result.data, append, search);
      return true;
    } catch (error) {
      if (error.status === 409 && append && request === currentRequest) {
        loading = false;
        // A retry of this same view must not be mistaken for the in-flight load.
        return runLoad(false);
      }
      if (error.name !== "AbortError" && request === currentRequest) {
        if (append) $("#status").textContent = `Could not load more series: ${error.message}`;
        else { $("#status").textContent = ""; renderStatePanel(index, errorSpec(error, "your series"), stateHandlers(() => load(false))); }
      }
      return false;
    } finally {
      if (request === currentRequest) loading = false;
      pager?.refresh();
    }
  }

  // Reaching the top fetches the rows the window dropped, at the offset it now
  // starts at, and prepends them inside the same window.
  async function loadEarlier() {
    if (loadingEarlier || windowStart <= 0) return false;
    loadingEarlier = true;
    const key = loadKey();
    const offset = Math.max(0, windowStart - 80);
    const limit = windowStart - offset;
    const query = new URLSearchParams({ limit: String(limit), offset: String(offset) });
    if (libraryState.selected()) query.set("library", libraryState.selected());
    if (getActiveBucket()) query.set("bucket", getActiveBucket());
    const search = $("#search").value.trim();
    if (search) query.set("q", search);
    try {
      const result = await requestJSON(`/api/series?${query}`);
      // The shelf, bucket or search can change while this page is in flight. Its rows
      // belong to the view they were asked for; prepending them onto the new one put
      // the old library at the top of the list and left windowStart above 0, so the
      // next page skipped rows of the shelf actually on screen.
      if (loadKey() !== key) return false;
      const covers = getSeriesLayout() === "covers";
      const wrapper = index.querySelector(covers ? ".series-cover-grid" : ".series-list");
      if (!wrapper) return false;
      const items = result.data || [];
      if (!items.length) { windowStart = 0; return false; }
      const insertion = groupWindow.begin("prepend");
      wrapper.insertAdjacentHTML("afterbegin", seriesMarkup(items, covers, offset));
      const removed = groupWindow.finish(insertion);
      groups.unshift(...items);
      if (removed.end > 0) groups.splice(-removed.end, removed.end);
      windowStart = offset;
      covers ? bindCoverActions(search) : bindListActions(search);
      return true;
    } catch (error) {
      $("#status").textContent = `Could not load earlier series: ${error.message}`;
      return false;
    } finally {
      loadingEarlier = false;
      earlierPager?.refresh();
      pager?.refresh();
    }
  }

  function installEarlierPager() {
    const sentinel = index.querySelector?.("[data-earlier-sentinel]");
    const button = index.querySelector?.("[data-earlier-more]");
    if (!sentinel || !button) return;
    earlierPager?.disconnect();
    earlierPager = createContinuousPager({
      sentinel,
      button,
      label: "earlier series",
      rootMargin: "300px",
      canLoad: () => !index.hidden && getBrowseMode() === "series" && !loading && windowStart > 0,
      load: () => loadEarlier(),
    });
    earlierPager.refresh();
  }

  function bind() {
    pager = createContinuousPager({
      sentinel: $("#series-sentinel"),
      button: $("#series-load-more"),
      label: "series",
      canLoad: () =>
        !index.hidden &&
        getBrowseMode() === "series" &&
        !loading &&
        windowStart + groups.length < total,
      load: () => load(true),
    });
  }

  function abort() {
    request?.abort();
    disconnectDetailPagers();
  }

  function updateCurrentFromScroll(top, setCurrentBucket) {
    const bucket = bucketTracker.find(index.querySelectorAll('.series-card[data-bucket], .series-row[data-bucket]'), top)
    if (bucket) setCurrentBucket(bucket)
  }

  return { abort, bind, load, open, updateCurrentFromScroll, openOnLoad: restore.openOnLoad, opened: restore.opened };
}
