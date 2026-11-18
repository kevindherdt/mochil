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
      <div class="dialog-actions">
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
    let current = "";
    let token = 0;

    async function open(path) {
      const mine = ++token;
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

    lastOutcome = "cancel";
    return new Promise((resolve) => {
      const finish = (value) => {
        host.close();
        resolve(value);
      };
      host.querySelector("[data-cancel]").onclick = () => finish(null);
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
