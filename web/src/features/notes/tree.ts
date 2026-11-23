// features/notes/tree.ts — the vault tree's pure logic, kept out of NoteTree.tsx so that file exports components
// only (eslint's react-refresh/only-export-components rule: a module mixing components and helpers can't be
// hot-reloaded safely). buildTree is ordinary data shaping, so it is also the part worth unit-testing later.
import type { VaultPage } from "./api/vault";

export interface VaultTreeNode {
  /** Folder name, "" for the vault root. */
  name: string;
  /** Vault-relative path, "" for the root. */
  path: string;
  folders: VaultTreeNode[];
  notes: VaultPage[];
}

const collate = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });

/** The file's name without `.md` — what Obsidian calls the note's title. */
export function noteName(page: VaultPage): string {
  const base = page.path.slice(page.path.lastIndexOf("/") + 1);
  return base.replace(/\.md$/i, "");
}

/** Builds the tree from the flat `tree=1` listing. Folder rows carry their full path, so no guessing is needed. */
export function buildTree(folderPaths: string[], notes: VaultPage[]): VaultTreeNode {
  const root: VaultTreeNode = { name: "", path: "", folders: [], notes: [] };
  const byPath = new Map<string, VaultTreeNode>([["", root]]);

  const ensure = (path: string): VaultTreeNode => {
    const existing = byPath.get(path);
    if (existing !== undefined) return existing;
    const cut = path.lastIndexOf("/");
    const parentPath = cut === -1 ? "" : path.slice(0, cut);
    const node: VaultTreeNode = { name: cut === -1 ? path : path.slice(cut + 1), path, folders: [], notes: [] };
    byPath.set(path, node);
    ensure(parentPath).folders.push(node);
    return node;
  };

  for (const folder of folderPaths) {
    if (folder !== "") ensure(folder);
  }
  for (const note of notes) {
    const cut = note.path.lastIndexOf("/");
    ensure(cut === -1 ? "" : note.path.slice(0, cut)).notes.push(note);
  }

  const sort = (node: VaultTreeNode) => {
    node.folders.sort((a, b) => collate.compare(a.name, b.name));
    node.notes.sort((a, b) => collate.compare(noteName(a), noteName(b)));
    node.folders.forEach(sort);
  };
  sort(root);
  return root;
}

/** Notes in this folder and every folder under it — the count a collapsed row shows. */
export function countNotes(node: VaultTreeNode): number {
  return node.notes.length + node.folders.reduce((total, child) => total + countNotes(child), 0);
}

