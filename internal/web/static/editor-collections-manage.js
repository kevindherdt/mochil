// The editor's "Manage collections" dialog: rename or delete a collection, and change the search of a smart one.

const escapeHTML = (value) => String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const RULE_TEXT = [["query", "Search words"], ["author", "Author"], ["series", "Series"], ["tag", "Tag"], ["format", "Format (epub, pdf…)"]];
const STATUSES = [["", "Any reading state"], ["unread", "Unread"], ["reading", "Reading"], ["finished", "Finished"]];

export const manageBody = (collections) => collections.map((c) => {
  const rule = c.rule || {};
  const search = c.rule ? `<details class="ed-rule"><summary>Search</summary>` +
    RULE_TEXT.map(([key, label]) => `<label>${label}<input type="text" name="rule:${c.id}:${key}" value="${escapeHTML(rule[key])}" autocomplete="off"></label>`).join("") +
    `<label>Reading state<select name="rule:${c.id}:status">${STATUSES.map(([value, label]) => `<option value="${value}"${rule.status === value || (!rule.status && !value) ? " selected" : ""}>${label}</option>`).join("")}</select></label>` +
    `<label>Added in the last (days)<input type="number" min="0" name="rule:${c.id}:added_days" value="${rule.added_days || ""}"></label>` +
    `<label class="ed-check-line"><input type="checkbox" name="rule:${c.id}:missing_volumes"${rule.missing_volumes ? " checked" : ""}> Only series with a volume missing</label></details>` : "";
  return `<div class="ed-manage-row"><input type="text" name="name:${c.id}" value="${escapeHTML(c.name)}" maxlength="80" aria-label="Name"><small>${c.rule ? "smart" : "picked books"} · ${c.count}</small>` +
    `<label class="ed-delete"><input type="checkbox" name="delete:${c.id}"> Delete</label>${search}</div>`;
}).join("") + '<p class="ed-hint">Deleting a collection does not delete its books.</p>';

const ruleFrom = (form, id) => ({
  query: String(form.get(`rule:${id}:query`) || "").trim(), author: String(form.get(`rule:${id}:author`) || "").trim(), series: String(form.get(`rule:${id}:series`) || "").trim(),
  tag: String(form.get(`rule:${id}:tag`) || "").trim(), format: String(form.get(`rule:${id}:format`) || "").trim(), status: String(form.get(`rule:${id}:status`) || ""),
  added_days: Math.max(0, Math.floor(Number(form.get(`rule:${id}:added_days`)) || 0)), missing_volumes: form.get(`rule:${id}:missing_volumes`) !== null,
});
const sameRule = (a, b) => ["query", "author", "series", "tag", "format", "status"].every((k) => (a[k] || "") === (b[k] || "")) && (a.added_days || 0) === (b.added_days || 0) && Boolean(a.missing_volumes) === Boolean(b.missing_volumes);

// changesFrom lists the requests that make the collections what the dialog now says: deletes, and patches for a new name
// or a new search. Nothing is listed for what was left alone.
export function changesFrom(form, collections) {
  const requests = [];
  for (const c of collections) {
    if (form.get(`delete:${c.id}`) !== null) { requests.push({ method: "DELETE", id: c.id }); continue; }
    const body = {};
    const name = String(form.get(`name:${c.id}`) || "").trim();
    if (name && name !== c.name) body.name = name;
    if (c.rule) { const rule = ruleFrom(form, c.id); if (!sameRule(rule, c.rule)) body.rule = rule; }
    if (Object.keys(body).length) requests.push({ method: "PATCH", id: c.id, body });
  }
  return requests;
}

// readingFailure is the one sentence the click reports when the list cannot be
// read. A failed request is not an empty library, and a reply that is not JSON (a
// sign-in page, a proxy error) used to throw out of the click handler and leave
// the button looking dead.
const LIST_FAILURE = "Could not read the collections. Reload this page and try again.";

export async function manageCollections({ state, ask, status, reload }) {
  let collections;
  try {
    const response = await fetch(`/api/collections?library=${encodeURIComponent(state.library)}`, { credentials: "same-origin" });
    const reply = await response.json().catch(() => null);
    if (!response.ok || !Array.isArray(reply)) {
      status(reply?.error ? `Could not read the collections: ${reply.error}` : LIST_FAILURE, "problem");
      return;
    }
    collections = reply;
  } catch {
    status(LIST_FAILURE, "problem");
    return;
  }
  if (!collections.length) { status("There are no collections in this library yet."); return; }
  let count = 0;
  const done = await ask("Manage collections", manageBody(collections), "Save", async (form) => {
    const requests = changesFrom(form, collections);
    // A save that never reaches the server keeps the dialog open with the reason:
    // the click used to reject silently and the button then did nothing at all.
    try {
      for (const request of requests) {
        const response = await fetch(`/api/collections/${encodeURIComponent(request.id)}`, {
          method: request.method, credentials: "same-origin", ...(request.body ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify(request.body) } : {}),
        });
        if (!response.ok) return (await response.text()).trim() || `Could not save (${response.status}).`;
        count++;
      }
    } catch (error) {
      return `Could not save: ${error.message || "the server could not be reached"}.`;
    }
    return "";
  });
  if (!done) return;
  await reload();
  status(count ? `Saved changes to ${count} ${count === 1 ? "collection" : "collections"}.` : "Nothing was changed.");
}
