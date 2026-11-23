// features/notes/api/vault.ts — Notes reads and writes the **vault library**, not `/api/wiki`.
//
// Corrected after Kevin reported "I don't see my obsidian vault at all": the first cut of this screen talked to
// `/api/wiki` (internal/web/wiki_http.go), a separate server-side wiki store that starts empty and never holds
// anything from Obsidian. His real vault is a *library* of kind `vault` — visible in Settings → Libraries as
// `/media/resilio/data/obsidian/notes`, 48 items — served by a completely different API. Two different things
// with similar names; the screen was pointed at the wrong one.
//
// The vault API (confirmed live against the dev server, not inferred):
//   GET  /api/libraries                              → every library; the vault one has kind === "vault"
//   GET  /api/documents?library=<id>&tree=1&sort=name → folders as {kind:"folder", path} and files as
//                                                       {id,path,title,format,revision,text_url,...}
//   GET  <text_url>                                   → {content: "<raw markdown>"}
//   POST /api/vaults/<lib>/pages    {path, content}   → create; path must end .md
//   PUT  /api/vaults/<lib>/pages/<itemId> {revision, content} → the page's new revision string
//   DELETE /api/vaults/<lib>/pages/<itemId> {revision} → moves the file into the vault's own hidden .trash
//   POST /api/vaults/<lib>/folders  {path}            → create a folder
// Every mutation answers `committed`/`index_refreshed` too: the write can succeed while the library re-scan
// fails, which the server reports as a 500 with `committed: true` — a save that landed but needs a refresh, not
// a save that failed. `messageFromError` surfaces that message verbatim rather than calling it a failure.
//
// Revisions are opaque strings here (file content hashes), not the wiki's integer counter — the editor must send
// back exactly what the server last returned, which is why `savePage` takes and returns one.
import { fetchLibraries, type LibraryInfo } from "../../library/api/libraries";
import { request } from "../../../lib/http";

export interface VaultFolder {
  kind: "folder";
  path: string;
}

/** Any file in the vault — a markdown note, or an asset an image embed points at. */
export interface VaultFile {
  id: string;
  library_id: string;
  path: string;
  title: string;
  format: string;
  size: number;
  modified: string;
  revision: string;
  /** Authenticated same-origin route that serves the file. This is what note images resolve to. */
  open_url?: string;
  download_url?: string;
  /** Only markdown gets one — the server mints it for editable text only. */
  text_url?: string;
}

/** A file the editor can open. */
export type VaultPage = VaultFile & { text_url: string };

export type VaultEntry = VaultFolder | VaultFile;

export function isVaultPage(entry: VaultEntry): entry is VaultPage {
  return !("kind" in entry) && typeof (entry as VaultFile).text_url === "string";
}

/** The vault libraries this server has. Normally one; the API allows several. */
export async function listVaultLibraries(): Promise<LibraryInfo[]> {
  const libraries = await fetchLibraries();
  return libraries.filter((library) => library.kind === "vault");
}

/** Every folder, note **and asset** in one vault. `tree=1` is what makes the server include folder rows at all. */
export async function listEntries(library: string): Promise<VaultEntry[]> {
  const params = new URLSearchParams({ library, tree: "1", sort: "name", limit: "2000" });
  const entries = await request<VaultEntry[]>(`/api/documents?${params}`);
  // Assets are kept, not filtered out: a note's `![[image.png]]` can only render if the image's own `open_url` is
  // in hand. The list/tree UI decides what to show with `isVaultPage`; this function returns the vault's truth.
  return entries.filter((entry) => ("kind" in entry && entry.kind === "folder") || typeof (entry as VaultFile).id === "string");
}

export interface VaultPageText {
  content: string;
  /** The **content** hash the writer checks — NOT the list item's `revision`, which fingerprints
   *  path+size+mtime instead (internal/document/item.go) and can never satisfy a save. Sending the list
   *  revision made every save fail with "This page changed in Obsidian or another browser", reproduced live. */
  editRevision: string;
  writable: boolean;
}

/**
 * Reads one note. Three fields matter, and only the first is obvious:
 *  - `content`       the markdown body
 *  - `edit_revision` the content hash a save must echo back (see above)
 *  - `writable`      whether *this* account may write to the vault at all — the server answers it per user, so a
 *                    read-only share still opens notes instead of offering a Save that would 403.
 */
export async function readPage(textUrl: string): Promise<VaultPageText> {
  const result = await request<{ content: string; edit_revision?: string; writable?: boolean }>(textUrl);
  return {
    content: result.content,
    editRevision: result.edit_revision ?? "",
    writable: result.writable !== false,
  };
}

/** `path` is vault-relative and must end in `.md` (the server rejects anything else). */
export async function createPage(library: string, path: string, content: string): Promise<void> {
  await request<{ ok: boolean }>(`/api/vaults/${encodeURIComponent(library)}/pages`, {
    method: "POST",
    body: { path, content },
  });
}

/** Saves `content` and returns the page's new revision, which the next save must send. */
export async function savePage(
  library: string,
  itemId: string,
  content: string,
  revision: string,
): Promise<string> {
  const result = await request<{ revision: string }>(
    `/api/vaults/${encodeURIComponent(library)}/pages/${encodeURIComponent(itemId)}`,
    { method: "PUT", body: { revision, content } },
  );
  return result.revision;
}

/** Moves the file into the vault's own `.trash` — recoverable in Obsidian, not destroyed. */
export async function trashPage(library: string, itemId: string, revision: string): Promise<void> {
  await request<{ ok: boolean }>(`/api/vaults/${encodeURIComponent(library)}/pages/${encodeURIComponent(itemId)}`, {
    method: "DELETE",
    body: { revision },
  });
}

export async function createFolder(library: string, path: string): Promise<void> {
  await request<{ ok: boolean }>(`/api/vaults/${encodeURIComponent(library)}/folders`, {
    method: "POST",
    body: { path },
  });
}
