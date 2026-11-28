// features/import/screens/ImportScreen.tsx — the drop folder's contents, for a person to read and correct before anything
// moves.
//
// Slice two of the Import port: the list, the filters, the naming choice, and the four fields a person can correct —
// title, author, series and volume — with the server recomputing what each book will be called. Copying is still absent and
// the screen says so: nothing here moves a file.
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { FilterChips } from "../../../ui/FilterChips";
import { Page } from "../../../ui/Page";
import { Stack } from "../../../ui/Stack";
import { Inline } from "../../../ui/Inline";
import { Heading, Text } from "../../../ui/Text";
import { Card } from "../../../ui/Card";
import { Button } from "../../../ui/Button";
import { TextField } from "../../../ui/TextField";
import { EmptyState } from "../../../ui/EmptyState";
import { DataTable, type DataTableCell } from "../../../ui/DataTable";
import { CopyPanel } from "./CopyPanel";
import { Async } from "../../../ui/Async";
import {
  IMPORT_FILTERS,
  NAMING_PRESETS,
  STATUS_TEXT,
  editFromRow,
  fetchImportLibraries,
  fetchImportNames,
  fetchImportPage,
  type ImportEdit,
  type ImportPage,
  type ImportRow,
} from "../api/import";


export function ImportScreen() {
  const [filter, setFilter] = useState("fresh");
  const [naming, setNaming] = useState("default");
  const [edits, setEdits] = useState<Record<string, ImportEdit>>({});
  const [editing, setEditing] = useState<string | undefined>(undefined);

  const libraries = useQuery({ queryKey: ["import", "libraries"], queryFn: fetchImportLibraries });
  // Default to Light Novels when it exists, as the legacy page did, so the shelf a reader is most likely importing into
  // is the one already chosen.
  const library = useMemo(() => {
    const rows = libraries.data ?? [];
    return (rows.find((entry) => entry.id === "light-novels") ?? rows[0])?.id ?? "";
  }, [libraries.data]);

  const page = useQuery({
    queryKey: ["import", "page", library, naming, filter],
    queryFn: () => fetchImportPage({ library, naming, filter }),
    enabled: Boolean(library),
  });

  const rows = page.data?.items ?? [];
  const editOf = (row: ImportRow) => edits[row.source_path] ?? editFromRow(row);
  // A signature of the fields as they stand: the naming call is keyed by it, so an edit re-asks the server what that book
  // will be called and nothing else does.
  // Only the rows a person changed go to the naming call, and only once typing pauses: a key over the whole page fired a
  // hundred-row POST per keystroke, each one cached and none cancelled.
  const changed = rows
    .filter((row) => !sameEdit(editOf(row), editFromRow(row)))
    .map((row) => ({ source: row.source_path, edit: editOf(row) }));
  const signature = changed.map((row) => `${row.source}#${JSON.stringify(row.edit)}`).join("|");
  const debouncedSignature = useDebounced(signature, 300);

  const targets = useQuery({
    queryKey: ["import", "names", naming, debouncedSignature],
    queryFn: () => fetchImportNames(naming, changed),
    enabled: changed.length > 0 && debouncedSignature === signature,
    placeholderData: keepPreviousData,
  });

  const editRow = (row: ImportRow, change: Partial<ImportEdit>) =>
    setEdits((current) => ({ ...current, [row.source_path]: { ...(current[row.source_path] ?? editFromRow(row)), ...change } }));
  const resetRow = (source: string) =>
    setEdits((current) => {
      const next = { ...current };
      delete next[source];
      return next;
    });

  const targetFor = (row: ImportRow) =>
    sameEdit(editOf(row), editFromRow(row)) && row.suggested_target
      ? row.suggested_target
      : targets.data?.get(row.source_path) ?? "";

  const editingRow = rows.find((row) => row.source_path === editing);

  // What copying would act on: books with a name worked out and not already copied. A book whose name is still empty is not
  // offered, because the server would refuse it and there is nothing to gain from saying so twice.
  const copyable = rows
    .filter((row) => !row.copied && targetFor(row) !== "")
    .map((row) => ({ source: row.source_path, target: targetFor(row) }));
  const queryClient = useQueryClient();

  // The four fields a sidecar carries, per copyable book. Only what the screen actually knows: inventing tags or a publisher
  // would write them into the book.
  const copyMetadata = Object.fromEntries(
    rows
      .filter((row) => !row.copied && targetFor(row) !== "")
      .map((row) => {
        const edit = editOf(row);
        return [row.source_path, { title: edit.title, authors: edit.authors, series: edit.series, seriesIndex: edit.seriesIndex }];
      }),
  );

  return (
    <Page wide>
      <Stack gap={16}>
        <Stack gap={4}>
          <Heading size="26">Import</Heading>
          <Text muted>
            Books sitting in the drop folder, and what Mochil works out each one is. Nothing is copied, moved or renamed
            here.
          </Text>
        </Stack>

        <Async query={libraries}>
          {(shelves) =>
            shelves.length === 0 ? (
              <EmptyState message="Import copies books into a library, and this server has no books, comics, manga or mixed shelf to sort them into." />
            ) : (
              <Stack gap={12}>
                <Inline gap={12} align="center" wrap>
                  <FilterChips
                    items={IMPORT_FILTERS.map((entry) => ({ id: entry.id, label: entry.label }))}
                    activeId={filter}
                    onSelect={setFilter}
                  />
                  <Inline gap={8} align="center">
                    <Text size="12" muted>
                      Names
                    </Text>
                    <select aria-label="Naming" value={naming} onChange={(event) => setNaming(event.target.value)}>
                      {NAMING_PRESETS.map((entry) => (
                        <option key={entry.id} value={entry.id}>
                          {entry.label}
                        </option>
                      ))}
                    </select>
                  </Inline>
                </Inline>

                <Async query={page}>
                  {(data) =>
                    !data.enabled ? (
                      <Card>
                        <Stack gap={8}>
                          <Heading size="18" level={3}>
                            No drop folder yet
                          </Heading>
                          <Text muted>
                            Import reads from a folder on the server that you point it at. Until one is set there is
                            nothing to list — and nothing here guesses at a path.
                          </Text>
                        </Stack>
                      </Card>
                    ) : (
                      <Stack gap={12}>
                        {targets.isError ? (
                          <Text size="12" tone="error">
                            The new names could not be worked out, so the column shows the server's first guess. Editing a
                            book will try again.
                          </Text>
                        ) : null}
                        <Text muted>
                          Reading <code>{data.incoming_directory}</code>
                          {data.managed_directory ? (
                            <>
                              {" "}
                              · copied books go to <code>{data.managed_directory}</code>
                            </>
                          ) : null}
                        </Text>
                        {data.items.length === 0 ? (
                          <EmptyState message={emptyMessage(filter, data)} />
                        ) : (
                          <>
                            <Card>
                              <DataTable
                                caption="Books in the drop folder"
                                columns={[
                                  { key: "status", label: "Status" },
                                  { key: "title", label: "Title" },
                                  { key: "authors", label: "Author" },
                                  { key: "series", label: "Series" },
                                  { key: "index", label: "#", align: "end" },
                                  { key: "destination", label: "Will go to" },
                                  { key: "target", label: "Will be saved as" },
                                  { key: "source", label: "From the drop folder" },
                                ]}
                                actionColumnLabel="Correct"
                                rows={data.items.map((row) => ({
                                  key: row.source_path,
                                  cells: cellsFor(row, editOf(row), targetFor(row), destinationName(row, data)),
                                  action: {
                                    label: "Correct…",
                                    ariaLabel: `Correct ${editOf(row).title || row.source_path}`,
                                    onClick: () => setEditing(row.source_path),
                                  },
                                  selected: editing === row.source_path,
                                }))}
                              />
                            </Card>
                            <CopyPanel
                              entries={copyable}
                              metadata={copyMetadata}
                              onCopied={() => {
                                void queryClient.invalidateQueries({ queryKey: ["import", "page"] });
                              }}
                            />
                            {editingRow ? (
                              <RowEditor
                                key={editingRow.source_path}
                                row={editingRow}
                                edit={editOf(editingRow)}
                                onChange={(change) => editRow(editingRow, change)}
                                onReset={() => resetRow(editingRow.source_path)}
                                onDone={() => setEditing(undefined)}
                                target={targetFor(editingRow)}
                              />
                            ) : null}
                          </>
                        )}
                        <Text muted size="12">
                          {shownMessage(data)} Nothing is copied until you review the list and press copy.
                        </Text>
                      </Stack>
                    )
                  }
                </Async>
              </Stack>
            )
          }
        </Async>
      </Stack>
    </Page>
  );
}

/** True when a row still holds exactly what the server proposed. */
function sameEdit(left: ImportEdit, right: ImportEdit) {
  return (
    left.title === right.title &&
    left.series === right.series &&
    left.seriesIndex === right.seriesIndex &&
    left.authors.join("\u0000") === right.authors.join("\u0000")
  );
}

/** A value that follows `value` once it has stopped changing, so a request is not sent per keystroke. */
function useDebounced<T>(value: T, delay: number): T {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(value), delay);
    return () => window.clearTimeout(timer);
  }, [value, delay]);
  return settled;
}

/** What an empty list actually means under each filter — "already copied" is only ever true of Everything. */
function emptyMessage(filter: string, data: ImportPage) {
  if (data.truncated) return "This folder is larger than one pass can walk, so there is more in it than is listed here.";
  switch (filter) {
    case "possible":
      return "No possible duplicates: nothing here matches a book you already have.";
    case "in_library":
      return "Nothing in the drop folder is already in the library.";
    case "new_volume":
      return "No new volumes of series you already have.";
    case "new":
      return "No books from series you do not have yet.";
    case "fresh":
      return "Nothing new: everything in the drop folder is already in the library or copied.";
    default:
      return "The drop folder is empty.";
  }
}

/** How many of the folder's books this page is showing, without implying the folder ends at the page. */
function shownMessage(data: ImportPage) {
  if (data.truncated) return `${data.items.length} shown, and this folder is too large to walk in one pass.`;
  if (data.next_cursor) return `${data.items.length} shown of at least ${data.scanned}; more are in the folder.`;
  return `${data.items.length} shown of ${data.scanned} read.`;
}

/**
 * One book's four fields. Author and volume are kept as the text a person is typing — splitting authors on every keystroke
 * made a comma impossible to type ("Le Guin," came out "Le Guin"), and Number("1e") is NaN, which rendered as "NaN" and
 * silently dropped the volume on send.
 */
function RowEditor({
  row,
  edit,
  onChange,
  onReset,
  onDone,
  target,
}: {
  row: ImportRow;
  edit: ImportEdit;
  onChange: (change: Partial<ImportEdit>) => void;
  onReset: () => void;
  onDone: () => void;
  target: string;
}) {
  const [authorsText, setAuthorsText] = useState(edit.authors.join(", "));
  const [volumeText, setVolumeText] = useState(edit.seriesIndex === null ? "" : String(edit.seriesIndex));
  const volume = volumeText.trim() === "" ? null : Number(volumeText.replace(",", "."));
  const volumeBad = volume !== null && !Number.isFinite(volume);
  return (
    <Card>
      <Stack gap={12}>
        <Heading size="18" level={3}>
          {edit.title || row.source_path}
        </Heading>
        <TextField label="Title" value={edit.title} onChange={(event) => onChange({ title: event.target.value })} />
        <TextField
          label="Author"
          value={authorsText}
          onChange={(event) => {
            setAuthorsText(event.target.value);
            onChange({
              authors: event.target.value
                .split(",")
                .map((value) => value.trim())
                .filter(Boolean),
            });
          }}
        />
        <Inline gap={12} align="start" wrap>
          <TextField label="Series" value={edit.series} onChange={(event) => onChange({ series: event.target.value })} />
          <TextField
            label="Volume"
            inputMode="decimal"
            value={volumeText}
            onChange={(event) => {
              setVolumeText(event.target.value);
              const next = event.target.value.trim() === "" ? null : Number(event.target.value.replace(",", "."));
              onChange({ seriesIndex: next !== null && Number.isFinite(next) ? next : null });
            }}
          />
        </Inline>
        {volumeBad ? (
          <Text size="12" tone="error">
            A volume has to be a number; this one is being ignored.
          </Text>
        ) : null}
        <Text size="12" muted>
          Will be saved as <code>{target || "— needs a title"}</code>
        </Text>
        <Inline gap={8}>
          <Button variant="secondary" onClick={onReset}>
            Undo my edits
          </Button>
          <Button onClick={onDone}>Done</Button>
        </Inline>
      </Stack>
    </Card>
  );
}

function destinationName(row: ImportRow, data: ImportPage) {
  const chosen = data.destinations.find((entry) => entry.id === row.destination);
  if (chosen) return chosen.name;
  return data.managed_library?.name ?? "Not sorted yet";
}

function cellsFor(row: ImportRow, edit: ImportEdit, target: string, destination: string): DataTableCell[] {
  const conflicts = row.proposal?.conflicts ?? [];
  const guessedAuthor = (row.proposal?.evidence ?? []).some(
    (entry) => entry.field === "authors" && entry.confidence === "low",
  );
  const titleChanged = edit.title !== (row.proposal?.title ?? "");
  return [
    {
      text: row.copied ? "Copied" : STATUS_TEXT[row.status] ?? row.status,
      tone: "pill",
      status: row.copied ? "in_library" : row.status,
    },
    {
      text: edit.title || "needs a title",
      ...(!edit.title ? { tone: "warn" as const } : titleChanged ? { tone: "warn" as const, title: "You changed this" } : {}),
      ...(conflicts.length > 0 ? { title: conflicts.join("\n") } : {}),
    },
    {
      text: edit.authors.join(", ") || "—",
      ...(guessedAuthor ? { tone: "warn" as const, title: "Guessed from a folder name" } : {}),
    },
    { text: edit.series || "—" },
    { text: edit.seriesIndex ?? "" },
    { text: destination },
    { text: target || "needs a title", ...(target ? { tone: "muted" as const } : { tone: "warn" as const }) },
    { text: row.source_path, tone: "muted", title: row.source_path },
  ];
}
