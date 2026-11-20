// What the Hardcover shelf sync matched each book to, so a wrong guess can be spotted and changed, and the books
// it could not match, so they can be linked by hand. It only shows while the Hardcover module is on.
//
// A shelf reaches thousands of books, so nothing here loads a whole list: each of the four groups is fetched a
// page at a time with an honest total from the server, and the search field is answered by the server too.
const escape = (value) =>
  String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

export const hardcoverShelfHTML = `
  <details class="settings-accordion hardcover-shelf" data-hardcover-shelf hidden>
    <summary><span><strong>Hardcover shelf</strong><small>What your books are matched to</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
    <div class="settings-accordion-body">
      <p>Books are matched by title and author. Change fixes a wrong match; Stop syncing leaves a book alone.</p>
      <p id="hardcover-shelf-summary" class="hardcover-summary"></p>
      <input id="hardcover-shelf-filter" type="search" placeholder="Find a book by title or author" aria-label="Find a book in your Hardcover matches" />
      <div id="hardcover-shelf-list"></div>
      <p id="hardcover-shelf-status" role="status"></p>
    </div>
  </details>`;

export function mountHardcoverShelf(document) {
  if (document.querySelector("[data-hardcover-shelf]")) return;
  document
    .querySelector('[data-settings-panel="libraries"]')
    .insertAdjacentHTML("beforeend", hardcoverShelfHTML);
}

export const PAGE = 25;

// The four things the shelf can say about a book, and the request that answers each.
const GROUPS = {
  skipped: { path: "/api/metadata/hardcover/skipped", heading: "Needs a link", note: "Hardcover has no book by this name" },
  doubts: { path: "/api/metadata/hardcover/matches?doubt=1", heading: "Check these", note: "matched, but not certainly" },
  matched: { path: "/api/metadata/hardcover/matches?doubt=0", heading: "Matched", note: "" },
  ignored: { path: "/api/metadata/hardcover/ignored", heading: "Not syncing", note: "reading them changes nothing" },
};

export function createHardcoverShelf({ document, $, requestJSON }) {
  mountHardcoverShelf(document);
  const root = () => $("[data-hardcover-shelf]");
  const blank = () => ({ items: [], total: 0, done: false, busy: false });
  const state = { skipped: blank(), doubts: blank(), matched: blank(), ignored: blank() };
  let query = "";
  const shown = new Set(); // groups that hold nothing wrong, opened by hand
  let open = null; // the book being linked: {library, book, query, results, busy}

  const keyOf = (item) => `${item.library}/${item.book}`;
  const status = (text) => ($("#hardcover-shelf-status").textContent = text || "");

  function url(group, offset) {
    const params = new URLSearchParams({ limit: String(PAGE), offset: String(offset) });
    if (query) params.set("q", query);
    const base = GROUPS[group].path;
    return `${base}${base.includes("?") ? "&" : "?"}${params}`;
  }

  async function fetchPage(group, offset) {
    const data = (await requestJSON(url(group, offset))).data || {};
    const items = data.items || [];
    const target = state[group];
    target.items = offset === 0 ? items : target.items.concat(items);
    target.total = typeof data.total === "number" ? data.total : target.items.length;
    target.done = target.items.length >= target.total || items.length === 0;
    if (group === "skipped" && data.enabled === false) root().hidden = true;
  }

  async function refresh() {
    for (const group of Object.keys(GROUPS)) {
      state[group] = blank();
      state[group].busy = true;
    }
    render();
    try {
      await Promise.all(Object.keys(GROUPS).map((group) => fetchPage(group, 0)));
    } catch (error) {
      root().hidden = true; // not an administrator, or the module is unavailable
      status("");
      return;
    }
    for (const group of Object.keys(GROUPS)) state[group].busy = false;
    render();
  }

  async function loadMore(group) {
    if (state[group].busy || state[group].done) return;
    state[group].busy = true;
    render();
    try {
      await fetchPage(group, state[group].items.length);
    } catch (error) {
      status(error.message || "That could not be loaded.");
    }
    state[group].busy = false;
    render();
  }

  function resultsMarkup(lookup) {
    if (lookup.busy) return "<p>Searching Hardcover…</p>";
    if (!lookup.results) return "";
    if (!lookup.results.length) return "<p>Nothing found. Try fewer words.</p>";
    return `<ul class="hardcover-results">${lookup.results
      .map(
        (result) =>
          `<li><span><strong>${escape(result.title)}</strong><small>${escape((result.authors || []).join(", "))}${result.year ? ` · ${escape(result.year)}` : ""}</small></span>` +
          `<a href="${escape(result.source_url)}" target="_blank" rel="noopener noreferrer">View</a>` +
          `<button type="button" class="primary" data-action="choose" data-id="${escape(result.provider_id)}">Choose</button></li>`,
      )
      .join("")}</ul>`;
  }

  function rowMarkup(item, match) {
    const lookup = open && keyOf(open) === keyOf(item) ? open : null;
    const authors = (item.authors || []).join(", ");
    let detail = "";
    if (match) {
      const by = (match.hardcover_authors || []).join(", ");
      const how = match.how === "linked" ? "linked by you" : "matched by name";
      detail =
        `<small class="hardcover-match">On Hardcover: <strong>${escape(match.hardcover_title || `book ${match.hardcover_id}`)}</strong>${by ? ` by ${escape(by)}` : ""} · ${how}` +
        (match.url ? ` · <a href="${escape(match.url)}" target="_blank" rel="noopener noreferrer">View</a>` : "") +
        `</small>` +
        (match.doubt ? `<small class="hardcover-doubt">Check this one: ${escape(match.doubt)}</small>` : "");
    }
    return (
      `<li data-key="${escape(keyOf(item))}"><div class="hardcover-skipped-row"><span><strong>${escape(item.title)}</strong>${authors ? `<small>${escape(authors)}</small>` : ""}${detail}</span>` +
      (lookup ? "" : `<button type="button" data-action="find" data-library="${escape(item.library)}" data-book="${escape(item.book)}">${match ? "Change" : "Find on Hardcover"}</button>`) +
      (lookup || !match ? "" : `<button type="button" data-action="stop" data-library="${escape(item.library)}" data-book="${escape(item.book)}">Stop syncing</button>`) +
      `</div>` +
      (lookup
        ? `<div class="hardcover-search"><input id="hardcover-search-input" value="${escape(lookup.query)}" aria-label="Search Hardcover" /><button type="button" data-action="search">Search</button><button type="button" data-action="cancel">Cancel</button></div>${resultsMarkup(lookup)}`
        : "") +
      `</li>`
    );
  }

  const ignoredRow = (item) =>
    `<li><div class="hardcover-skipped-row"><span><strong>${escape(item.title)}</strong><small>${escape(item.reason)}</small></span><button type="button" data-action="resume" data-library="${escape(item.library)}" data-book="${escape(item.book)}">Sync again</button></div></li>`;

  function section(group, rowFor, { alwaysOpen = false } = {}) {
    const { items, total, done, busy } = state[group];
    const meta = GROUPS[group];
    // "Matched" and "Not syncing" hold nothing that needs a decision: they stay a count until asked for.
    if (!alwaysOpen && !shown.has(group) && !query) {
      if (!total) return "";
      const phrase =
        group === "matched"
          ? `${total} ${total === 1 ? "book looks" : "books look"} right.`
          : `${total} left alone.`;
      return `<p class="hardcover-quiet">${phrase} <button type="button" class="hardcover-more" data-action="more" data-group="${group}">Show them</button></p>`;
    }
    if (!total) return "";
    const more = !done
      ? `<button type="button" class="hardcover-more" data-action="page" data-group="${group}"${busy ? " disabled" : ""}>${busy ? "Loading…" : `Show ${Math.min(PAGE, total - items.length)} more of ${total}`}</button>`
      : "";
    return `<h3>${escape(meta.heading)} <small>${total}</small>${meta.note ? ` <small>${escape(meta.note)}</small>` : ""}</h3><ul class="hardcover-skipped">${items
      .map(rowFor)
      .join("")}</ul>${more}`;
  }

  function render() {
    const { skipped, doubts, matched, ignored } = state;
    const heading = $("#hardcover-shelf-summary");
    if (heading) {
      heading.textContent = [
        `${matched.total + doubts.total} sent to Hardcover`,
        `${skipped.total} need${skipped.total === 1 ? "s" : ""} a link`,
        doubts.total ? `${doubts.total} to check` : "",
        ignored.total ? `${ignored.total} left alone` : "",
      ]
        .filter(Boolean)
        .join(" · ");
    }
    const parts = [
      section("skipped", (item) => rowMarkup(item, null), { alwaysOpen: true }),
      section("doubts", (match) => rowMarkup(match, match), { alwaysOpen: true }),
      section("matched", (match) => rowMarkup(match, match)),
      section("ignored", ignoredRow),
    ];
    const nothing = !skipped.total && !doubts.total && !matched.total && !ignored.total;
    $("#hardcover-shelf-list").innerHTML = parts.join("")
      || (nothing
        ? query
          ? "<p>No book in your shelf matches that.</p>"
          : "<p>Nothing has been sent to Hardcover yet. Read a page of a book to see what it is matched to.</p>"
        : "");
  }

  async function search() {
    const field = $("#hardcover-search-input");
    // The box holds what the reader typed; before it is drawn it holds nothing, and the book's own title is
    // what the first lookup uses.
    const value = (field?.value || "").trim() || open?.query || "";
    if (!open || !value) return;
    open.query = value;
    open.busy = true;
    open.results = null;
    render();
    try {
      open.results = (await requestJSON(`/api/metadata/hardcover/search?q=${encodeURIComponent(value)}`)).data;
      status("");
    } catch (error) {
      open.results = [];
      status(error.message || "Hardcover could not be reached.");
    }
    open.busy = false;
    render();
  }

  async function choose(providerID) {
    if (!open) return;
    const { library, book } = open;
    try {
      await requestJSON("/api/metadata/hardcover/link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ library, book, provider_id: providerID }),
      });
      open = null;
      status("Linked. The book is being sent to your Hardcover shelf now. If an earlier wrong match was already sent, remove it on hardcover.app.");
      await refresh();
      return;
    } catch (error) {
      status(error.message || "The link could not be saved.");
    }
    render();
  }

  async function setIgnored(library, book, ignore) {
    try {
      await requestJSON("/api/metadata/hardcover/ignore", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ library, book, ignore }),
      });
      status(ignore ? "Not syncing this book any more. It stays on Hardcover as it is." : "Syncing this book again.");
      await refresh();
    } catch (error) {
      status(error.message || "That could not be saved.");
    }
  }

  function onClick(event) {
    const target = event.target.closest?.("[data-action]");
    if (!target) return;
    const { action } = target.dataset;
    if (action === "find") {
      const item = [...state.skipped.items, ...state.doubts.items, ...state.matched.items].find(
        (entry) => entry.library === target.dataset.library && entry.book === target.dataset.book,
      );
      if (!item) return;
      open = { library: item.library, book: item.book, query: item.title, results: null, busy: false };
      status("");
      render();
      search();
    } else if (action === "search") search();
    else if (action === "cancel") {
      open = null;
      render();
    } else if (action === "choose") choose(target.dataset.id);
    else if (action === "stop") setIgnored(target.dataset.library, target.dataset.book, true);
    else if (action === "resume") setIgnored(target.dataset.library, target.dataset.book, false);
    else if (action === "more") {
      // The first page is already in hand, only hidden behind the count: opening it shows that, it does not
      // fetch a second one.
      shown.add(target.dataset.group);
      if (state[target.dataset.group].items.length) render();
      else loadMore(target.dataset.group);
    } else if (action === "page") loadMore(target.dataset.group);
  }

  return {
    bind() {
      $("#hardcover-shelf-list").addEventListener("click", onClick);
      // One field for the whole shelf, answered by the server: the panel never holds more than a page of it.
      const field = $("#hardcover-shelf-filter");
      let timer = null;
      if (field) {
        field.addEventListener("input", () => {
          clearTimeout(timer);
          timer = setTimeout(() => {
            query = field.value.trim();
            refresh();
          }, 250);
        });
      }
      root().addEventListener("toggle", () => {
        if (root().open) refresh();
      });
    },
    load: refresh,
  };
}
