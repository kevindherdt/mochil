// What the Hardcover shelf sync matched each book to, so a wrong guess can be spotted and changed, and the books
// it could not match, so they can be linked by hand. It only shows while the Hardcover module is on.
const escape = (value) =>
  String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

export const hardcoverShelfHTML = `
  <details class="settings-accordion hardcover-shelf" data-hardcover-shelf hidden>
    <summary><span><strong>Hardcover shelf</strong><small>What your books are matched to</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
    <div class="settings-accordion-body">
      <p>Books are matched to Hardcover by title and author. Check the matches here: if one is wrong, press Change and pick the right book. Books that could not be matched wait under “Needs a link” after you read them. “Stop syncing” leaves a book alone, and a book you remove from your Hardcover shelf is left off automatically.</p>
      <div id="hardcover-shelf-list"></div>
      <p id="hardcover-shelf-status" role="status"></p>
    </div>
  </details>`;

export function mountHardcoverShelf(document) {
  if (document.querySelector("[data-hardcover-shelf]")) return;
  document
    .querySelector('[data-settings-panel="metadata"]')
    .insertAdjacentHTML("beforeend", hardcoverShelfHTML);
}

export function createHardcoverShelf({ document, $, requestJSON }) {
  mountHardcoverShelf(document);
  const root = () => $("[data-hardcover-shelf]");
  let skipped = [];
  let matched = [];
  let ignored = [];
  let open = null; // the book being linked: {library, book, query, results, busy}

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

  function render() {
    const needing = skipped.length
      ? `<ul class="hardcover-skipped">${skipped.map((item) => rowMarkup(item, null)).join("")}</ul>`
      : "<p>Nothing needs a link. Books Hardcover cannot match by name will appear here after you read them.</p>";
    const done = matched.length
      ? `<ul class="hardcover-skipped">${matched.map((match) => rowMarkup(match, match)).join("")}</ul>`
      : "<p>No books have been sent to Hardcover yet. Read a page of one to see what it is matched to.</p>";
    const left = ignored.length
      ? `<h3>Not syncing</h3><p>These books are left alone: reading them changes nothing on Hardcover.</p><ul class="hardcover-skipped">${ignored
          .map((item) => `<li><div class="hardcover-skipped-row"><span><strong>${escape(item.title)}</strong><small>${escape(item.reason)}</small></span><button type="button" data-action="resume" data-library="${escape(item.library)}" data-book="${escape(item.book)}">Sync again</button></div></li>`)
          .join("")}</ul>`
      : "";
    $("#hardcover-shelf-list").innerHTML = `<h3>Needs a link</h3>${needing}<h3>Matched to Hardcover</h3>${done}${left}`;
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
    else if (action === "stop") setIgnored(target.dataset.library, target.dataset.book, true);
    else if (action === "resume") setIgnored(target.dataset.library, target.dataset.book, false);
  }

  return {
    bind() {
      $("#hardcover-shelf-list").addEventListener("click", onClick);
      root().addEventListener("toggle", () => {
        if (root().open) load();
      });
    },
    load,
  };
}
