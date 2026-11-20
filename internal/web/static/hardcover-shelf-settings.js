// What the Hardcover shelf sync matched each book to, so a wrong guess can be spotted and changed, and the books
// it could not match, so they can be linked by hand. It only shows while the Hardcover module is on.
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

export function createHardcoverShelf({ document, $, requestJSON }) {
  mountHardcoverShelf(document);
  const root = () => $("[data-hardcover-shelf]");
  let skipped = [];
  let matched = [];
  let ignored = [];
  let open = null; // the book being linked: {library, book, query, results, busy}
  // A library grows past a few hundred books, so the panel cannot list everything it knows. It shows what
  // is wrong — books with no link, and matches worth a second look — and gives the rest as a count that
  // expands on request, with one filter that searches the whole set.
  let filter = "";
  const expanded = new Set();
  const PAGE = 8;

  const keyOf = (item) => `${item.library}/${item.book}`;
  const status = (text) => ($("#hardcover-shelf-status").textContent = text || "");

  function resultsMarkup(search) {
    if (search.busy) return "<p>Searching Hardcover…</p>";
    if (!search.results) return "";
    if (!search.results.length) return "<p>Nothing found. Try fewer words.</p>";
    return `<ul class="hardcover-results">${search.results
      .map(
        (result) =>
          `<li><span><strong>${escape(result.title)}</strong><small>${escape((result.authors || []).join(", "))}${result.year ? ` · ${escape(result.year)}` : ""}</small></span>` +
          `<a href="${escape(result.source_url)}" target="_blank" rel="noopener noreferrer">View</a>` +
          `<button type="button" class="primary" data-action="choose" data-id="${escape(result.provider_id)}">Choose</button></li>`,
      )
      .join("")}</ul>`;
  }

  function rowMarkup(item, match) {
    const search = open && keyOf(open) === keyOf(item) ? open : null;
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
      (search ? "" : `<button type="button" data-action="find" data-library="${escape(item.library)}" data-book="${escape(item.book)}">${match ? "Change" : "Find on Hardcover"}</button>`) +
      (search || !match ? "" : `<button type="button" data-action="stop" data-library="${escape(item.library)}" data-book="${escape(item.book)}">Stop syncing</button>`) +
      `</div>` +
      (search
        ? `<div class="hardcover-search"><input id="hardcover-search-input" value="${escape(search.query)}" aria-label="Search Hardcover" /><button type="button" data-action="search">Search</button><button type="button" data-action="cancel">Cancel</button></div>${resultsMarkup(search)}`
        : "") +
      `</li>`
    );
  }

  const matchesFilter = (item, match) =>
    !filter || `${item.title} ${(item.authors || []).join(" ")} ${match?.hardcover_title || ""}`.toLowerCase().includes(filter);

  function group(heading, items, rowFor, note) {
    if (!items.length) return "";
    const key = heading;
    const shownAll = expanded.has(key) || filter;
    const visible = shownAll ? items : items.slice(0, PAGE);
    const more =
      items.length > visible.length
        ? `<button type="button" class="hardcover-more" data-action="more" data-group="${escape(key)}">Show all ${items.length}</button>`
        : "";
    return `<h3>${escape(heading)}${note ? ` <small>${escape(note)}</small>` : ""}</h3><ul class="hardcover-skipped">${visible
      .map(rowFor)
      .join("")}</ul>${more}`;
  }

  function render() {
    const needsLink = skipped.filter((item) => matchesFilter(item, null));
    const doubts = matched.filter((match) => match.doubt && matchesFilter(match, match));
    const rest = matched.filter((match) => !match.doubt && matchesFilter(match, match));
    const left = ignored.filter((item) => matchesFilter(item, null));

    // Every matched book that looks right is one number, not a row: it is the normal state and there is
    // nothing to do about it.
    const summary = [
      `${matched.length} sent to Hardcover`,
      `${skipped.length} need${skipped.length === 1 ? "s" : ""} a link`,
      doubts.length ? `${doubts.length} to check` : "",
      ignored.length ? `${ignored.length} left alone` : "",
    ]
      .filter(Boolean)
      .join(" · ");
    const heading = $("#hardcover-shelf-summary");
    if (heading) heading.textContent = summary;

    const parts = [
      group("Needs a link", needsLink, (item) => rowMarkup(item, null), "Hardcover has no book by this name"),
      group("Check these", doubts, (match) => rowMarkup(match, match), "matched, but not certainly"),
      filter || expanded.has("Matched")
        ? group("Matched", rest, (match) => rowMarkup(match, match))
        : rest.length
          ? `<h3>Matched</h3><p class="hardcover-quiet">${rest.length} more ${rest.length === 1 ? "book looks" : "books look"} right. <button type="button" class="hardcover-more" data-action="more" data-group="Matched">Show them</button></p>`
          : "",
      expanded.has("Not syncing") || filter
        ? group("Not syncing", left, (item) => `<li><div class="hardcover-skipped-row"><span><strong>${escape(item.title)}</strong><small>${escape(item.reason)}</small></span><button type="button" data-action="resume" data-library="${escape(item.library)}" data-book="${escape(item.book)}">Sync again</button></div></li>`, "reading them changes nothing")
        : left.length
          ? `<p class="hardcover-quiet">${left.length} left alone. <button type="button" class="hardcover-more" data-action="more" data-group="Not syncing">Show them</button></p>`
          : "",
    ];
    const empty = !matched.length && !skipped.length
      ? "<p>Nothing has been sent to Hardcover yet. Read a page of a book to see what it is matched to.</p>"
      : matched.length || skipped.length
        ? ""
        : "<p>Nothing here yet.</p>";
    const list = $("#hardcover-shelf-list");
    list.innerHTML = parts.join("") + empty;
  }

  async function load() {
    try {
      const [lacking, found, left] = await Promise.all([
        requestJSON("/api/metadata/hardcover/skipped"),
        requestJSON("/api/metadata/hardcover/matches"),
        requestJSON("/api/metadata/hardcover/ignored"),
      ]);
      ignored = left.data.items || [];
      skipped = lacking.data.items || [];
      // matches worth a second look come first
      matched = [...(found.data.items || [])].sort((a, b) => Boolean(b.doubt) - Boolean(a.doubt));
      root().hidden = !lacking.data.enabled;
      if (lacking.data.enabled) render();
    } catch (_) {
      root().hidden = true; // not an administrator, or the module is unavailable
    }
  }

  async function search() {
    const query = $("#hardcover-search-input")?.value.trim();
    if (!open || !query) return;
    open.query = query;
    open.busy = true;
    open.results = null;
    render();
    try {
      open.results = (await requestJSON(`/api/metadata/hardcover/search?q=${encodeURIComponent(query)}`)).data;
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
      await load();
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
      await load();
    } catch (error) {
      status(error.message || "That could not be saved.");
    }
  }

  function onClick(event) {
    const target = event.target.closest?.("[data-action]");
    if (!target) return;
    const { action } = target.dataset;
    if (action === "find") {
      const item = [...skipped, ...matched].find((entry) => entry.library === target.dataset.library && entry.book === target.dataset.book);
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
    else if (action === "more") {
      expanded.add(target.dataset.group);
      render();
    }
    else if (action === "stop") setIgnored(target.dataset.library, target.dataset.book, true);
    else if (action === "resume") setIgnored(target.dataset.library, target.dataset.book, false);
  }

  return {
    bind() {
      $("#hardcover-shelf-list").addEventListener("click", onClick);
      // One field for the whole set: with hundreds of matches, finding the one wrong row is the actual job.
      const search = $("#hardcover-shelf-filter");
      if (search) {
        search.addEventListener("input", () => {
          filter = search.value.trim().toLowerCase();
          render();
        });
      }
      root().addEventListener("toggle", () => {
        if (root().open) load();
      });
    },
    load,
  };
}
