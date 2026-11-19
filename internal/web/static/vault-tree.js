import {vaultPageEntries} from './vault-pages.js?v=1';

export function createVaultTree({ escapeHTML }) {
  const naturalName = (left, right) =>
    left.localeCompare(right, undefined, {
      numeric: true,
      sensitivity: "base",
    });

  function branch() {
    return { folders: new Map(), pages: [] };
  }

  function build(items) {
    const root = branch();
    for (const item of vaultPageEntries(items)) {
      const parts = String(item.path || "")
        .split("/")
        .filter(Boolean);
      let current = root;
      const folderParts = item.kind === "folder" ? parts : parts.slice(0, -1);
      for (const name of folderParts) {
        if (!current.folders.has(name)) current.folders.set(name, branch());
        current = current.folders.get(name);
      }
      if (item.kind !== "folder") current.pages.push(item);
    }
    return root;
  }

  function holds(node, id) {
    if (!id) return false;
    if (node.pages.some(item => item.id === id)) return true;
    for (const child of node.folders.values()) if (holds(child, id)) return true;
    return false;
  }

  // Folders that start closed are drawn empty and filled in when first opened, so a
  // vault with thousands of pages does not put every page in the document at once.
  let lazy = new Map();
  let smallVault = true;
  const OPEN_ROOT_LIMIT = 400;

  function renderBranch(node, depth = 0, activeID = "", prefix = "") {
    const folders = [...node.folders.entries()].sort(([a], [b]) =>
      naturalName(a, b),
    );
    const pages = [...node.pages].sort((a, b) =>
      naturalName(a.title, b.title),
    );
    // Obsidian's standard File Explorer groups folders before loose files.
    // Sorting remains natural within both groups (for example Note 2 before
    // Note 10), matching the default name ordering users see in a vault.
    return `${folders
      .map(
        ([name, child]) => {
          const open = (depth === 0 && smallVault) || holds(child, activeID);
          if (!open) lazy.set(prefix + name, { node: child, depth: depth + 1 });
          return `<details class="vault-folder" data-vault-folder="${escapeHTML(prefix + name)}" ${open ? "open" : ""}><summary style="--vault-indent:${depth * 14}px"><svg class="vault-chevron" viewBox="0 0 24 24" aria-hidden="true"><path d="m9 6 6 6-6 6"/></svg><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h7l2 2h9v11H3z"/></svg><span>${escapeHTML(name)}</span><small>${countPages(child).toLocaleString()}</small><button type="button" class="vault-folder-rename" data-vault-rename="${escapeHTML(prefix + name)}" aria-label="Rename ${escapeHTML(name)}" title="Rename folder"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 20 1-4L16 5l3 3L8 19zM14 7l3 3"/></svg></button></summary><div>${open ? renderBranch(child, depth + 1, activeID, `${prefix}${name}/`) : ""}</div></details>`;
        },
      )
      .join("")}${pages
      .map(
        (item) =>
          `<button class="vault-page" draggable="true" data-document="${escapeHTML(item.id)}" style="--vault-indent:${depth * 14}px"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 3h7l4 4v14H7zM14 3v5h5"/></svg><span>${escapeHTML(item.title)}</span></button>`,
      )
      .join("")}`;
  }

  function countPages(node) {
    let total = node.pages.length;
    for (const child of node.folders.values()) total += countPages(child);
    return total;
  }

  /** Draws the contents of a folder that was rendered closed; call when it is opened. */
  function expand(details) {
    const path = details.dataset.vaultFolder;
    const entry = lazy.get(path);
    const box = details.querySelector(":scope > div");
    if (!entry || !box || box.childElementCount) return;
    lazy.delete(path);
    box.innerHTML = renderBranch(entry.node, entry.depth, "", `${path}/`);
  }

  const render = (items, activeID = "") => {
    lazy = new Map();
    const tree = build(items);
    smallVault = countPages(tree) <= OPEN_ROOT_LIMIT;
    return `<nav class="vault-tree" aria-label="Vault pages">${renderBranch(tree, 0, activeID).replace(`<button class="vault-page" draggable="true" data-document="${escapeHTML(activeID)}"`, `<button class="vault-page active" aria-current="page" draggable="true" data-document="${escapeHTML(activeID)}"`)}</nav>`;
  };
  render.expand = expand;
  return render;
}
