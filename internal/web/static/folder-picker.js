// Server-side folder chooser. Browsing is limited by the server to configured
// locations, so the dialog only ever shows folders a library could use.
export function createFolderPicker({ document, requestJSON }) {
  let dialog;
  // Why the last pick returned nothing: "cancel" (the reader closed it) or
  // "empty" (the server has no browsable locations at all). A caller that adds a
  // library needs the difference — one means "do nothing", the other must fall
  // back to a path the reader can type.
  let lastOutcome = "cancel";

  function build() {
    dialog = document.createElement("dialog");
    dialog.className = "folder-picker";
    dialog.setAttribute("aria-labelledby", "folder-picker-title");
    dialog.innerHTML = `
      <header><h2 id="folder-picker-title">Choose a folder</h2></header>
      <p class="folder-picker-path" data-path></p>
      <p class="folder-picker-status" role="status" data-status></p>
      <ul class="folder-picker-list" data-list></ul>
      <form class="folder-picker-create" data-create hidden>
        <label>New folder here<input type="text" data-create-name maxlength="255" placeholder="Name" autocomplete="off" /></label>
        <button type="submit" data-create-confirm>Create</button>
        <button type="button" data-create-cancel>Cancel</button>
      </form>
      <div class="dialog-actions">
        <button type="button" data-new-folder>New folder…</button>
        <button type="button" data-cancel>Cancel</button>
        <button type="button" class="primary" data-choose>Use this folder</button>
      </div>`;
    document.body.append(dialog);
    return dialog;
  }

  function pick(initialPath = "") {
    const host = dialog || build();
    const list = host.querySelector("[data-list]");
    const pathLabel = host.querySelector("[data-path]");
    const status = host.querySelector("[data-status]");
    const choose = host.querySelector("[data-choose]");
    const newFolder = host.querySelector("[data-new-folder]");
    const create = host.querySelector("[data-create]");
    const createName = host.querySelector("[data-create-name]");
    let current = "";
    let token = 0;

    function closeCreate() {
      create.hidden = true;
      createName.value = "";
    }

    async function open(path) {
      const mine = ++token;
      closeCreate();
      status.textContent = "Loading…";
      let listing;
      try {
        listing = (
          await requestJSON(`/api/folders?path=${encodeURIComponent(path)}`)
        ).data;
      } catch (error) {
        if (path) return open("");
        status.textContent = `Could not list folders: ${error.message}`;
        return;
      }
      if (mine !== token) return;
      current = listing.path || "";
      if (!current && !listing.parent && !listing.directories.length)
        lastOutcome = "empty";
      pathLabel.textContent = current || "Locations";
      choose.disabled = !current;
      // A new folder needs somewhere to go inside; the root list is a choice
      // between locations, not a folder of its own.
      newFolder.disabled = !current;
      status.textContent = listing.directories.length
        ? listing.truncated
          ? "Showing the first folders only. Type a path to go deeper."
          : ""
        : current
          ? "No sub-folders here."
          : "No browsable locations are configured on the server.";
      list.replaceChildren(
        ...(listing.parent
          ? [folderRow("↑ Up one level", listing.parent, "up")]
          : current
            ? [folderRow("↑ All locations", "", "up")]
            : []),
        ...listing.directories.map((folder) =>
          folderRow(folder.name, folder.path),
        ),
      );
      list.querySelector("button")?.focus();
    }

    function folderRow(label, path, kind = "") {
      const item = document.createElement("li");
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = label;
      if (kind) button.className = kind;
      button.onclick = () => open(path);
      item.append(button);
      return item;
    }

    async function createFolder(event) {
      event.preventDefault();
      const name = createName.value.trim();
      if (!name) return;
      const confirmButton = host.querySelector("[data-create-confirm]");
      confirmButton.disabled = true;
      status.textContent = "Creating…";
      try {
        const folder = (
          await requestJSON("/api/folders", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ path: current, name }),
          })
        ).data;
        await open(current);
        status.textContent = `“${folder.name}” created.`;
      } catch (error) {
        status.textContent = error.message;
      } finally {
        confirmButton.disabled = false;
      }
    }

    lastOutcome = "cancel";
    return new Promise((resolve) => {
      const finish = (value) => {
        host.close();
        resolve(value);
      };
      host.querySelector("[data-cancel]").onclick = () => finish(null);
      newFolder.onclick = () => {
        create.hidden = false;
        createName.focus();
      };
      host.querySelector("[data-create-cancel]").onclick = () => closeCreate();
      create.onsubmit = createFolder;
      choose.onclick = () => {
        lastOutcome = "chosen";
        finish(current || null);
      };
      host.onclose = () => resolve(null);
      host.showModal();
      open(initialPath.trim());
    });
  }

  return {
    pick,
    outcome: () => lastOutcome,
  };
}

// Adds a Browse button after a path input. The input keeps its own validation.
export function attachFolderBrowse(input, picker) {
  if (input.dataset.browseBound) return;
  input.dataset.browseBound = "true";
  const button = input.ownerDocument.createElement("button");
  button.type = "button";
  button.className = "browse-folder";
  button.textContent = "Browse…";
  button.setAttribute("aria-label", "Browse server folders");
  button.onclick = async () => {
    const path = await picker.pick(input.value);
    if (!path) return;
    input.value = path;
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.focus();
  };
  input.insertAdjacentElement("afterend", button);
  input.parentElement.classList.add("path-field");
}
