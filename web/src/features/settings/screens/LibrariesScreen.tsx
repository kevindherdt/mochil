// features/settings/screens/LibrariesScreen.tsx — Settings → Libraries (plan 2.2, the biggest panel).
//
// Shows what is configured, adds a library by walking the server's browsable folders, and edits one: name, kind,
// rescan interval, page streaming, metadata writes and preferred formats — the fields the legacy settings page used
// to own. Every option offered here is one the server validates and accepts (internal/appconfig/config.go).
//
// Still not built, and not pretended to be: choosing a Calibre or remote source when adding. Those need a source
// picker with `metadata.db` detection, which is the next slice, not a field on this form.
//
// Two decisions worth knowing:
//   * the metadata provider is never asked for. The server derives it from the kind (appconfig's
//     `automaticMetadataProvider`), and Kevin's standing ruling is that the kind already decides it. A new library
//     is written with `metadata_provider: ""` so it follows the kind.
//   * saving is a live apply, so the panel reports what the server actually did (`libraries_scanned`) instead of
//     telling the reader to restart or rescan by hand.
import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { Async, Button, Card, Heading, Inline, Page, PillRow, Stack, Switch, Text, TextField } from "../../../ui";
import { ChevronLeft, Folder, FolderOpen } from "lucide-react";
import { useSession } from "../../auth/useSession";
import { useEditorEnabled } from "../../../lib/useModules";
import { useLibraries } from "../../library/useLibraries";
import { useFolders, useLibraryAdmin, useLibraryMutations } from "../useLibrariesAdmin";
import {
  LIBRARY_KINDS,
  PREFERRED_FORMATS,
  SCAN_INTERVAL_OPTIONS,
  inspectLibraryFolder,
  isPublicationKind,
  readLibraryOptions,
  supportsMetadataWrites,
  type LibraryKind,
  type LibraryOptions,
} from "../api/libraries";
import { messageFromError } from "../api/accounts";

export function LibrariesScreen() {
  const session = useSession();
  const editorEnabled = useEditorEnabled();
  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Libraries</Heading>
        {/* The editor is the other half of "libraries": adding one is here, correcting what is in it lives in the
            grid. It was reachable by URL only, which is no way to ship a daily-use screen. */}
        {editorEnabled && (
          <Inline gap={8} align="center" wrap>
            <Text size="12" muted>
              Correcting the metadata of many books at once?
            </Text>
            <Link
              to="/editor"
              style={{
                display: "inline-flex",
                alignItems: "center",
                minHeight: "var(--space-24)",
                fontSize: "var(--font-size-12)",
                color: "var(--color-accent)",
              }}
            >
              Open the metadata editor
            </Link>
          </Inline>
        )}
        <Async query={session}>
          {(me) => (
            <Stack gap={12}>
              {/* Import is a basic need, not the optional spreadsheet editor, so it is not behind the editor module; the
                  endpoints are administrator-only, so the link is too. */}
              {me.role === "administrator" ? (
                <Inline gap={8} align="center" wrap>
                  <Text size="12" muted>
                    Filling a library from a folder of files?
                  </Text>
                  <Link
                    to="/import"
                    style={{
                      display: "inline-flex",
                      alignItems: "center",
                      minHeight: "var(--space-24)",
                      fontSize: "var(--font-size-12)",
                      color: "var(--color-accent)",
                    }}
                  >
                    Open Import
                  </Link>
                </Inline>
              ) : null}
              <LibrariesBody isAdministrator={me.role === "administrator"} />
            </Stack>
          )}
        </Async>
      </Stack>
    </Page>
  );
}

function LibrariesBody({ isAdministrator }: { isAdministrator: boolean }) {
  const admin = useLibraryAdmin(isAdministrator);
  const browsing = useLibraries();
  const mutations = useLibraryMutations();

  // The folder browser is only mounted while adding; `pickPath` undefined means "not adding".
  const [pickPath, setPickPath] = useState<string | undefined>(undefined);
  const [chosenPath, setChosenPath] = useState<string | undefined>(undefined);
  const [name, setName] = useState("");
  const [kind, setKind] = useState<LibraryKind>("books");
  const [justAdded, setJustAdded] = useState<string | undefined>(undefined);
  // What the folder turned out to hold: the server inspects it when it is chosen, so the name, the kind and even
  // whether it is a Calibre library are filled in rather than asked for. `storage` is what `source.type` is written
  // with — a Calibre folder is read differently from a plain one.
  const [inspection, setInspection] = useState<{ evidence: string[]; storage: string } | undefined>(undefined);
  const [inspecting, setInspecting] = useState(false);
  // Which library's Remove has been pressed once. Removing only edits the config (files stay), but it also throws
  // away that library's own settings — preferred formats, scan interval, Calibre options — which have to be set up
  // again from scratch. One tap is too little for that; a second, explicit one is enough.
  const [confirmRemove, setConfirmRemove] = useState<string | undefined>(undefined);
  // Which library is being renamed / re-kinded. Kind is not cosmetic — it decides how the folder is scanned and which
  // metadata provider the server derives — so the row says that while it is open rather than acting as if this were
  // a label change.
  const [editing, setEditing] = useState<string | undefined>(undefined);
  const [editName, setEditName] = useState("");
  const [editKind, setEditKind] = useState<LibraryKind>("books");
  // The switches the legacy settings page owned: rescan interval, page streaming, metadata writes, preferred formats.
  // They are read from the library's own object when the row opens, so a field this panel does not model is still
  // preserved on save (updateLibrary spreads the original object).
  const [editOptions, setEditOptions] = useState<LibraryOptions>({
    scanEvery: "",
    pageStreaming: false,
    metadataWrites: false,
    preferredFormats: [],
  });

  if (!isAdministrator) {
    return <Text muted>Only an administrator can add or change libraries on this server.</Text>;
  }

  const counts = new Map((browsing.data ?? []).map((library) => [library.id, library.count]));

  function startAdding() {
    mutations.reset();
    setJustAdded(undefined);
    setChosenPath(undefined);
    setPickPath("");
    setName("");
  }

  function cancelAdding() {
    setPickPath(undefined);
    setChosenPath(undefined);
    setName("");
    mutations.reset();
  }

  return (
    <Stack gap={24}>
      <Text muted>
        Each library is one folder of files on this server. Saving takes effect immediately — the server opens the
        new folder and scans it.
      </Text>

      {mutations.error !== null && mutations.error !== undefined && (
        <Text size="13" tone="error">{messageFromError(mutations.error)}</Text>
      )}
      {justAdded !== undefined && <Text size="13">{justAdded}</Text>}

      <Async query={admin}>
        {(state) => (
          <Stack gap={12}>
            {state.entries.length === 0 && <Text muted>No libraries configured yet.</Text>}
            {state.entries.map((entry) => (
              <Card key={entry.id}>
                <Stack gap={8}>
                  <Inline gap={12} align="center">
                    <div style={{ flex: 1, minWidth: 0 }}>
                      {editing === entry.id ? (
                        <Stack gap={8}>
                          <TextField label="Name" value={editName} onChange={(event) => setEditName(event.target.value)} />
                          <Stack gap={4}>
                            <Text size="11" muted>
                              Kind — decides how this folder is scanned and which metadata provider is used. The files
                              are not touched; the index is rebuilt on the next scan.
                            </Text>
                            <PillRow
                              items={LIBRARY_KINDS}
                              activeId={editKind}
                              onSelect={(id) => setEditKind(id as LibraryKind)}
                            />
                          </Stack>

                          <Stack gap={4}>
                            <Text size="11" muted>
                              Rescan — how often this one library is checked. Large or rarely changing folders can be
                              checked less often than the server default.
                            </Text>
                            <PillRow
                              items={SCAN_INTERVAL_OPTIONS}
                              activeId={editOptions.scanEvery}
                              onSelect={(id) => setEditOptions({ ...editOptions, scanEvery: id })}
                            />
                          </Stack>

                          {isPublicationKind(editKind) && (
                            <Stack gap={4}>
                              <Text size="11" muted>
                                Preferred formats — first match wins when the same book exists in several. Tap in the
                                order you want them; the number is that order.
                              </Text>
                              <Inline gap={8}>
                                {PREFERRED_FORMATS.map((format) => {
                                  const position = editOptions.preferredFormats.indexOf(format);
                                  return (
                                    <Button
                                      key={format}
                                      variant={position === -1 ? "secondary" : "primary"}
                                      onClick={() =>
                                        setEditOptions({
                                          ...editOptions,
                                          preferredFormats:
                                            position === -1
                                              ? [...editOptions.preferredFormats, format]
                                              : editOptions.preferredFormats.filter((value) => value !== format),
                                        })
                                      }
                                    >
                                      {position === -1 ? format : `${position + 1}. ${format}`}
                                    </Button>
                                  );
                                })}
                              </Inline>
                            </Stack>
                          )}

                          {/* `Switch` is a bare track with only an aria-label — the caller draws the text, or the
                              panel shows two unlabelled toggles nobody can identify. */}
                          <Stack gap={8}>
                            <Inline gap={12} align="center">
                              <Switch
                                label="OPDS page streaming"
                                checked={editOptions.pageStreaming}
                                onChange={(checked) => setEditOptions({ ...editOptions, pageStreaming: checked })}
                              />
                              <Stack gap={4}>
                                <Text size="12">OPDS page streaming</Text>
                                <Text size="11" muted>
                                  Let compatible readers open comic pages one at a time instead of the whole file.
                                </Text>
                              </Stack>
                            </Inline>
                            {supportsMetadataWrites(entry) && isPublicationKind(editKind) && (
                              <Inline gap={12} align="center">
                                <Switch
                                  label="Write metadata into files"
                                  checked={editOptions.metadataWrites}
                                  onChange={(checked) => setEditOptions({ ...editOptions, metadataWrites: checked })}
                                />
                                <Stack gap={4}>
                                  <Text size="12">Write metadata into files</Text>
                                  <Text size="11" muted>
                                    Let Mochil save corrected metadata back into the book files. Off keeps every file
                                    read-only.
                                  </Text>
                                </Stack>
                              </Inline>
                            )}
                          </Stack>
                        </Stack>
                      ) : (
                      <Stack gap={4}>
                        <Text as="span" size="14">
                          {entry.name}
                          <Text as="span" size="11" muted>
                            {` · ${entry.kind}`}
                            {counts.get(entry.id) === undefined ? "" : ` · ${counts.get(entry.id)} items`}
                          </Text>
                        </Text>
                        {entry.path !== "" && <Text size="11" muted>{entry.path}</Text>}
                      </Stack>
                      )}
                    </div>
                  </Inline>
                  <Inline gap={8}>
                    {editing === entry.id ? (
                      <>
                        <Button
                          variant="primary"
                          disabled={mutations.isPending || editName.trim() === ""}
                          onClick={() =>
                            mutations.update(
                              {
                                id: entry.id,
                                name: editName.trim(),
                                ...(editKind === entry.kind ? {} : { kind: editKind }),
                                options: editOptions,
                              },
                              (result) => {
                                setEditing(undefined);
                                setJustAdded(`${editName.trim()} updated.${result.applied ? "" : " Restart Mochil to finish applying it."}`);
                              },
                            )
                          }
                        >
                          Save
                        </Button>
                        <Button variant="secondary" onClick={() => setEditing(undefined)}>
                          Cancel
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="secondary"
                        onClick={() => {
                          setEditing(entry.id);
                          setEditName(entry.name);
                          setEditKind(entry.kind as LibraryKind);
                          setEditOptions(readLibraryOptions(entry));
                        }}
                      >
                        Edit
                      </Button>
                    )}
                    <Button variant="secondary" onClick={() => mutations.scan(entry.id)}>
                      Scan now
                    </Button>
                    {confirmRemove === entry.id ? (
                      <>
                        <Button
                          variant="primary"
                          onClick={() => {
                            setConfirmRemove(undefined);
                            mutations.reset();
                            mutations.remove(entry.id, () => {
                              setJustAdded(`Removed ${entry.name}. Its files were not touched.`);
                            });
                          }}
                        >
                          Remove {entry.name}
                        </Button>
                        <Button variant="secondary" onClick={() => setConfirmRemove(undefined)}>Cancel</Button>
                      </>
                    ) : (
                      <Button variant="secondary" onClick={() => setConfirmRemove(entry.id)}>Remove</Button>
                    )}
                  </Inline>
                </Stack>
              </Card>
            ))}
          </Stack>
        )}
      </Async>

      {pickPath === undefined ? (
        <Inline gap={8}>
          <Button variant="secondary" onClick={startAdding}>Add a library</Button>
        </Inline>
      ) : (
        <Card>
          <Stack gap={12}>
            <Heading level={2} size="16">Add a library</Heading>

            {chosenPath === undefined ? (
              <FolderBrowser
                path={pickPath}
                onNavigate={setPickPath}
                onChoose={(path) => {
                  setChosenPath(path);
                  setName(path.split("/").filter(Boolean).pop() ?? "Library");
                  setInspection(undefined);
                  setInspecting(true);
                  void inspectLibraryFolder(path)
                    .then((result) => {
                      // The folder's own name is a better default than a guess, and only an empty name is replaced.
                      setName((current) => (current.trim() === "" ? result.suggestedName : current));
                      setKind(result.suggestedKind);
                      setInspection({ evidence: result.evidence, storage: result.storage });
                    })
                    .catch(() => setInspection(undefined))
                    .finally(() => setInspecting(false));
                }}
              />
            ) : (
              <Stack gap={12}>
                <Text size="13" muted>{chosenPath}</Text>
                <TextField label="Name" value={name} autoComplete="off" onChange={(event) => setName(event.target.value)} />
                {inspecting && (
                  <Text size="11" muted>
                    Looking at what is in this folder…
                  </Text>
                )}
                {inspection !== undefined && (
                  <Stack gap={4}>
                    <Text size="12">
                      {inspection.storage === "calibre"
                        ? "This looks like a Calibre library."
                        : "This looks like a plain folder."}
                    </Text>
                    {inspection.evidence.length > 0 && (
                      <Text size="11" muted>
                        {inspection.evidence.join(" · ")}
                      </Text>
                    )}
                  </Stack>
                )}
                <Stack gap={8}>
                  <Text size="13" muted>Kind</Text>
                  <PillRow items={LIBRARY_KINDS} activeId={kind} onSelect={(id) => setKind(id as LibraryKind)} />
                  <Text size="11" muted>
                    The kind decides what Mochil looks for in the folder, and which metadata source it uses.
                  </Text>
                </Stack>
                <Inline gap={8}>
                  <Button
                    variant="primary"
                    disabled={name.trim() === "" || mutations.isPending}
                    onClick={() =>
                      mutations.add(
                        { name: name.trim(), kind, path: chosenPath, ...(inspection === undefined ? {} : { storage: inspection.storage }) },
                        (result) => {
                        const scanned = result.libraries_scanned ?? 0;
                        setJustAdded(
                          `Added ${name.trim()}. ${result.applied ? "Applied" : "Saved"} — ${result.libraries} ${
                            result.libraries === 1 ? "library" : "libraries"
                          }${scanned > 0 ? `, ${scanned} scanned` : ""}.`,
                        );
                        setPickPath(undefined);
                        setChosenPath(undefined);
                        setName("");
                        setInspection(undefined);
                      })
                    }
                  >
                    Add library
                  </Button>
                  <Button variant="secondary" onClick={() => setChosenPath(undefined)}>Choose another folder</Button>
                  <Button variant="secondary" onClick={cancelAdding}>Cancel</Button>
                </Inline>
              </Stack>
            )}
          </Stack>
        </Card>
      )}

      <Text size="11" muted>
        Not here yet: adding a library from a remote source (WebDAV or an OPDS catalogue) — a folder on this server is
        all this panel can add. Everything else on the old settings page — Calibre detection, formats, streaming,
        metadata writes, rescan interval — is here.
      </Text>
    </Stack>
  );
}

/** Walks the server's browsable roots (GET /api/folders). "Use this folder" is offered at every level, because a
 *  library is very often the folder you are already standing in — being forced to descend one level deeper to
 *  say "here" is the usual way these pickers get it wrong. */
function FolderBrowser({ path, onNavigate, onChoose }: { path: string; onNavigate: (path: string) => void; onChoose: (path: string) => void }) {
  const listing = useFolders(path);
  return (
    <Stack gap={12}>
      <Inline gap={8} align="center">
        {path !== "" && (
          <Button variant="secondary" onClick={() => onNavigate(listing.data?.parent ?? "")}>
            <ChevronLeft size={16} aria-hidden="true" /> Up
          </Button>
        )}
        <Text size="12" muted>{path === "" ? "Browsable folders" : path}</Text>
      </Inline>

      <Async query={listing}>
        {(data) => (
          <Stack gap={8}>
            {data.directories.length === 0 && <Text size="13" muted>No sub-folders here.</Text>}
            {data.directories.map((directory) => (
              <Inline key={directory.path} gap={8} align="center">
                <Button variant="secondary" onClick={() => onNavigate(directory.path)}>
                  <Folder size={16} aria-hidden="true" /> {directory.name}
                </Button>
                <Button variant="secondary" onClick={() => onChoose(directory.path)}>
                  <FolderOpen size={16} aria-hidden="true" /> Use this folder
                </Button>
              </Inline>
            ))}
            {data.truncated && <Text size="11" muted>Only the first 500 folders are listed.</Text>}
          </Stack>
        )}
      </Async>

      {path !== "" && (
        <Inline gap={8}>
          <Button variant="primary" onClick={() => onChoose(path)}>
            <FolderOpen size={16} aria-hidden="true" /> Use this folder
          </Button>
        </Inline>
      )}
    </Stack>
  );
}
