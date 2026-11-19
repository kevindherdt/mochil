// The Tools menu of the library editor: things to do to a library that are not editing a cell. Each tool is one entry
// in TOOLS ({ id, label, hint, run }), so adding another is adding an entry, not another button in the page.

// These tools read every book's file, so they run on the server in the background and the page only watches them.
// The server keeps going if the page is closed, so a page that opens while one is running picks the watching back up.
// Whatever a stopped job already did stays done; the rest is left as it was.
const JOBS = {
  count: { path: "/api/metadata/editor/count", doing: "Counting words and pages for", done: "Counted." },
  check: { path: "/api/metadata/editor/check", doing: "Checking", done: "Checked." },
  isbn: { path: "/api/metadata/editor/isbn", doing: "Looking for ISBNs in", done: "Searched." },
  isbnAsk: { path: "/api/metadata/editor/isbn-ask", doing: "Asking Hermes about ISBNs in", done: "Asked.", detail: true },
  matchAsk: { path: "/api/metadata/editor/match-ask", doing: "Asking Hermes about Hardcover matches in", done: "Asked.", detail: true },
  matches: { path: "/api/metadata/editor/matches", doing: "Finding Hardcover matches for", done: "Looked up." },
};
const wait = () => new Promise((resolve) => setTimeout(resolve, 1000));
const scopeOf = (state) => (state.selected.size ? `${state.selected.size} selected ${state.selected.size === 1 ? "book" : "books"}` : "the whole library");

async function watchJob({ state, api, status, reload, stop }, job, { start = true } = {}) {
  const library = encodeURIComponent(state.library);
  const label = start ? `${job.doing} ${scopeOf(state)}` : `${job.doing} the library`;
  if (start) {
    status(`${label}…`);
    try { await api(job.path, { method: "POST", body: { library: state.library, ids: [...state.selected] } }); }
    catch (error) { status(error.message, "problem"); return; }
  }
  let stopping = false;
  let failures = 0;
  stop?.show(async () => {
    stopping = true;
    status("Stopping…");
    try { await api(`${job.path}?library=${library}`, { method: "DELETE" }); } catch (error) { status(error.message); }
  });
  try {
    for (;;) {
      await wait();
      let reply;
      try { reply = await api(`${job.path}?library=${library}`); failures = 0; }
      catch (error) {
        // The server may be restarting (it answers with a page, not data): wait a little before giving up.
        if (++failures <= 12) { status("Waiting for the server…"); continue; }
        status("Lost contact with the server. What was done is kept; start it again to continue."); return;
      }
      if (reply.status === "ready" || reply.status === "failed") {
        await reload();
        status(reply.error || reply.message || job.done); // after the reload, which writes its own status
        return;
      }
      if (reply.status === "canceled" || reply.status === "idle") {
        await reload();
        status(stopping || reply.status === "canceled" ? `Stopped${reply.total ? ` after ${reply.current || 0} of ${reply.total}` : ""}. What was done is kept.` : "");
        return;
      }
      if (!stopping) status(`${label}… ${reply.current || 0} of ${reply.total || "?"}${job.detail && reply.message ? ` · ${reply.message}` : ""}`);
    }
  } finally { stop?.hide(); }
}

// resumeJobs re-attaches to anything still running on the server for this library.
export async function resumeJobs(context) {
  const library = encodeURIComponent(context.state.library);
  for (const job of Object.values(JOBS)) {
    let reply;
    try { reply = await context.api(`${job.path}?library=${library}`); } catch { continue; }
    if (reply.status === "running" || reply.status === "queued") { watchJob(context, job, { start: false }); return; }
  }
}

const countWordsAndPages = (context) => watchJob(context, JOBS.count);
const checkQuality = (context) => watchJob(context, JOBS.check);
const findISBNs = (context) => watchJob(context, JOBS.isbn);
const askHermesISBN = (context) => watchJob(context, JOBS.isbnAsk);
const findMatches = (context) => watchJob(context, JOBS.matches);
const askHermesMatches = (context) => watchJob(context, JOBS.matchAsk);

// requiresHermes tools ask a helper model something about the book; mountTools leaves them out entirely when Hermes
// is off, instead of showing a button that can only fail.
export const TOOLS = [
  { id: "matches", label: "Find Hardcover matches", hint: "Looks books up by ISBN, else by title; slow (about a minute per sixty books) and safe to stop. Nothing changes until you choose Use clear matches", run: findMatches },
  { id: "match-ask", label: "Ask Hermes about Hardcover matches", requiresHermes: true, hint: "For books where Find Hardcover matches was unsure: Hermes may only pick one of Hardcover's candidates, and only when confident. Goes through every eligible book one after another (Stop keeps what is done, and running it again continues); nothing changes until you choose Use clear matches", run: askHermesMatches },
  { id: "isbn", label: "Find ISBNs", hint: "Reads each book without an ISBN for the one printed in it; nothing changes until you choose Use found ISBN", run: findISBNs },
  { id: "isbn-ask", label: "Ask Hermes about ISBNs", requiresHermes: true, hint: "For books where Find ISBNs was unsure or found none: Hermes reads the words around each number and may only pick one that is really in the book (checked again here). Goes through every eligible book one after another (Stop keeps what is done, and running it again continues); nothing changes until you choose Use found ISBN", run: askHermesISBN },
  { id: "check", label: "Check quality", hint: "Finds missing covers and files that will not open; shown under the filters", run: checkQuality },
  { id: "count", label: "Count words and pages", hint: "Reads each book once; pages are estimated at 250 words", run: countWordsAndPages },
];

// hermesEnabled may be a boolean or a promise of one (so a caller can pass an in-flight /api/modules check without
// awaiting it first); it defaults to false so a tool that only fails without Hermes never flashes into view.
export async function mountTools({ list, details, context, hermesEnabled = false }) {
  const enabled = await hermesEnabled;
  const tools = TOOLS.filter((tool) => !tool.requiresHermes || enabled);
  list.innerHTML = tools.map((tool) => `<button type="button" class="ed-tool" data-tool="${tool.id}"><strong>${tool.label}</strong><small>${tool.hint}</small></button>`).join("");
  list.addEventListener("click", (event) => {
    const button = event.target.closest("[data-tool]");
    if (!button) return;
    details.open = false;
    tools.find((tool) => tool.id === button.dataset.tool)?.run(context());
  });
}
