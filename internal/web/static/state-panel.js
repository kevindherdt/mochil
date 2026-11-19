// One calm, consistent panel for "nothing here" and "that failed", so a shelf
// is never blank and a failed load always offers a way forward.
const escapeText = (value) =>
  String(value).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]);

export function describeProblem(error, host = globalThis.location?.host || "") {
  const status = Number(error?.status) || 0;
  const message = String(error?.message || "");
  if (status === 401 || status === 403) return "Your session has ended. Sign in again to continue.";
  if (status >= 500) return "The server hit a problem. Try again in a moment.";
  if (status === 404) return "That item is no longer available.";
  if (/failed to fetch|networkerror|load failed|offline/i.test(message)) return `You seem to be offline, or ${host || "the server"} can't be reached from this network. If other sites load, check your VPN, Private DNS or wifi settings.`;
  return "Something went wrong while loading this.";
}

export function statePanelHTML({ title, detail = "", actions = [], tone = "empty" }) {
  const buttons = actions
    .map((action) => `<button type="button" data-state-action="${escapeText(action.id)}"${action.primary ? ' class="primary"' : ""}>${escapeText(action.label)}</button>`)
    .join("");
  return `<section class="state-panel" data-tone="${tone}" role="${tone === "error" ? "alert" : "status"}"><h3>${escapeText(title)}</h3>${detail ? `<p>${escapeText(detail)}</p>` : ""}${buttons ? `<div class="state-actions">${buttons}</div>` : ""}</section>`;
}

export function renderStatePanel(target, spec, handlers = {}) {
  target.innerHTML = statePanelHTML(spec);
  const panel = target.querySelector?.(".state-panel");
  panel?.addEventListener("click", (event) => {
    const button = event.target.closest("[data-state-action]");
    if (button) handlers[button.dataset.stateAction]?.(event);
  });
}

// What an empty list means depends on why it is empty.
export function emptySpec({ noun, query = "", bucket = "" }) {
  const label = query.trim();
  if (label) {
    return { title: `No matches for “${label}”`, detail: `Check the spelling, or try fewer words. Search looks at titles, authors and series.`, actions: [{ id: "clear-search", label: "Clear search", primary: true }] };
  }
  if (bucket) {
    return { title: `Nothing under “${bucket}”`, detail: `No ${noun} start with this letter. Pick another one.`, actions: [] };
  }
  return {
    title: "This shelf is empty",
    detail: `Add books to this library's folder and they will appear after the next scan.`,
    actions: [{ id: "settings", label: "Library settings", primary: true }],
  };
}

export function errorSpec(error, what = "this shelf") {
  return { tone: "error", title: `Couldn't load ${what}`, detail: describeProblem(error), actions: [{ id: "retry", label: "Try again", primary: true }] };
}
