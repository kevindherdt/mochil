// features/notes/screens/NotesScreen.tsx — the real Notes body (plan docs/rewrite-progress.md §2.7).
//
// **This screen was pointed at the wrong backend in its first cut.** It used `/api/wiki` (a server-side wiki
// store that starts empty and holds nothing from Obsidian), which is why Kevin saw "No notes yet" while his vault
// sat there with 48 items — the vault is a *library* of kind `vault`, served by `/api/documents` +
// `/api/vaults/.../pages`. Both were called "wiki" in the code; they are unrelated. `api/vault.ts` carries the
// full endpoint notes.
//
// What it is now: his real Obsidian vault, read and written in place.
//   - every folder and markdown note in the vault, grouped by folder, searchable by path
//   - open → edit → save, with the server's file revision round-tripped (a note edited in Obsidian or on another
//     device while it is open here refuses the save instead of overwriting, and says so)
//   - new note into any folder (existing ones offered, new ones created on save)
//   - delete = move the file into the vault's own hidden `.trash`, recoverable in Obsidian
//
// Still left out, and worth naming rather than pretending otherwise: Obsidian's wikilinks/tags/properties/
// backlinks/graph (the legacy app renders all of these and has `/api/documents/<lib>/backlinks|tags|properties|
// graph` behind it), image attachments in notes, and a rendered markdown preview — this is a plain-text editor,
// which is what Obsidian's own source mode is. Non-markdown files in the vault (images, PDFs) are listed nowhere
// and untouched here: `api/vault.ts`'s `listEntries` filters to entries the server minted a `text_url` for.
import { useMemo, useState } from "react";
import { Page, Stack, Inline, Text, Heading, Button, TextField, TextArea, SearchField, Async, EmptyState } from "../../../ui";
import { useVaultEntries, useVaultLibrary, useNotesMutations, loadPage } from "../useNotes";
import { messageFromError } from "../../settings/api/accounts";
import { NoteTree, NoteSearchResults } from "../NoteTree";
import { buildTree, noteName } from "../tree";
import { Markdown } from "../markdown";
import { isVaultPage, type VaultEntry, type VaultPage } from "../api/vault";

type Selection = { kind: "existing"; page: VaultPage; folder: string; name: string } | { kind: "new"; folder: string } | null;

/** The folder a vault path lives in — "" for a note at the vault root. */
function folderOf(path: string): string {
  const cut = path.lastIndexOf("/");
  return cut === -1 ? "" : path.slice(0, cut);
}

export function NotesScreen() {
  const library = useVaultLibrary();
  const entriesQuery = useVaultEntries(library);
  const mutations = useNotesMutations();
  const [selection, setSelection] = useState<Selection>(null);
  const [content, setContent] = useState("");
  const [revision, setRevision] = useState("");
  const [dirty, setDirty] = useState(false);
  const [search, setSearch] = useState("");
  // The entry list carries metadata only, so opening a note fetches its body — a real gap between click and
  // editor that needs its own state.
  const [openingId, setOpeningId] = useState<string | null>(null);
  const [openError, setOpenError] = useState<string | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [writable, setWritable] = useState(true);
  // Obsidian opens notes in read mode and edits on demand; so does this. Read mode is the default because most
  // visits are reading, and it is the only mode that can render images and wikilinks at all.
  const [editing, setEditing] = useState(false);
  const [newFolder, setNewFolder] = useState("");
  const [newName, setNewName] = useState("");

  const entries: VaultEntry[] = useMemo(() => entriesQuery.data ?? [], [entriesQuery.data]);
  // Assets (images, PDFs) are in the list too — only markdown is a note.
  const pages = useMemo(() => entries.filter(isVaultPage), [entries]);
  const folderPaths = useMemo(
    () => (entriesQuery.data ?? []).flatMap((entry) => ("kind" in entry && entry.kind === "folder" ? [entry.path] : [])),
    [entriesQuery.data],
  );
  const tree = useMemo(() => buildTree(folderPaths, pages), [folderPaths, pages]);
  const folders = useMemo(() => {
    const set = new Set<string>([""]);
    for (const page of pages) {
      const folder = folderOf(page.path);
      set.add(folder);
      // Offer every ancestor too, so a note can be filed in Agent/default without typing the whole path.
      let cut = folder.lastIndexOf("/");
      while (cut !== -1) {
        set.add(folder.slice(0, cut));
        cut = folder.slice(0, cut).lastIndexOf("/");
      }
    }
    return [...set].sort();
  }, [pages]);

  const visible = search.trim() === "" ? pages : pages.filter((page) => page.path.toLowerCase().includes(search.trim().toLowerCase()));

  const openPage = (page: VaultPage) => {
    setOpeningId(page.id);
    setOpenError(null);
    loadPage(page.text_url)
      .then((text) => {
        setSelection({ kind: "existing", page, folder: folderOf(page.path), name: noteName(page) });
        setContent(text.content);
        setRevision(text.editRevision);
        setWritable(text.writable);
        setDirty(false);
        setEditing(false);
        setConfirmingDelete(false);
        mutations.reset();
      })
      .catch((error: unknown) => setOpenError(messageFromError(error)))
      .finally(() => setOpeningId(null));
  };

  const openNew = () => {
    setSelection({ kind: "new", folder: "" });
    setNewFolder("");
    setNewName("");
    setContent("");
    setDirty(false);
    setEditing(true);
    setConfirmingDelete(false);
    mutations.reset();
  };

  const close = () => {
    setSelection(null);
    mutations.reset();
  };

  if (library === undefined) {
    // No vault library configured at all is a real, explainable state, not an error: Settings → Libraries is
    // where one gets added, and pointing there beats an empty screen with no explanation.
    return (
      <Page>
        <Stack gap={16}>
          <Heading size="18">Notes</Heading>
          <EmptyState message="No vault library is set up on this server. Add one in Settings → Libraries — a folder of markdown files, which is what an Obsidian vault is." />
        </Stack>
      </Page>
    );
  }

  if (selection !== null) {
    const isNew = selection.kind === "new";
    const heading = isNew ? "New note" : noteName(selection.page);
    const location = isNew ? (newFolder === "" ? "Vault root" : newFolder) : folderOf(selection.page.path) || "Vault root";
    return (
      <Page>
        <Stack gap={16}>
          {/* Sticky, because a long note scrolls for screens and the Edit button must not scroll away with it. */}
          <div
            style={{
              position: "sticky",
              top: 0,
              zIndex: 2,
              background: "var(--color-bg)",
              paddingBottom: "var(--space-8)",
            }}
          >
          <Inline justify="space-between" align="center">
            <Stack gap={4}>
              <Heading size="18">{heading}</Heading>
              <Text size="12" muted>
                {location}
                {dirty && !isNew ? " · unsaved changes" : ""}
              </Text>
            </Stack>
            <Inline gap={8}>
              {!isNew && writable && (
                <Button variant="secondary" onClick={() => setEditing((current) => !current)}>
                  {editing ? "Done" : "Edit"}
                </Button>
              )}
              <Button variant="secondary" onClick={close}>
                Back
              </Button>
            </Inline>
          </Inline>
          </div>

          {isNew && (
            <>
              <TextField label="Title" value={newName} onChange={(event) => setNewName(event.target.value)} />
              <TextField
                label="Folder (blank for the vault root)"
                value={newFolder}
                onChange={(event) => setNewFolder(event.target.value)}
                list="notes-folders"
              />
              <datalist id="notes-folders">
                {folders.map((folder) => (
                  <option key={folder || "__root__"} value={folder} />
                ))}
              </datalist>
            </>
          )}

          {editing ? (
            <TextArea
              label="Markdown"
              rows={20}
              value={content}
              readOnly={!writable}
              onChange={(event) => {
                setContent(event.target.value);
                setDirty(true);
              }}
            />
          ) : (
            <div>
              {content.trim() === "" ? (
                <Text size="13" muted>
                  This note is empty{writable ? " — press Edit to start writing." : "."}
                </Text>
              ) : (
                <Markdown
                  source={content}
                  context={{
                    noteFolder: folderOf(selection.kind === "existing" ? selection.page.path : ""),
                    entries,
                    onOpenNote: openPage,
                  }}
                />
              )}
            </div>
          )}

          {!writable && (
            <Text size="13" tone="error">
              This vault is read-only for your account, so notes open but cannot be changed here.
            </Text>
          )}

          {mutations.error !== null && (
            <Text size="13" tone="error">
              {messageFromError(mutations.error)}
            </Text>
          )}

          <Inline gap={8}>
            {(editing || isNew) && (
            <Button
              variant="primary"
              disabled={mutations.isPending || !writable || (isNew ? newName.trim() === "" : !dirty)}
              onClick={() => {
                if (selection.kind === "new") {
                  const folder = newFolder.trim().replace(/^\/+|\/+$/g, "");
                  const name = newName.trim().replace(/\.md$/i, "");
                  if (name === "") return;
                  const path = folder === "" ? `${name}.md` : `${folder}/${name}.md`;
                  mutations.create({ library, path, content }, close);
                  return;
                }
                mutations.save(
                  { library, itemId: selection.page.id, content, revision },
                  (next) => {
                    setRevision(next);
                    setDirty(false);
                  },
                );
              }}
            >
              {isNew ? "Create" : "Save"}
            </Button>
            )}
            {!isNew && selection.kind === "existing" && (
              <>
                {confirmingDelete ? (
                  <>
                    <Text size="13">Move this note to the vault's .trash?</Text>
                    <Button variant="secondary" onClick={() => setConfirmingDelete(false)}>
                      Cancel
                    </Button>
                    <Button
                      variant="primary"
                      disabled={mutations.isPending}
                      onClick={() => mutations.remove({ library, itemId: selection.page.id, revision }, close)}
                    >
                      Move to trash
                    </Button>
                  </>
                ) : (
                  <Button variant="secondary" onClick={() => setConfirmingDelete(true)}>
                    Delete
                  </Button>
                )}
              </>
            )}
          </Inline>
          {!isNew && (
            <Text size="11" muted>
              Saving writes straight into the vault folder, so Obsidian picks the change up on its next sync. A note
              edited somewhere else while it is open here is refused rather than overwritten.
            </Text>
          )}
        </Stack>
      </Page>
    );
  }

  return (
    <Page>
      <Stack gap={16}>
        <Inline justify="space-between" align="center">
          <Heading size="18">Notes</Heading>
          <Button variant="primary" onClick={openNew}>
            New note
          </Button>
        </Inline>
        <SearchField value={search} onChange={setSearch} placeholder="Search notes" autoFocus={false} />
        <Async
          query={entriesQuery}
          empty={() => pages.length === 0}
          emptyMessage="This vault has no markdown notes yet — start one with New note."
        >
          {() => (
            <Stack gap={16}>
              {openError !== null && (
                <Text size="13" tone="error">
                  {openError}
                </Text>
              )}
              {search.trim() !== "" ? (
                visible.length === 0 ? (
                  <Text size="13" muted>
                    No notes match “{search.trim()}”.
                  </Text>
                ) : (
                  <NoteSearchResults notes={visible} onOpen={openPage} openingId={openingId} />
                )
              ) : (
                <NoteTree root={tree} onOpen={openPage} openingId={openingId} />
              )}
            </Stack>
          )}
        </Async>
      </Stack>
    </Page>
  );
}
