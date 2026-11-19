import { vaultMutationCommitted } from './vault-mutation-error.js';

const tools = [
  ["Bold", "**", "**"],
  ["Italic", "_", "_"],
  ["Heading", "## ", ""],
  ["Link", "[", "](https://)"],
  ["Wiki link", "[[", "]]"],
  ["Inline code", "`", "`"],
  ["Quote", "> ", ""],
  ["Bullet list", "- ", ""],
];

export function mountVaultEditor({
  pane,
  item,
  content,
  revision,
  escapeHTML,
  requestJSON,
  onCancel,
  onSaved,
}) {
  pane.innerHTML = `<section class="vault-edit" aria-label="Edit ${escapeHTML(item.title)}">
    <header><div><h1>${escapeHTML(item.title)}</h1><p>${escapeHTML(item.path)}</p></div><span class="vault-edit-state" role="status">Not saved</span></header>
    <div class="vault-format" role="toolbar" aria-label="Markdown formatting">${tools.map(([label], index) => `<button type="button" data-vault-format="${index}" title="${label}" aria-label="${label}">${escapeHTML(label)}</button>`).join("")}</div>
    <textarea class="vault-editor" spellcheck="true" aria-label="Markdown source">${escapeHTML(content)}</textarea>
    <footer><button type="button" data-vault-cancel>Cancel</button><button type="button" class="primary" data-vault-save>Save page</button></footer>
  </section>`;
  const editor = pane.querySelector(".vault-editor");
  const state = pane.querySelector(".vault-edit-state");
  const save = pane.querySelector("[data-vault-save]");
  editor.focus();
  editor.addEventListener("input", () => {
    state.textContent = "Unsaved changes";
  });
  pane.querySelector("[data-vault-cancel]").onclick = onCancel;
  pane.querySelector(".vault-format").onclick = (event) => {
    const button = event.target.closest("[data-vault-format]");
    if (!button) return;
    const [label, before, after] = tools[Number(button.dataset.vaultFormat)];
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const selected = editor.value.slice(start, end) || label;
    editor.setRangeText(`${before}${selected}${after}`, start, end, "select");
    editor.focus();
    editor.dispatchEvent(new Event("input", { bubbles: true }));
  };
  save.onclick = async () => {
    if (save.disabled || !editor.isConnected) return;
    save.disabled = true;
    editor.readOnly = true;
    const formatButtons = pane.querySelectorAll("[data-vault-format]");
    formatButtons.forEach(button => { button.disabled = true; });
    state.textContent = "Saving…";
    let committed = false;
    try {
      await requestJSON(
        `/api/vaults/${encodeURIComponent(item.library_id)}/pages/${encodeURIComponent(item.id)}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ revision, content: editor.value }),
        },
      );
      committed = true;
      if (!editor.isConnected) return;
      state.textContent = "Saved";
      await onSaved();
    } catch (error) {
      if (!editor.isConnected) return;
      state.textContent = error.message;
      if (committed || vaultMutationCommitted(error)) {
        state.textContent = "Saved. Refresh the library before editing again; do not repeat this save.";
        pane.querySelector("[data-vault-cancel]").textContent = "Close";
        return;
      }
      save.disabled = false;
      editor.readOnly = false;
      formatButtons.forEach(button => { button.disabled = false; });
    }
  };
}
