import { ruleFieldsHTML } from "./smart-collection-form.js?v=2";

export const collectionDialogHTML = `
  <form method="dialog">
    <header>
      <h2 id="collection-dialog-title">Save to collection</h2>
      <button value="cancel" aria-label="Close"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17" /></svg></button>
    </header>
    <div id="collection-choices"></div>
    ${ruleFieldsHTML}
    <div class="collection-create">
      <input id="collection-name" maxlength="80" placeholder="New collection name" aria-label="New collection name" />
      <button type="button" id="create-collection">Create</button>
    </div>
    <p id="collection-dialog-status" role="status"></p>
  </form>`;

export function mountCollectionDialog(document) {
  if (document.querySelector("#collection-dialog")) return;
  const dialog = document.createElement("dialog");
  dialog.className = "collection-dialog";
  dialog.id = "collection-dialog";
  // A dialog needs a name of its own; the heading is the only visible one.
  dialog.setAttribute("aria-labelledby", "collection-dialog-title");
  dialog.innerHTML = collectionDialogHTML;
  document.body.append(dialog);
}
