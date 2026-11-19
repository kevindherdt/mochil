// Collections in the editor: add the picked books to one, take them out, make a new one from them, and show only one
// collection's books. Collections that fill themselves from a search cannot be changed by hand, so they are listed but
// not offered when adding.

const escapeHTML = (value) => String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const NEW = "__new__";

export const collectionBody = (collections) => {
  const hand = collections.filter((c) => !c.smart);
  const options = hand.map((c) => `<option value="${escapeHTML(c.id)}">${escapeHTML(c.name)} (${c.count})</option>`).join("");
  return `<fieldset class="ed-choice"><label><input type="radio" name="mode" value="add" checked> Add to</label><label><input type="radio" name="mode" value="remove"${hand.length ? "" : " disabled"}> Take out of</label></fieldset>` +
    `<label>Collection<select name="collection">${options}<option value="${NEW}">New collection…</option></select></label>` +
    `<label>Name of the new collection<input type="text" name="name" maxlength="80" autocomplete="off" placeholder="Only for a new collection"></label>` +
    (collections.some((c) => c.smart) ? "<p>Collections that fill themselves from a search are not listed; change their search in the library page.</p>" : "");
};

// collectionRequest turns the dialog's answers into the request, or says what is missing.
export function collectionRequest(form, library, ids) {
  const mode = form.get("mode") === "remove" ? "remove" : "add";
  const chosen = form.get("collection");
  if (chosen === NEW || !chosen) {
    const name = String(form.get("name") || "").trim();
    if (mode === "remove") return { problem: "Pick the collection to take the books out of." };
    if (!name) return { problem: "Give the new collection a name." };
    return { body: { library, name, ids } };
  }
  return { body: { library, collection: chosen, ids, remove: mode === "remove" } };
}

export async function collectionsDialog({ state, api, ask, status, reload }) {
  const ids = [...state.selected];
  if (!ids.length) return;
  let result = null;
  const done = await ask(`Collections for ${ids.length} ${ids.length === 1 ? "book" : "books"}`, collectionBody(state.collections), "Apply", async (form) => {
    const { body, problem } = collectionRequest(form, state.library, ids);
    if (problem) return problem;
    try { result = await api("/api/metadata/editor/collections", { method: "POST", body }); } catch (error) { return error.message; }
    return "";
  });
  if (!done) return;
  await reload();
  status(result.changed ? `Done: ${result.changed} ${result.changed === 1 ? "book" : "books"} changed.` : "Nothing changed: the books were already that way.");
}

// fillCollectionFilter fills the "show one collection" list, and says whether it should show at all.
export function fillCollectionFilter(select, collections, chosen) {
  select.parentElement.hidden = collections.length === 0;
  select.innerHTML = `<option value="">All books</option>` + collections.map((c) => `<option value="${escapeHTML(c.id)}"${c.id === chosen ? " selected" : ""}>${escapeHTML(c.name)}${c.smart ? " (smart)" : ""} · ${c.count}</option>`).join("");
}
