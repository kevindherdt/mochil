// The editor's "Hardcover sync…" dialog: leave the picked books out of the Hardcover sync, or put them back in.

export const syncBody = (count) => `<fieldset class="ed-choice ed-choice-column"><label><input type="radio" name="mode" value="out" checked> Leave out of the sync</label><label><input type="radio" name="mode" value="in"> Put back in the sync</label></fieldset>` +
  `<p>${count === 1 ? "This book's reading is" : `These ${count} books' reading is`} sent to Hardcover only while ${count === 1 ? "it is" : "they are"} in the sync. A book that is left out keeps its place on Hardcover; nothing is deleted there.</p>`;

export const syncRequest = (form, library, ids) => ({ library, ids, ignore: form.get("mode") !== "in" });

export const syncMessage = (changed, ignore) => changed
  ? `${changed} ${changed === 1 ? "book" : "books"} ${ignore ? "left out of" : "put back in"} the Hardcover sync.`
  : `Nothing changed: the books were already ${ignore ? "left out" : "in the sync"}.`;

export async function hardcoverSyncDialog({ state, api, ask, status, reload }) {
  const ids = [...state.selected];
  if (!ids.length) return;
  let result = null, ignore = true;
  const done = await ask("Hardcover sync", syncBody(ids.length), "Apply", async (form) => {
    const body = syncRequest(form, state.library, ids);
    ignore = body.ignore;
    try { result = await api("/api/metadata/editor/hardcover-sync", { method: "POST", body }); } catch (error) { return error.message; }
    return "";
  });
  if (!done) return;
  await reload();
  status(syncMessage(result.changed, ignore));
}
