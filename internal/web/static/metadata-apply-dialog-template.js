export const metadataApplyDialogHTML = `
  <form method="dialog">
    <header>
      <h2>Apply metadata repairs?</h2>
      <button value="cancel" aria-label="Close"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17" /></svg></button>
    </header>
    <p>This writes the reviewed metadata through the shelf’s supported adapter. Restore copies are kept and the entire job rolls back if one book fails.</p>
    <label>Type <strong id="metadata-confirmation-label"></strong><input id="metadata-confirmation" autocomplete="off" spellcheck="false" /></label>
    <button type="button" class="danger" id="confirm-metadata-apply" disabled>Apply reviewed changes</button>
    <p id="metadata-apply-status" role="status"></p>
  </form>`;

export function mountMetadataApplyDialog(document) {
  if (document.querySelector("#metadata-apply-dialog")) return;
  const dialog = document.createElement("dialog");
  dialog.className = "apply-dialog";
  dialog.id = "metadata-apply-dialog";
  dialog.innerHTML = metadataApplyDialogHTML;
  document.body.append(dialog);
}
