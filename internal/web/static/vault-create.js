import { vaultMutationCommitted } from './vault-mutation-error.js';

export function mountVaultCreate({ host, type, library, requestJSON, onCreated }) {
  host.querySelector(".vault-create")?.remove();
  const form = host.ownerDocument.createElement("form");
  form.className = "vault-create";
  form.innerHTML = `<label><span>${type === "folder" ? "New folder" : "New page"}</span><input required maxlength="240" autocomplete="off" placeholder="${type === "folder" ? "Projects" : "Projects/New page"}" /></label><div><button type="button">Cancel</button><button type="submit">Create</button></div><small role="status"></small>`;
  host.prepend(form);
  const input = form.querySelector("input");
  const status = form.querySelector("small");
  input.focus();
  form.querySelector('button[type="button"]').onclick = () => form.remove();
  form.onsubmit = async (event) => {
    event.preventDefault();
    const submit = form.querySelector('button[type="submit"]');
    if (submit.disabled || !form.isConnected) return;
    let path = input.value.trim();
    if (type === "page" && !/\.md$/i.test(path)) path += ".md";
    submit.disabled = true;
    input.readOnly = true;
    status.textContent = "Creating…";
    try {
      await requestJSON(
        `/api/vaults/${encodeURIComponent(library)}/${type === "folder" ? "folders" : "pages"}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path, content: "" }),
        },
      );
      if (!form.isConnected) return;
      form.remove();
      await onCreated();
    } catch (error) {
      if (!form.isConnected) return;
      status.textContent = error.message;
      if (vaultMutationCommitted(error)) {
        form.querySelector('button[type="button"]').textContent = "Close";
        return;
      }
      submit.disabled = false;
      input.readOnly = false;
    }
  };
}
