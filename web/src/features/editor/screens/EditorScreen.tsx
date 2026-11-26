// features/editor/screens/EditorScreen.tsx — the metadata editor.
//
// A spreadsheet over the library: pick columns, sort by any of them, select many rows and set one field across all of
// them, then inspect a single book's every field in a details pane underneath. That shape is the shape the work has —
// fixing a wrong series name is a bulk job, fixing a description is not.
//
// The whole library is loaded (in 1 000-row chunks) and scrolled as one virtualized list, the way the legacy editor
// did it — paging is the wrong mental model for a spreadsheet and breaks selection and renumber across pages. Only the
// rows on screen are drawn; blank space stands in for the rest (see windowFor).
//
// Deliberately included: preview-before-write and undo, the Calibre ownership guard, a details pane as the beginner's
// comfort zone, spreadsheet keys (see keys.ts), the bulk-ops dialog, column resize, and the find-ISBNs / find-matches /
// AI-ask tools, collections and transfer.
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Columns3, Folders, LibraryBig, ListFilter, Moon, Search, Sun } from "lucide-react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Page, Heading, Text, Button, TextField, Stack, Inline, Async, EmptyState, ResizeHandle, Menu, IconButton, Toolbar } from "../../../ui";
import { useEditorEnabled } from "../../../lib/useModules";
import { useThemeContext } from "../../theme/ThemeContext";
import { useShellVariant } from "../../../shell/ShellVariant";
import { BulkEditDialog } from "../BulkEditDialog";
import { EditorTools, type HermesAbilities } from "../EditorTools";
import { MatchDialog } from "../MatchDialog";
import { hardcoverTone, isbnTone, issueTone, scoreLabel, scoreTone, toneColor, toneInk, toneMark, toneWash, toneWorst } from "../tone";
import { gridKey } from "../keys";
import { ReviewPanel } from "../ReviewPanel";
import { CollectionsDialog } from "../CollectionsDialog";
import { TransferDialog } from "../TransferDialog";
import {
  FIELD_LABELS,
  applyChanges,
  matchReplace,
  changeCount,
  changesFrom,
  editableFieldsFor,
  CALIBRE_OWNED_FIELDS,
  fetchAllEditorBooks,
  fetchCollections,
  mergeChanges,
  previewChanges,
  tagChanges,
  undoChanges,
  withoutChanges,
  withoutSources,
  isbnOf,
  isbnState,
  sortKeyFor,
  valueOf,
  yearOf,
  windowFor,
  type EditorBook,
  type EditorChange,
  type EditorChanges,
  type EditorField,
  type EditorSources,
} from "../api/editor";
import { dismissBooks } from "../taskDismissals";

interface EditorLibraries {
  libraries: LibraryOption[];
  hermes: HermesAbilities;
}

interface LibraryOption {
  id: string;
  name: string;
  kind: string;
  /** "filesystem", "calibre", … — decides which fields may be written. */
  sourceType: string;
}

const PUBLICATION_KINDS = ["books", "comics", "manga", "mixed"];

// One shared empty object for the rows with nothing staged. `changes[book.id] ?? {}` made a fresh object on every render,
// and a fresh prop defeats memo on every row at once — the table re-rendered all 33 visible rows for one cursor move.
const NO_CHANGES: Partial<Record<EditorField, string>> = {};
const ROW_HEIGHT = 34;
// Mirrors the server's editorMaximumEdits: one request saves at most this many books.
const EDITOR_MAXIMUM_EDITS = 2000;

// The column layout is a preference, not server state: it lives in this browser, the way Calibre keeps its own.
function loadLayout<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key);
    return raw === null ? fallback : (JSON.parse(raw) as T);
  } catch {
    return fallback;
  }
}

function saveLayout(key: string, value: unknown): void {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // A browser in private mode can refuse; the layout simply does not persist.
  }
}

type ColumnId =
  | EditorField
  | "rating"
  | "format"
  | "collections"
  | "progress"
  | "words"
  | "pages"
  | "isbn_found"
  | "match"
  | "hardcover"
  | "status"
  | "path";

interface Column {
  id: ColumnId;
  label: string;
  /** Starting width in pixels; a drag or the saved layout can override it. */
  defaultWidth: number;
  /** Present when the column can be written; absent means the cell is read-only. */
  field?: EditorField;
  /** Starts hidden; every column except the title can be shown or hidden from the column picker. */
  startsHidden?: boolean;
  /** The one column that cannot be hidden: a row with no title cannot be told from the next row. */
  fixed?: boolean;
  num?: boolean;
}

const COLUMNS: Column[] = [
  { id: "title", label: "Title", defaultWidth: 260, field: "title", fixed: true },
  { id: "authors", label: "Authors", defaultWidth: 180, field: "authors" },
  { id: "series", label: "Series", defaultWidth: 150, field: "series" },
  { id: "series_index", label: "#", defaultWidth: 56, field: "series_index", num: true },
  { id: "published", label: "Year", defaultWidth: 70, field: "published" },
  { id: "publisher", label: "Publisher", defaultWidth: 140, field: "publisher" },
  { id: "format", label: "Format", defaultWidth: 80 },
  { id: "status", label: "Status", defaultWidth: 190 },
  { id: "tags", label: "Tags", defaultWidth: 150, field: "tags", startsHidden: true },
  { id: "language", label: "Language", defaultWidth: 90, field: "language", startsHidden: true },
  { id: "rating", label: "Rating", defaultWidth: 80, startsHidden: true, num: true },
  { id: "identifiers", label: "Identifiers", defaultWidth: 150, field: "identifiers", startsHidden: true },
  { id: "collections", label: "Collections", defaultWidth: 140, startsHidden: true },
  { id: "progress", label: "Progress", defaultWidth: 90, startsHidden: true, num: true },
  { id: "words", label: "Words", defaultWidth: 90, startsHidden: true, num: true },
  { id: "pages", label: "Pages", defaultWidth: 70, startsHidden: true, num: true },
  { id: "isbn_found", label: "Found ISBN", defaultWidth: 130, startsHidden: true },
  // Visible from the start: this column is the one that answers "how sure is this match, and is it even the same book",
  // and it is the way into the candidate list. Hidden by default, the whole feature had no door.
  { id: "match", label: "Match", defaultWidth: 170 },
  { id: "hardcover", label: "Hardcover", defaultWidth: 120, startsHidden: true },
  { id: "path", label: "File", defaultWidth: 180, startsHidden: true },
];

// How each Hardcover state reads; the note behind it is the cell's tooltip.
const HARDCOVER_TEXT: Record<string, string> = {
  synced: "Synced",
  doubt: "Check match",
  linked: "Linked",
  nomatch: "Not on Hardcover",
  ignored: "Left out",
};

/** What a read-only cell shows. Editable columns go through valueOf instead. */
function readOnlyText(book: EditorBook, id: ColumnId): { text: string; title: string } {
  switch (id) {
    case "format":
      return { text: book.format.toUpperCase(), title: book.format };
    case "collections": {
      const joined = book.collections.join(", ");
      return { text: joined, title: joined };
    }
    case "progress":
      return { text: book.progress >= 1 ? "Finished" : book.progress > 0 ? `${Math.max(1, Math.round(book.progress * 100))}%` : "", title: "" };
    case "words":
      return { text: book.words > 0 ? book.words.toLocaleString() : "", title: book.words > 0 ? `${book.words} words` : "not counted yet" };
    case "pages":
      return { text: book.pages > 0 ? String(book.pages) : "", title: book.pages > 0 ? `about ${book.pages} pages` : "not counted yet" };
    case "isbn_found": {
      // The file's ISBN, and — when the book already has one — whether they agree. A difference is the whole point of
      // reading a book that already carries an ISBN, so it is marked rather than silently shown.
      const state = isbnState(book);
      if (state === "differs") return { text: `${book.isbnFound} ≠`, title: `${book.isbnNote ?? ""}\nThe book's own metadata says ${isbnOf(book)}.`.trim() };
      if (state === "same") return { text: `${book.isbnFound} ✓`, title: `${book.isbnNote ?? ""}\nAgrees with the metadata.`.trim() };
      return { text: book.isbnFound, title: book.isbnNote };
    }
    case "match":
      return book.match === undefined
        ? { text: "", title: book.matchNote }
        : { text: `${book.match.title} · ${Math.round(Math.max(0, Math.min(1, book.match.score)) * 100)}%${book.matchClear ? "" : " ?"}`, title: book.matchNote };
    case "hardcover":
      return { text: HARDCOVER_TEXT[book.hardcover] ?? "", title: book.hardcoverNote };
    case "rating":
      // The row's rating is already stars (0–5); the server divided the stored 0–10 before sending.
      return { text: book.rating > 0 ? "★".repeat(Math.round(book.rating)) : "", title: book.rating > 0 ? `${book.rating} of 5` : "unrated" };
    case "path":
      return { text: book.path, title: book.path };
    default:
      return { text: "", title: "" };
  }
}

const FILTERS = [
  { value: "", label: "All books" },
  { value: "edited", label: "Edited" },
  { value: "issues", label: "Issues" },
];

export function EditorScreen() {
  // Off means no editor at all, not a half-working one: the module switch has to mean something.
  const editorEnabled = useEditorEnabled();
  const queryClient = useQueryClient();
  const libraries = useQuery({
    queryKey: ["editor", "libraries"],
    queryFn: async (): Promise<EditorLibraries> => {
      const [settings, response] = await Promise.all([
        fetch("/api/settings", { credentials: "include" }).then((result) => result.json()),
        fetch("/api/libraries", { credentials: "include" }),
      ]);
      if (!response.ok) throw new Error("The libraries could not be loaded.");
      const sourceTypes = new Map<string, string>(
        ((settings as { libraries?: { id?: string; source?: { type?: string } }[] }).libraries ?? []).map((entry) => [
          entry.id ?? "",
          entry.source?.type ?? "filesystem",
        ]),
      );
      const body = (await response.json()) as { libraries?: Omit<LibraryOption, "sourceType">[] } | Omit<LibraryOption, "sourceType">[];
      const list = Array.isArray(body) ? body : (body.libraries ?? []);
      const raw = (settings as { hermes?: { url?: string; model?: string; helper_url?: string; helper_model?: string; features?: Record<string, boolean> } }).hermes ?? {};
      const ability = (name: string) => raw.features?.[name] !== false;
      return {
        libraries: list
          .filter((library) => PUBLICATION_KINDS.includes(library.kind))
          .map((library) => ({ ...library, sourceType: sourceTypes.get(library.id) ?? "filesystem" })),
        hermes: {
          // A dedicated helper model, or the gateway itself answering with the profile's default model.
          helper:
            ((raw.helper_url ?? "").trim() !== "" && (raw.helper_model ?? "").trim() !== "") ||
            ((raw.url ?? "").trim() !== "" && (raw.model ?? "").trim() !== ""),
          isbn: ability("isbn"),
          match: ability("match"),
        },
      };
    },
    staleTime: 5 * 60_000,
  });

  const [chosenLibrary, setChosenLibrary] = useState<string | undefined>(undefined);
  const libraryId = chosenLibrary ?? libraries.data?.libraries[0]?.id;

  const [search, setSearch] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [filter, setFilter] = useState("");
  /** A collection id to narrow the library to, or "" for all of it. */
  const [collection, setCollection] = useState("");
  const [sort, setSort] = useState("title");
  const [dir, setDir] = useState<"asc" | "desc">("asc");
  const [hidden, setHidden] = useState<ColumnId[]>(() => {
    // The stored answer wins. The migration reads the *old* key exactly once — its first version re-derived the hidden
    // set from that key on every load, so a column hidden in this build came back on the next reload (the old key was
    // still there and knew nothing about it). Kevin: "remember the selected columns that I show and hide".
    const stored = loadLayout<ColumnId[] | undefined>("mochil-editor-hidden", undefined);
    if (stored !== undefined) return stored;
    const legacy = loadLayout<ColumnId[] | undefined>("mochil-editor-columns", undefined);
    const migrated =
      legacy === undefined
        ? COLUMNS.filter((column) => column.startsHidden === true).map((column) => column.id)
        : COLUMNS.filter((column) => column.startsHidden === true && !legacy.includes(column.id)).map((column) => column.id);
    try {
      localStorage.removeItem("mochil-editor-columns");
    } catch {
      // Private mode: the migration just runs once more next time.
    }
    return migrated;
  });
  const [order, setOrder] = useState<string[]>(() => loadLayout<string[]>("mochil-editor-order", []));
  const [widths, setWidths] = useState<Record<string, number>>(() => loadLayout<Record<string, number>>("mochil-editor-widths", {}));
  /** Whether the columns follow the window. Off once a column is dragged; the column panel offers "Fit to window". */
  const [fitToWindow, setFitToWindow] = useState(() => loadLayout<boolean>("mochil-editor-fit", true));
  const [showColumns, setShowColumns] = useState(false);
  const [selected, setSelected] = useState<string[]>([]);
  const [anchor, setAnchor] = useState<string | undefined>(undefined);
  // The keyboard cursor: a row index into the loaded library and a column index into the shown columns.
  const [cursor, setCursor] = useState<{ row: number; col: number }>({ row: 0, col: 0 });
  // Which tool staged each field of each book. Staging used to be one anonymous map, so the review could not say where
  // a change came from and one tool's batch could not be applied without the other's. The map itself is unchanged —
  // apply, undo and the per-book path still take an EditorChanges — this only tags the entries.
  const [sources, setSources] = useState<EditorSources>({});
  // The task whose review is open, or undefined for "everything staged" (typed cells and bulk edits included).
  const [activeTask, setActiveTask] = useState<string | undefined>(undefined);
  // Dismissal is per tool now, and lives in taskDismissals.ts: the Tasks panel and the review panel's Discard both
  // write it, and the server's counts are what the row reports. Nothing here clears it on a library switch — a
  // dismissal belongs to the library it was made in, which is the bug that was just fixed.

  // The book whose match candidates are open. Choosing one stages its difference like any other edit.
  const [matchFor, setMatchFor] = useState<EditorBook | undefined>(undefined);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [collectionsOpen, setCollectionsOpen] = useState(false);
  const [toolsOpen, setToolsOpen] = useState(false);
  const [transferOpen, setTransferOpen] = useState(false);
  // Books whose edits should be forgotten ("reset to files") — sent as the request's reset list, not as a field change.
  const [resets, setResets] = useState<string[]>([]);
  const [changes, setChanges] = useState<EditorChanges>({});
  const [editing, setEditing] = useState<{ book: string; field: EditorField } | undefined>(undefined);
  const [draft, setDraft] = useState("");
  const [preview, setPreview] = useState<EditorChange[] | undefined>(undefined);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | undefined>(undefined);
  const [undoToken, setUndoToken] = useState<string | undefined>(undefined);

  useEffect(() => {
    saveLayout("mochil-editor-hidden", hidden);
  }, [hidden]);
  useEffect(() => {
    saveLayout("mochil-editor-order", order);
  }, [order]);
  useEffect(() => {
    saveLayout("mochil-editor-widths", widths);
  }, [widths]);
  useEffect(() => {
    saveLayout("mochil-editor-fit", fitToWindow);
  }, [fitToWindow]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const [viewport, setViewport] = useState(600);
  const [scrollTop, setScrollTop] = useState(0);
  const [tableWidth, setTableWidth] = useState(0);
  const { mode, setMode } = useThemeContext();
  const variant = useShellVariant();

  useEffect(() => {
    const timer = setTimeout(() => {
      setAppliedSearch(search.trim());
      setScrollTop(0);
      setCursor({ row: 0, col: 0 });
      // An id that is no longer in the table must not stay selected: with 10 books selected and a search showing 2,
      // "Move or copy" acted on all ten and the select-all box read as checked because 10 >= 2.
      setSelected([]);
      setAnchor(undefined);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const all = useQuery({
    queryKey: ["editor", "books", libraryId, appliedSearch, filter, collection, sort, dir],
    // The signal matters: typing in the search box starts a new six-request scan, and without it the previous one kept
    // running to the end, competing with the one the user is waiting for.
    queryFn: ({ signal }) =>
      fetchAllEditorBooks({ library: libraryId as string, q: appliedSearch, filter, collection, sort, dir }, signal),
    enabled: libraryId !== undefined,
    staleTime: 15_000,
  });

  // Measure the table once it exists, and on every resize after that. It has to wait for the query: the scroll area is
  // inside the table's Async, so on the first render there is nothing to measure — which is why this used to run against
  // a null ref and leave every row count and column width at its default.
  const tableReady = all.isSuccess && libraryId !== undefined;
  useEffect(() => {
    const element = scrollRef.current;
    if (!element) return;
    const measure = () => {
      setViewport(element.clientHeight);
      setTableWidth(element.clientWidth);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [tableReady]);

  const collections = useQuery({
    queryKey: ["editor", "collections", libraryId],
    queryFn: () => fetchCollections(libraryId as string),
    enabled: libraryId !== undefined,
    staleTime: 30_000,
  });

  /** The library the dropdown trigger shows; it has to read as a library even before the list has arrived. */
  const chosenLibraryName =
    libraries.data?.libraries.find((library) => library.id === libraryId)?.name ?? "Choose a library";

  /** What the collection filter is currently narrowed to, for the dropdown's accessible name. */
  const collectionName =
    collection === "" ? "All collections" : (collections.data?.find((entry) => entry.id === collection)?.name ?? collection);

  const books = useMemo(() => all.data?.items ?? [], [all.data]);
  const columns = useMemo(() => {
    const available = COLUMNS.filter((column) => !hidden.includes(column.id));
    if (order.length === 0) return available;
    const remaining = new Map<ColumnId, Column>(available.map((column) => [column.id, column]));
    const ordered: Column[] = [];
    for (const id of order) {
      const column = remaining.get(id as ColumnId);
      if (column !== undefined) {
        ordered.push(column);
        remaining.delete(id as ColumnId);
      }
    }
    return [...ordered, ...available.filter((column) => remaining.has(column.id))];
  }, [hidden, order]);
  const focus = books[cursor.row] ?? books[0];

  const chosen = libraries.data?.libraries.find((library) => library.id === libraryId);
  const hermes: HermesAbilities = libraries.data?.hermes ?? { helper: false, isbn: false, match: false };
  const editable = editableFieldsFor(chosen?.sourceType ?? "filesystem");
  const calibreOwned = chosen?.sourceType === "calibre";
  const pending = changeCount(changes) + resets.length;
  // The server's own limit (editorMaximumEdits): a control that is going to be refused is worse than one that says why.
  const overCap = pending > EDITOR_MAXIMUM_EDITS;
  const selectedBooks = useMemo(() => books.filter((book) => selected.includes(book.id)), [books, selected]);
  // The row asks "am I selected" once per render; `selected.includes(id)` is a scan of every selected id, so with a whole
  // library selected (5 000+) that was 33 scans of 5 000 per keystroke.
  const selectedSet = useMemo(() => new Set(selected), [selected]);

  // What the tools may stage into. A new array identity here made EditorTools recompute three full-library
  // match fill/replace scans (each normalising every title through NFKD) on every arrow key and every keystroke.
  // Per-tool dismissals are applied inside EditorTools, where each task's staging is computed: a book put aside by
  // the ISBN run is still offered to the match run, which is the whole point.

  const windowed = windowFor({ scrollTop, viewport, rowHeight: ROW_HEIGHT, count: books.length });
  const visible = books.slice(windowed.start, windowed.end);
  const widthOf = (column: Column) => widths[column.id] ?? column.defaultWidth;
  // A wide window gets wider columns, not a wider empty strip. The stored widths are minimums; whatever is left over
  // after the checkbox column is shared by the columns that hold prose, because those are the ones that truncate.
  // Which columns take the surplus: the ones holding prose. A status chip does not need to grow.
  const GROWABLE: readonly string[] = ["title", "authors", "series", "publisher", "description"];
  const growable = columns.filter((column) => GROWABLE.includes(column.id));
  const gap = 8;
  const used = 24 + columns.reduce((total, column) => total + widthOf(column), 0) + gap * Math.max(0, columns.length - 1);
  // Only spare room is shared out. A deficit is a scrollbar, never a squeeze: shrinking columns to fit made the resize
  // handle feel like it hit a wall, because the floor was fighting the drag.
  const spareRaw = fitToWindow && growable.length > 0 ? Math.floor((tableWidth - used) / growable.length) : 0;
  const spareFor = (column: Column) =>
    spareRaw > 0 && GROWABLE.includes(column.id) ? spareRaw : 0;
  const template = `24px ${columns.map((column) => `${widthOf(column) + spareFor(column)}px`).join(" ")}`;

  // Stable identities, so a row that did not change does not re-render. Everything a row receives is either a primitive,
  // the book object (stable while the list does not refetch), or one of these.
  const currentValue = useCallback(
    (book: EditorBook, field: EditorField): string => changes[book.id]?.[field] ?? valueOf(book, field),
    [changes],
  );

  const write = useCallback((book: EditorBook, field: EditorField, next: string) => {
    setChanges((current) => {
      const forBook = { ...(current[book.id] ?? {}) };
      if (valueOf(book, field) === next) delete forBook[field];
      else forBook[field] = next;
      const nextChanges = { ...current };
      if (Object.keys(forBook).length === 0) delete nextChanges[book.id];
      else nextChanges[book.id] = forBook;
      return nextChanges;
    });
  }, []);

  const startEditing = useCallback(
    (book: EditorBook, field: EditorField, initial?: string) => {
      const row = books.indexOf(book);
      if (row >= 0) setCursor({ row, col: columns.findIndex((column) => column.field === field) });
      setEditing({ book: book.id, field });
      setDraft(initial ?? changes[book.id]?.[field] ?? valueOf(book, field));
    },
    [books, columns, changes],
  );

  const cancelEdit = useCallback(() => setEditing(undefined), []);


  // The one handler that reads state changing on every keystroke, kept stable on purpose: a new identity per keystroke
  // would re-render every visible row while somebody types in one cell. The values come from a ref written after render.
  const fast = useRef({ editing, draft, books, cursor, columns });
  useEffect(() => {
    fast.current = { editing, draft, books, cursor, columns };
  });

  // Tab from inside a cell editor takes exactly the transition the grid's own Tab takes. Stable, because it is handed to
  // every row: an identity that changed with the cursor would re-render the whole table on every arrow key.
  const tabFromCell = useCallback((shift: boolean) => {
    const { cursor, books, columns } = fast.current;
    const intent = gridKey("Tab", cursor, { rows: books.length, columns: columns.length, page: 1 }, { shift });
    if (intent.kind === "move") setCursor(intent.cursor);
  }, []);

  const openMatch = useCallback((book: EditorBook) => setMatchFor(book), []);

  const commitCell = useCallback(() => {
    const { editing, draft, books } = fast.current;
    if (editing === undefined) return;
    const book = books.find((row) => row.id === editing.book);
    if (book !== undefined) write(book, editing.field, draft);
    setEditing(undefined);
    // The input unmounts from under the focus and the document falls back to <body>, which loses the keyboard's place
    // entirely — the next Tab then starts again from the top of the page. Put the focus back on the table, but only if
    // nothing else has claimed it (clicking another cell must keep that cell).
    requestAnimationFrame(() => {
      const active = document.activeElement;
      if (active === null || active === document.body) scrollRef.current?.focus();
    });
  }, [write]);

  const sortKeyOf = (column: Column) => (column.field !== undefined ? sortKeyFor(column.field) : column.id);

  function toggleSort(column: Column) {
    const key = sortKeyOf(column);
    if (sort === key) setDir(dir === "asc" ? "desc" : "asc");
    else {
      setSort(key);
      setDir("asc");
    }
    setScrollTop(0);
    setCursor({ row: 0, col: 0 });
    setSelected([]);
    setAnchor(undefined);
  }

  // Click selects one row; Shift-click selects the range from the last click, the way the legacy editor and Calibre do.
  const toggleSelected = useCallback((book: EditorBook, range = false) => {
    setSelected((current) => {
      if (range && anchor !== undefined) {
        const ids = books.map((row) => row.id);
        const from = ids.indexOf(anchor);
        const to = ids.indexOf(book.id);
        if (from >= 0 && to >= 0) {
          const [low, high] = from <= to ? [from, to] : [to, from];
          const adding = !current.includes(book.id);
          const next = new Set(current);
          for (const id of ids.slice(low, high + 1)) {
            if (adding) next.add(id);
            else next.delete(id);
          }
          return [...next];
        }
      }
      return current.includes(book.id) ? current.filter((id) => id !== book.id) : [...current, book.id];
    });
    setAnchor(book.id);
  }, [anchor, books]);

  /**
   * Splits a staged set into requests the server will accept. One request carries at most EDITOR_MAXIMUM_EDITS books;
   * sending more is refused outright, which is how a 2 859-book review ended with a message and no panel.
   */
  function batchesOf(all: EditorChanges, size: number): EditorChanges[] {
    const ids = Object.keys(all);
    const out: EditorChanges[] = [];
    for (let start = 0; start < ids.length; start += size) {
      const batch: EditorChanges = {};
      for (const id of ids.slice(start, start + size)) batch[id] = all[id] ?? {};
      out.push(batch);
    }
    return out;
  }

  /** A server preview with each change labelled by the tool that proposed it — the panel's "From Find ISBNs". */
  function labelled(preview: EditorChange[], from: EditorSources): EditorChange[] {
    return preview.map((change) => {
      const source = from[change.book]?.[change.field as EditorField];
      return source === undefined ? change : { ...change, source: source.title };
    });
  }

  async function review() {
    if (libraryId === undefined) return;
    setBusy(true);
    setMessage(undefined);
    try {
      setActiveTask(undefined);
      setPreview(labelled(await previewChanges(libraryId, changes, resets), sources));
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Those changes could not be checked.");
    } finally {
      setBusy(false);
    }
  }

  /**
   * Review one task's results: stage what that tool proposed, tag it with the tool, then ask the server for the diff of
   * just those fields. Another task's staged work is left where it is — it is reviewed on its own row, and checked out
   * on its own, so a 251-row title diff and a 41-row ISBN fill can both be dealt with in one sitting.
   */
  async function reviewTask(tool: string, title: string, incoming: EditorChanges) {
    if (libraryId === undefined) return;
    const merged = mergeChanges(changes, incoming);
    const tagged = tagChanges(sources, incoming, { tool, title });
    setChanges(merged);
    setSources(tagged);
    const subset = changesFrom(merged, tagged, tool);
    if (changeCount(subset) === 0) {
      setMessage(`Nothing to change from ${title}: those books already say what it proposes.`);
      return;
    }
    setBusy(true);
    setMessage(undefined);
    try {
      setActiveTask(tool);
      // The preview is asked for in batches, because one request carries at most EDITOR_MAXIMUM_EDITS books and a tool
      // that found 2 859 would otherwise be answered with "Edit between 1 and 2000 books at a time" — the row said
      // Review, the review never opened, and nothing on screen explained why.
      const rows: EditorChange[] = [];
      for (const batch of batchesOf(subset, EDITOR_MAXIMUM_EDITS)) {
        // No resets here: a reset is "forget every Mochil edit of this book", which is a bulk-edit decision, not a tool's.
        rows.push(...labelled(await previewChanges(libraryId, batch, []), tagged));
      }
      setPreview(rows);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Those changes could not be checked.");
    } finally {
      setBusy(false);
    }
  }

  async function apply() {
    if (libraryId === undefined) return;
    // A task's review applies that task and nothing else. Everything staged by hand still applies as one batch.
    const applying = activeTask === undefined ? changes : changesFrom(changes, sources, activeTask);
    const applyingResets = activeTask === undefined ? resets : [];
    setBusy(true);
    setMessage(undefined);
    try {
      // Batched for the same reason the preview is: the server saves at most EDITOR_MAXIMUM_EDITS books per request.
      // Resets ride on the first batch — they are a whole-book decision, not a per-batch one.
      let changed = 0;
      let undo = "";
      let batches = 0;
      for (const batch of batchesOf(applying, EDITOR_MAXIMUM_EDITS)) {
        const result = await applyChanges(libraryId, batch, batches === 0 ? applyingResets : []);
        changed += result.changed;
        undo = result.undo || undo;
        batches += 1;
      }
      setPreview(undefined);
      setChanges((current) => withoutChanges(current, applying));
      setSources((current) => withoutSources(current, applying));
      if (activeTask === undefined) setResets([]);
      setActiveTask(undefined);
      setSelected([]);
      setUndoToken(undo);
      setMessage(
        `Saved ${changed} ${changed === 1 ? "change" : "changes"}` +
          (batches > 1 ? ` in ${batches} batches; undo covers the last one.` : "."),
      );
      // The Tasks rows count what is left to do, so they are wrong the moment something is saved. Without this the
      // panel kept its old counts: the row still read "2 859 found" after all 2 859 had been written, which reads as
      // "did that work, or is there something else to review?"
      await Promise.all([all.refetch(), queryClient.invalidateQueries({ queryKey: ["editor", "tasks"] })]);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Those changes could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  /**
   * Apply one book's staged changes and leave the rest pending — the per-book button in the review panel. The server
   * takes a changes map, so "one book" is simply a map with one entry; nothing about the request changes.
   */
  async function applyBook(bookId: string) {
    if (libraryId === undefined) return;
    const full = changes[bookId];
    if (full === undefined) return;
    // In a task's review, "this book" means this task's fields on it: the other task's staged work is still waiting on
    // its own row, and writing it here would save something the person was never shown.
    const staged = activeTask === undefined ? full : (changesFrom({ [bookId]: full }, sources, activeTask)[bookId] ?? {});
    if (Object.keys(staged).length === 0) return;
    setBusy(true);
    setMessage(undefined);
    try {
      // Only this book's own reset, never the batch's: `reset` means "forget every Mochil edit of these books", so
      // passing the whole list here silently reset the other 19 books the user was still deciding about.
      const result = await applyChanges(libraryId, { [bookId]: staged }, resets.filter((id) => id === bookId));
      setResets((current) => current.filter((id) => id !== bookId));
      setChanges((current) => withoutChanges(current, { [bookId]: staged }));
      setSources((current) => withoutSources(current, { [bookId]: staged }));
      setPreview((current) => current?.filter((change) => change.book !== bookId) ?? current);
      setUndoToken(result.undo);
      setMessage(`Saved ${result.changed} ${result.changed === 1 ? "change" : "changes"} for one book.`);
      await Promise.all([all.refetch(), queryClient.invalidateQueries({ queryKey: ["editor", "tasks"] })]);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Those changes could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  /** The tools that proposed a book's staged fields. A typed cell has none, so discarding it hides it from no run. */
  function toolsBehind(bookId: string): string[] {
    if (activeTask !== undefined) return [activeTask];
    return [...new Set(Object.values(sources[bookId] ?? {}).map((source) => source.tool))];
  }

  /** Put a book aside for one tool, and say so if the server refuses — the local set moves first, so a silent failure
   *  would read as a dismissal that quietly came back. */
  function dismissFor(tool: string, books: string[]): void {
    if (libraryId === undefined) return;
    void dismissBooks(libraryId, tool, books).catch((error: unknown) => {
      setMessage(error instanceof Error ? error.message : "That could not be set aside.");
    });
  }

  /** The part of a book's staged work that the review on screen is about: one task's fields, or all of them. */
  function stagedFor(bookId: string): Partial<Record<EditorField, string>> {
    const full = changes[bookId] ?? {};
    if (activeTask === undefined) return full;
    return changesFrom({ [bookId]: full }, sources, activeTask)[bookId] ?? {};
  }

  /** Drop one book's staged changes without writing anything, and stop that tool offering it again. */
  function discardBook(bookId: string) {
    const staged = stagedFor(bookId);
    for (const tool of toolsBehind(bookId)) dismissFor(tool, [bookId]);
    const fields = new Set(Object.keys(staged));
    setChanges((current) => withoutChanges(current, { [bookId]: staged }));
    setSources((current) => withoutSources(current, { [bookId]: staged }));
    // A book staged to be reset ("back to what the files say") is staged in `resets`, not in `changes`. Dropping only
    // the changes left the reset behind, so a book the user had just discarded was still the one thing Apply wrote.
    setResets((current) => current.filter((id) => id !== bookId));
    setPreview((current) => current?.filter((change) => !(change.book === bookId && fields.has(change.field))) ?? current);
  }

  /** Drop every staged change at once — the batch is often "none of these", and 38 clicks is not an answer. */
  function discardAll() {
    // Whatever is on screen: a task's review discards that task's books, the unfiltered one discards them all.
    const books = preview !== undefined ? [...new Set(preview.map((change) => change.book))] : Object.keys(changes);
    // One request per tool rather than one per book: 38 books from a single tool is one dismissal, not 38.
    const byTool = new Map<string, string[]>();
    const dropped: EditorChanges = {};
    for (const bookId of books) {
      for (const tool of toolsBehind(bookId)) byTool.set(tool, [...(byTool.get(tool) ?? []), bookId]);
      const staged = stagedFor(bookId);
      if (Object.keys(staged).length > 0) dropped[bookId] = staged;
    }
    for (const [tool, ids] of byTool) dismissFor(tool, ids);
    setChanges((current) => withoutChanges(current, dropped));
    setSources((current) => withoutSources(current, dropped));
    if (activeTask === undefined) setResets((current) => current.filter((id) => !books.includes(id)));
    setPreview(undefined);
    setActiveTask(undefined);
  }

  async function undo() {
    if (libraryId === undefined || undoToken === undefined) return;
    setBusy(true);
    try {
      await undoChanges(libraryId, undoToken);
      setUndoToken(undefined);
      setMessage("Undone.");
      await Promise.all([all.refetch(), queryClient.invalidateQueries({ queryKey: ["editor", "tasks"] })]);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "That undo could not be applied.");
    } finally {
      setBusy(false);
    }
  }

  const dragging = useRef<{ id: string; x: number; moved: boolean } | null>(null);
  const resizing = useRef<{ id: string; x: number; width: number } | null>(null);
  const justDragged = useRef(false);

  useEffect(() => {
    function onPointerMove(event: PointerEvent) {
      const resize = resizing.current;
      if (resize !== null) {
        setWidths((current) => ({ ...current, [resize.id]: Math.max(48, resize.width + (event.clientX - resize.x)) }));
        return;
      }
      const drag = dragging.current;
      if (drag === null) return;
      if (!drag.moved && Math.abs(event.clientX - drag.x) < 5) return;
      drag.moved = true;
      const under = document.elementFromPoint(event.clientX, event.clientY)?.closest("[data-column-id]");
      const target = under?.getAttribute("data-column-id");
      if (target === null || target === undefined || target === drag.id) return;
      setOrder((current) => {
        const ids = columns.map((column) => column.id as string);
        const from = ids.indexOf(drag.id);
        const to = ids.indexOf(target);
        if (from < 0 || to < 0) return current;
        const next = [...ids];
        next.splice(from, 1);
        next.splice(to, 0, drag.id);
        return next;
      });
    }
    function onPointerUp() {
      if (dragging.current?.moved === true) {
        justDragged.current = true;
        setTimeout(() => {
          justDragged.current = false;
        }, 0);
      }
      resizing.current = null;
      dragging.current = null;
    }
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", onPointerUp);
    return () => {
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", onPointerUp);
    };
  }, [columns]);

  const selectAll = () => setSelected(books.map((book) => book.id));

  /**
   * Every key the table answers, on the table itself.
   *
   * A window listener read through a ref was how this started, and it made the editor's keyboard page-wide: Tab could
   * not reach the toolbar or leave the table at all, Space selected a book instead of scrolling, and with a dialog open
   * the keys typed into the table behind it. A grid's keys belong to the grid. The transitions live in keys.ts, so Tab
   * leaving at the edges and Tab wrapping onto the next row are testable without a DOM.
   */
  function onGridKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    // An open cell editor owns every key except the ones its input handles itself.
    if (editing !== undefined || preview !== undefined) return;
    const target = event.target instanceof Element ? event.target : null;
    if (target?.closest("input, textarea, select")) return;
    // A cell is a button only so it can be clicked; anything else that is a button is chrome and owns its own keys.
    if (target?.closest("button:not([data-editor-cell])")) return;

    const visible = Math.floor((scrollRef.current?.clientHeight ?? 400) / ROW_HEIGHT);
    const intent = gridKey(
      event.key,
      cursor,
      { rows: books.length, columns: columns.length, page: Math.max(1, visible - 1) },
      { shift: event.shiftKey, ctrl: event.ctrlKey, meta: event.metaKey },
    );

    // The edge of the table is an exit, not a wall: preventDefault here would trap the keyboard in a 5000-row grid.
    if (intent.kind === "leave") return;
    if (intent.kind === "select-all") {
      event.preventDefault();
      selectAll();
      return;
    }
    if (intent.kind === "move") {
      event.preventDefault();
      const reached = books[intent.cursor.row];
      // Shift carries the selection, the way Calibre's book list does: put the cursor somewhere and hold Shift to take
      // the rows in between.
      if (event.shiftKey && intent.cursor.row !== cursor.row && reached !== undefined) toggleSelected(reached, true);
      setCursor(intent.cursor);
      return;
    }

    const book = books[cursor.row];
    const field = columns[cursor.col]?.field;
    if ((event.key === "Enter" || event.key === "F2") && book !== undefined && field !== undefined) {
      event.preventDefault();
      if (editable.includes(field)) startEditing(book, field);
      return;
    }
    if (event.key === " " && book !== undefined) {
      event.preventDefault();
      toggleSelected(book, event.shiftKey);
      return;
    }
    // Typing a letter starts editing that cell with the letter, the way a spreadsheet does.
    if (
      event.key.length === 1 &&
      !event.ctrlKey &&
      !event.metaKey &&
      !event.altKey &&
      book !== undefined &&
      field !== undefined &&
      editable.includes(field)
    ) {
      event.preventDefault();
      startEditing(book, field, event.key);
    }
  }

  // The table takes the keyboard when the library arrives, the way Calibre's book list does. Moving the keys onto the
  // grid means they only work where the grid has focus, and an editor that ignores the arrow keys until you click a row
  // looks broken. Only when nothing else has focus: a restored tab stop must not be yanked away from under the user.
  useEffect(() => {
    if (books.length === 0) return;
    const active = document.activeElement;
    if (active === null || active === document.body) scrollRef.current?.focus();
  }, [books.length]);

  // Follow the cursor: scroll the focused row into view, moving as little as possible.
  useEffect(() => {
    const element = scrollRef.current;
    if (element === null) return;
    const top = cursor.row * ROW_HEIGHT;
    const header = 34;
    if (top < element.scrollTop + header) element.scrollTop = Math.max(0, top - header);
    else if (top + ROW_HEIGHT > element.scrollTop + element.clientHeight) element.scrollTop = top + ROW_HEIGHT - element.clientHeight;

    // Sideways as well: the columns can be wider than the window, and a 1px ring that has scrolled out of view is not a
    // cursor anyone can see. Read the drawn cell's box rather than doing the column arithmetic a second time.
    const cell = element.querySelector<HTMLElement>(`[id="cell-${cursor.row}-${cursor.col}"]`);
    if (cell !== null) cell.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [cursor.row, cursor.col]);

  // A spreadsheet with resizable columns and a keyboard cursor is not a phone screen. Say so rather than render a
  // table six hundred pixels wide that nobody can drag.
  if (variant === "phone") {
    return (
      <Page>
        <EmptyState message="The metadata editor needs a wider screen. Open Mochil on a computer to edit the library." />
      </Page>
    );
  }

  if (!editorEnabled) {
    return (
      <Page>
        <EmptyState message="The library editor is switched off for this server. Open Settings → Modules to turn it on." />
      </Page>
    );
  }

  return (
    <Page wide>
      <Stack gap={12}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "var(--space-12)" }}>
          <Inline gap={12} align="center" wrap>
            <span title="Arrow keys move · Enter edits a cell · Space selects · Shift-click for a range">
              <Heading size="22">Metadata editor</Heading>
            </span>
            {calibreOwned && (
              <span
                title={`This library is managed by Calibre, so ${CALIBRE_OWNED_FIELDS.map((field) => FIELD_LABELS[field].toLowerCase()).join(", ")} are read-only here — Calibre would overwrite them on its next save.`}
              >
                <Text size="11" muted>
                  Calibre library · {CALIBRE_OWNED_FIELDS.length} fields read-only
                </Text>
              </span>
            )}
          </Inline>
          {/* The editor is the one screen where the mode is a working decision rather than a taste: a wide table is
              easier to read in light on a bright day, and easier in dark at night. It flips the app's own mode, so
              the rest of Mochil follows instead of the screen drifting out of step with the settings page. */}
          <IconButton
            icon={mode === "dark" ? Sun : Moon}
            label={mode === "dark" ? "Switch to light mode" : "Switch to dark mode"}
            onClick={() => setMode(mode === "dark" ? "light" : "dark")}
          />
        </div>


        <Toolbar>
          <Async query={libraries}>
            {(data) =>
              data.libraries.length === 0 ? (
                <Text muted>No book libraries yet — add a folder of books first.</Text>
              ) : (
                <Menu
                  icon={LibraryBig}
                  text={chosenLibraryName}
                  label={`Library: ${chosenLibraryName}`}
                  activeIds={[libraryId ?? ""]}
                  items={data.libraries.map((library) => ({ id: library.id, label: library.name }))}
                  onSelect={(id) => {
                    setChosenLibrary(id);
                    setCollection("");
                    setSelected([]);
                    setCursor({ row: 0, col: 0 });
                    setChanges({});
                    // Staged work and its review belong to the library they were made in; the dismissals below do not
                    // belong to this screen at all — they live per library, in taskDismissals.ts.
                    setSources({});
                    setPreview(undefined);
                    setActiveTask(undefined);
                    // Dismissals are NOT cleared here any more. They are kept per library, so switching shelves already
                    // shows the right answer for the shelf you land on — and clearing them here meant every reload (a
                    // rebuild, a new tab) wiped the decision the person had just made, which is how the same 41 found
                    // ISBNs kept coming back.
                    setMessage(undefined);
                    setUndoToken(undefined);
                    setScrollTop(0);
                  }}
                />
              )
            }
          </Async>
          <div style={{ flex: "1 1 320px", minWidth: 0 }}>
            <TextField
              label="Search the library"
              hideLabel
              icon={Search}
              placeholder="Search the library"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </div>
          <div style={{ display: "flex", gap: "var(--space-8)", flexWrap: "wrap", alignItems: "center" }}>
            <Menu
              icon={ListFilter}
              text={FILTERS.find((entry) => entry.value === filter)?.label ?? "All books"}
              label={`Filter: ${FILTERS.find((entry) => entry.value === filter)?.label ?? "All books"}`}
              align="end"
              activeIds={[filter]}
              items={FILTERS.map((entry) => ({ id: entry.value, label: entry.label }))}
              onSelect={(id) => {
                setFilter(id);
                setScrollTop(0);
                setCursor({ row: 0, col: 0 });
                setSelected([]);
                setAnchor(undefined);
              }}
            />
            {(collections.data?.length ?? 0) > 0 && (
              <Menu
                icon={Folders}
                text={collectionName === "All collections" ? "Collections" : collectionName}
                label={`Collection filter: ${collectionName}`}
                align="end"
                activeIds={[collection]}
                items={[
                  { id: "", label: "All collections", section: "Collection" },
                  ...(collections.data ?? []).map((entry) => ({ id: entry.id, label: `${entry.name} (${entry.count})` })),
                ]}
                onSelect={(id) => {
                  setCollection(id);
                  setScrollTop(0);
                  setCursor({ row: 0, col: 0 });
                  setSelected([]);
                  setAnchor(undefined);
                }}
              />
            )}
            <Button variant="secondary" onClick={() => setToolsOpen(!toolsOpen)} aria-expanded={toolsOpen}>
              Tools
            </Button>
            <IconButton
              icon={Columns3}
              label={showColumns ? "Hide the column picker" : "Show the column picker"}
              onClick={() => setShowColumns(!showColumns)}
            />
          </div>
        </Toolbar>


        {matchFor !== undefined && libraryId !== undefined && (
          <MatchDialog
            libraryId={libraryId}
            book={matchFor}
            onClose={() => setMatchFor(undefined)}
            onUse={(candidate) => {
              // The chosen candidate becomes the book's match for one comparison: the same replace rules as the bulk
              // mode, so what gets staged is exactly what the review panel would show for any other match.
              const fields = matchReplace({ ...matchFor, match: candidate, matchClear: true }, editable);
              const count = Object.keys(fields).length;
              // Nothing to stage means no entry: an empty object here counted as "staged" for Discard all, which then
              // set the book aside as though the user had reviewed and rejected it.
              if (count > 0) setChanges((current) => mergeChanges(current, { [matchFor.id]: fields }));
              setMatchFor(undefined);
              setMessage(
                count === 0
                  ? `Nothing to change: “${candidate.title}” already agrees with every field it owns.`
                  : `Staged ${count} field${count === 1 ? "" : "s"} for ${matchFor.title} from “${candidate.title}”. Review to see the diff.`,
              );
            }}
          />
        )}

        {libraryId !== undefined && all.data !== undefined && (
          <EditorTools
            libraryId={libraryId}
            books={books}
            selected={selected}
            open={toolsOpen}
            editable={editable}
            hermes={hermes}
            pending={pending}
            calibre={chosen?.sourceType === "calibre"}
            onReviewTask={(tool, title, incoming) => void reviewTask(tool, title, incoming)}
            refetch={all.refetch}
          />
        )}

        {showColumns && (
          <Inline gap={12} wrap>
            {COLUMNS.filter((column) => column.fixed !== true).map((column) => (
              <label key={column.id} style={{ display: "flex", gap: "var(--space-8)", alignItems: "center", fontSize: "var(--font-size-12)" }}>
                <input
                  type="checkbox"
                  checked={!hidden.includes(column.id)}
                  onChange={() =>
                    setHidden((current) =>
                      current.includes(column.id) ? current.filter((id) => id !== column.id) : [...current, column.id],
                    )
                  }
                />
                {column.label}
              </label>
            ))}
            {!fitToWindow && (
              <Button
                variant="secondary"
                onClick={() => {
                  // Fitting means going back to the starting widths *and* following the window again; keeping the
                  // dragged widths and merely re-flagging did nothing visible when a column was already too wide.
                  setWidths({});
                  setFitToWindow(true);
                }}
              >
                Reset columns and fit
              </Button>
            )}
          </Inline>
        )}

        {message !== undefined && (
          // A live region: applying, discarding and failing are all silent to anyone using a screen reader otherwise,
          // and "Saved 12 changes" is the one thing this screen must not whisper.
          <div role="status" style={{ display: "flex", alignItems: "center", gap: "var(--space-12)", flexWrap: "wrap" }}>
            <Text size="13">{message}</Text>
            {undoToken !== undefined && (
              <Button variant="secondary" onClick={() => void undo()} disabled={busy}>
                Undo
              </Button>
            )}
          </div>
        )}

        {selected.length > 0 && (
          <Inline gap={8} align="center" wrap>
            <Text size="12">{selected.length} selected</Text>
            <Button variant="secondary" onClick={() => setBulkOpen(true)}>
              Bulk edit…
            </Button>
            <Button variant="secondary" onClick={() => setCollectionsOpen(true)}>
              Collections…
            </Button>
            {(all.data?.transferTo.length ?? 0) > 0 && (
              <Button variant="secondary" onClick={() => setTransferOpen(true)}>
                Move or copy…
              </Button>
            )}
            <Button variant="secondary" onClick={() => setSelected([])}>
              Clear selection
            </Button>
          </Inline>
        )}

        {libraryId !== undefined && (
          <Async query={all}>
            {(data) => (
              <Stack gap={12}>
                <div
                  ref={scrollRef}
                  // Focusable so the arrow keys have somewhere to belong, and the only place they are answered: keys
                  // typed here drive the grid, keys typed anywhere else on the page stay with the page.
                  tabIndex={0}
                  role="grid"
                  aria-label="Books"
                  aria-rowcount={books.length + 1}
                  aria-colcount={columns.length + 1}
                  aria-multiselectable="true"
                  // The cursor is a value in state, not DOM focus, so assistive tech is told where it is the way the
                  // APG prescribes: one stable focus target and a pointer at the cell it is on.
                  aria-activedescendant={`cell-${cursor.row}-${cursor.col}`}
                  onKeyDown={onGridKeyDown}
                  onScroll={(event) => setScrollTop(event.currentTarget.scrollTop)}
                  onMouseDown={(event) => {
                    // Clicking the table's own background takes focus, so the keys land here without having to click a
                    // cell first (clicking a cell would open it for editing instead).
                    if (event.target === event.currentTarget) event.currentTarget.focus();
                  }}
                  style={{ height: "clamp(340px, 70vh, 800px)", overflow: "auto", border: "1px solid var(--color-line)", borderRadius: "var(--radius-12)", position: "relative" }}
                >
                  <div style={{ position: "sticky", top: 0, zIndex: 2, background: "var(--color-bg)", minWidth: 760 }} role="rowgroup">
                    <div role="row" aria-rowindex={1} style={{ display: "grid", gridTemplateColumns: template, gap: "var(--space-8)", padding: "6px 8px", borderBottom: "1px solid color-mix(in srgb, var(--color-text) 18%, transparent)", background: "var(--color-raised)", fontSize: "var(--font-size-12)", fontWeight: 600 }}>
                      <div role="columnheader" aria-colindex={1}>
                        <input
                          type="checkbox"
                          aria-label="Select every book in the library"
                          checked={books.length > 0 && selected.length >= books.length}
                          onChange={() => setSelected(selected.length >= books.length ? [] : books.map((book) => book.id))}
                        />
                      </div>
                      {columns.map((column) => (
                        <HeaderCell
                          key={column.id}
                          column={column}
                          active={sort === sortKeyOf(column)}
                          dir={dir}
                          onSort={() => {
                            if (!justDragged.current) toggleSort(column);
                          }}
                          onDragStart={(event) => {
                            dragging.current = { id: column.id, x: event.clientX, moved: false };
                          }}
                          onResizeStart={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            // From here the widths are the user's, not the window's. Bake what is on screen into real
                            // widths first: otherwise the drag would start from a narrower base than it looks and the
                            // column would lag behind the pointer by the amount the window had added.
                            if (fitToWindow) {
                              setWidths(Object.fromEntries(columns.map((entry) => [entry.id, widthOf(entry) + spareFor(entry)])));
                              setFitToWindow(false);
                            }
                            resizing.current = {
                              id: column.id,
                              x: event.clientX,
                              // The width the user can see, not the width that was stored: with a fitted layout those
                              // differ by the shared spare, and starting from the stored one made the column jump
                              // sideways on the first pixel of movement.
                              width: widthOf(column) + spareFor(column),
                            };
                          }}
                          onResizeReset={() =>
                            setWidths((current) => {
                              const next = { ...current };
                              delete next[column.id];
                              return next;
                            })
                          }
                        />
                      ))}
                    </div>
                  </div>

                  <div style={{ height: books.length * ROW_HEIGHT, position: "relative", minWidth: 760 }}>
                    <div style={{ position: "absolute", top: windowed.before, left: 0, right: 0 }}>
                      {visible.map((book, index) => (
                        <Row
                          key={book.id}
                          book={book}
                          rowIndex={windowed.start + index}
                          cursorCol={cursor.row === windowed.start + index ? cursor.col : -1}
                          onTab={tabFromCell}
                          columns={columns}
                          template={template}
                          editingField={editing?.book === book.id ? editing.field : undefined}
                          draft={draft}
                          onDraft={setDraft}
                          onStart={startEditing}
                          onCommit={commitCell}
                          onCancel={cancelEdit}
                          value={currentValue}
                          pendingFor={changes[book.id] ?? NO_CHANGES}
                          editable={editable}
                          reviewing={preview !== undefined}
                          selected={selectedSet.has(book.id)}
                          onSelect={toggleSelected}
                          onMatch={calibreOwned ? undefined : openMatch}
                          focused={focus?.id === book.id}
                        />
                      ))}
                    </div>
                  </div>
                </div>

                <Inline gap={8} align="center" wrap>
                  <Text size="11" muted>
                    {data.total === 0
                      ? "No books match"
                      : `${data.total}${appliedSearch === "" && filter === "" ? ` of ${data.libraryTotal}` : ""} books — scroll the whole library`}
                  </Text>
                  {/* One line, because the keys are the editor's real interface and Calibre's are what people already
                      have in their fingers: arrows move, Tab is the next field, Enter or F2 edits, Space selects. */}
                  <Text size="11" muted>
                    Arrows move · Tab next field · Enter/F2 edit · Space select · Shift+arrows select a range · Ctrl+A all
                  </Text>
                  <div style={{ flex: 1 }} />
                  <Text size="11" muted>
                    {pending === 0 ? "No pending changes" : `${pending} pending`}
                    {overCap ? ` · ${EDITOR_MAXIMUM_EDITS} per save, stage the rest after` : ""}
                  </Text>
                  <Button
                    variant="primary"
                    disabled={pending === 0 || busy || overCap}
                    onClick={() => void review()}
                    title={overCap ? `The server saves at most ${EDITOR_MAXIMUM_EDITS} books at a time. Apply this batch, then stage the rest.` : undefined}
                  >
                    Review changes
                  </Button>
                </Inline>

                {/* An empty review panel is a panel with nothing in it: after a save there is no preview to close by hand,
                    and leaving it open read as "there is still something here" under a bar that said "No pending changes". */}
                {preview !== undefined && preview.length > 0 && (
                  <ReviewPanel
                    changes={preview}
                    books={books}
                    busy={busy}
                    onApply={() => void apply()}
                    onCancel={() => setPreview(undefined)}
                    onApplyBook={(bookId) => void applyBook(bookId)}
                    onDiscardBook={discardBook}
                    onDiscardAll={discardAll}
                  />
                )}

                {focus !== undefined && (
                  <div style={{ borderTop: "1px solid var(--color-line)", paddingTop: "var(--space-16)" }}>
                    <Stack gap={12}>
                      <Inline gap={8} align="center" wrap>
                        <Heading size="16">{focus.title}</Heading>
                        {!calibreOwned && (
                          // The one place a wrong match can be argued with: the ranked candidates, and the words that
                          // were searched. Nothing here writes; choosing stages, and the review panel shows the diff.
                          <Button variant="secondary" onClick={() => setMatchFor(focus)} title={focus.match === undefined ? "Search Hardcover for this book and see every candidate." : `Matched to ${focus.match.title} at ${Math.round(focus.match.score * 100)}% — see the other candidates.`}>
                            {focus.match === undefined ? "Match…" : `Match… ${Math.round(focus.match.score * 100)}%`}
                          </Button>
                        )}
                        {focus.edited && (
                          <Text size="11" muted>
                            edited
                          </Text>
                        )}
                        <div style={{ flex: 1 }} />
                        <Text size="11" muted>
                          {[focus.format, focus.path].filter((part) => part !== "").join(" · ")}
                        </Text>
                      </Inline>
                      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))", gap: "var(--space-12)" }}>
                        {(["title", "authors", "series", "series_index", "published", "publisher", "language", "tags", "identifiers", "title_sort", "author_sort"] as EditorField[]).map((field) => {
                          const locked = !editable.includes(field);
                          return (
                            <label key={field} style={{ display: "block" }}>
                              <Text size="11" muted>
                                {FIELD_LABELS[field]}
                                {locked ? " · Calibre's" : ""}
                              </Text>
                              <input
                                value={currentValue(focus, field)}
                                disabled={locked}
                                onChange={(event) => write(focus, field, event.target.value)}
                                style={{
                                  width: "100%",
                                  boxSizing: "border-box",
                                  padding: "6px 8px",
                                  border: "1px solid var(--color-line)",
                                  borderRadius: "var(--radius-8)",
                                  background: locked ? "transparent" : "var(--color-bg)",
                                  color: "var(--color-text)",
                                  font: "inherit",
                                  fontSize: "var(--font-size-12)",
                                  opacity: locked ? 0.6 : 1,
                                }}
                              />
                            </label>
                          );
                        })}
                        <label style={{ display: "block", gridColumn: "1 / -1" }}>
                          <Text size="11" muted>
                            Description
                          </Text>
                          <textarea
                            value={currentValue(focus, "description")}
                            disabled={!editable.includes("description")}
                            onChange={(event) => write(focus, "description", event.target.value)}
                            rows={4}
                            style={{
                              width: "100%",
                              boxSizing: "border-box",
                              padding: "8px",
                              border: "1px solid var(--color-line)",
                              borderRadius: "var(--radius-8)",
                              background: "var(--color-bg)",
                              color: "var(--color-text)",
                              font: "inherit",
                              fontSize: "var(--font-size-12)",
                              resize: "vertical",
                            }}
                          />
                        </label>
                      </div>
                      <Text size="11" muted>
                        {[
                          focus.rating > 0 ? `rated ${focus.rating} of 5` : "unrated",
                          focus.issues.length > 0 ? `${focus.issues.length} issue${focus.issues.length === 1 ? "" : "s"}: ${focus.issues.join(", ")}` : "no issues",
                          `published ${yearOf(focus.published) || "unknown"}`,
                          focus.words > 0 ? `${focus.words.toLocaleString()} words` : "not counted yet",
                          focus.progress > 0 ? `${Math.round(focus.progress * 100)}% read` : "unread",
                        ].join(" · ")}
                      </Text>
                    </Stack>
                  </div>
                )}
              </Stack>
            )}
          </Async>
        )}
      </Stack>

      {transferOpen && libraryId !== undefined && (
        <TransferDialog
          libraryId={libraryId}
          ids={selected}
          targets={all.data?.transferTo ?? []}
          blocked={
            pending > 0
              ? "Save or discard your changes first: a book that moves takes its saved changes with it."
              : ""
          }
          onClose={() => setTransferOpen(false)}
          onDone={(text) => {
            setTransferOpen(false);
            setMessage(text);
            setSelected([]);
            void all.refetch();
          }}
        />
      )}

      {collectionsOpen && libraryId !== undefined && (
        <CollectionsDialog
          libraryId={libraryId}
          ids={selected}
          collections={collections.data ?? []}
          onClose={() => setCollectionsOpen(false)}
          onDone={(text) => {
            setCollectionsOpen(false);
            setMessage(text);
            setSelected([]);
            void collections.refetch();
            void all.refetch();
          }}
        />
      )}

      {bulkOpen && (
        <BulkEditDialog
          books={selectedBooks}
          editable={editable}
          onCancel={() => setBulkOpen(false)}
          onApply={(incoming, reset) => {
            if (reset) {
              setResets((current) => Array.from(new Set([...current, ...selected])));
              setChanges((current) => {
                const next = { ...current };
                for (const id of selected) delete next[id];
                return next;
              });
            } else {
              setChanges((current) => mergeChanges(current, incoming));
              setResets((current) => current.filter((id) => !selected.includes(id)));
            }
            setBulkOpen(false);
            setMessage(undefined);
          }}
        />
      )}
    </Page>
  );
}

function HeaderCell({
  column,
  active,
  dir,
  onSort,
  onDragStart,
  onResizeStart,
  onResizeReset,
}: {
  column: Column;
  active: boolean;
  dir: "asc" | "desc";
  onSort: () => void;
  onDragStart: (event: React.PointerEvent<HTMLDivElement>) => void;
  onResizeStart: (event: React.PointerEvent<HTMLSpanElement>) => void;
  onResizeReset: () => void;
}) {
  return (
    <div
      data-column-id={column.id}
      role="columnheader"
      // A header in a grid says which way the table is sorted, or that it could be.
      aria-sort={active ? (dir === "asc" ? "ascending" : "descending") : "none"}
      onPointerDown={onDragStart}
      style={{
        position: "relative",
        minWidth: 0,
        display: "flex",
        alignItems: "center",
        paddingRight: "var(--space-4)",
      }}
    >
      <button
        type="button"
        onClick={onSort}
        title="Click to sort · drag to move"
        style={{
          all: "unset",
          cursor: "pointer",
          flex: 1,
          minWidth: 0,
          // The header row is 34px tall, but `all: unset` left the button at its text height (17px), so the thing you
          // aim at is half the space it looks like it fills.
          display: "flex",
          alignItems: "center",
          minHeight: "var(--space-24)",
          fontSize: "var(--font-size-11)",
          color: active ? "var(--color-accent)" : "var(--color-muted)",
          textAlign: column.num === true ? "right" : "left",
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {column.label}
        {active ? (dir === "asc" ? " ↑" : " ↓") : ""}
      </button>
      {/* The column rule is the resize handle: see ui/ResizeHandle. */}
      <ResizeHandle
        onPointerDown={onResizeStart}
        onDoubleClick={onResizeReset}
        title="Drag to resize this column · double-click to reset it"
      />
    </div>
  );
}

const Row = memo(function Row({
  book,
  columns,
  template,
  rowIndex,
  cursorCol,
  onTab,
  editingField,
  draft,
  onDraft,
  onStart,
  onCommit,
  onCancel,
  value,
  pendingFor,
  editable,
  selected,
  reviewing,
  onSelect,
  onMatch,
  focused,
}: {
  book: EditorBook;
  columns: Column[];
  template: string;
  rowIndex: number;
  cursorCol: number;
  /** Tab with the editor open: commit, then move where Tab would move. */
  onTab: (shift: boolean) => void;
  /** The field open for editing on this row, or undefined: a per-row value, so one row's edit does not re-render the rest. */
  editingField: EditorField | undefined;
  draft: string;
  onDraft: (value: string) => void;
  onStart: (book: EditorBook, field: EditorField) => void;
  onCommit: () => void;
  onCancel: () => void;
  value: (book: EditorBook, field: EditorField) => string;
  pendingFor: Partial<Record<EditorField, string>>;
  editable: readonly EditorField[];
  selected: boolean;
  /** The review panel is open: its diff is a snapshot, so the table must not accept another edit behind it. */
  reviewing: boolean;
  onSelect: (book: EditorBook, range: boolean) => void;
  /** Open the candidates behind this book's match. Undefined on a Calibre library, where none of it could be saved. */
  onMatch?: ((book: EditorBook) => void) | undefined;
  focused: boolean;
}) {
  return (
    <div
      data-row-index={rowIndex}
      role="row"
      // The header is row one in an ARIA grid, and the count is of every book, not of the rows drawn.
      aria-rowindex={rowIndex + 2}
      aria-selected={selected}
      style={{ display: "grid", gridTemplateColumns: template, gap: "var(--space-8)", height: ROW_HEIGHT, alignItems: "center", padding: "0 8px", borderBottom: "1px solid color-mix(in srgb, var(--color-line) 40%, transparent)" }}
    >
      <div role="gridcell" aria-colindex={1}>
        <input type="checkbox" aria-label={`Select ${book.title}`} checked={selected} onChange={(event) => onSelect(book, event.nativeEvent instanceof MouseEvent && event.nativeEvent.shiftKey)} />
      </div>
      {columns.map((column, columnIndex) => {
        const isCursor = cursorCol === columnIndex;
        const field = column.field;
        const ring = { boxShadow: isCursor ? "inset 0 0 0 1px var(--color-accent)" : "none" } as const;

        // Read-only columns carry no edit path: they are a value the server computed, not a field.
        if (field === undefined) {
          if (column.id === "status") {
            return (
              <div key={column.id} role="gridcell" aria-colindex={columnIndex + 2} id={`cell-${rowIndex}-${columnIndex}`} style={{ minWidth: 0, ...ring, display: "flex", gap: "var(--space-4)", overflow: "hidden", alignItems: "center", padding: "0 2px" }}>
                {/* One badge, then a count: a row with three issues used to push the last badge out of the cell, where
                    it was clipped mid-word and could not be read or hovered. */}
                {book.issues.slice(0, 1).map((issue) => (
                  <span
                    key={issue}
                    title={issue}
                    style={{ fontSize: "var(--font-size-10)", padding: "1px 6px", borderRadius: "var(--radius-pill)", background: toneWash(issueTone(issue)), color: toneInk(issueTone(issue)), whiteSpace: "nowrap" }}
                  >
                    {toneMark(issueTone(issue))} {issue}
                  </span>
                ))}
                {book.issues.length > 1 && (
                  <span
                    title={book.issues.slice(1).join(" · ")}
                    // The rest of the badges, coloured by the worst one behind the count: a cell reading "+3" is worth
                    // the same glance as the badge it hides.
                    style={{
                      fontSize: "var(--font-size-10)",
                      padding: "1px 6px",
                      borderRadius: "var(--radius-pill)",
                      background: toneWash(toneWorst(book.issues.slice(1).map(issueTone))),
                      color: toneInk(toneWorst(book.issues.slice(1).map(issueTone))),
                      whiteSpace: "nowrap",
                    }}
                  >
                    +{book.issues.length - 1}
                  </span>
                )}
                {book.edited && (
                  <span title="Mochil holds edits for this book" style={{ fontSize: "var(--font-size-10)", padding: "1px 6px", borderRadius: "var(--radius-pill)", background: "color-mix(in srgb, var(--color-text) 10%, transparent)", color: "var(--color-text)", whiteSpace: "nowrap" }}>
                    Edited
                  </span>
                )}
                {book.issues.length === 0 && !book.edited && (
                  <span style={{ fontSize: "var(--font-size-11)", opacity: 0.45 }}>ok</span>
                )}
              </div>
            );
          }
          const cell = readOnlyText(book, column.id);
          // Three columns are a judgement rather than a value, so they carry a colour: how sure the match is, whether
          // the file's ISBN agrees with the book, and what Hardcover already knows.
          const tone =
            column.id === "match"
              ? scoreTone(book.match?.score)
              : column.id === "isbn_found"
                ? isbnTone(isbnState(book))
                : column.id === "hardcover"
                  ? hardcoverTone(book.hardcover)
                  : "quiet";
          // The match score carries its judgement as a character as well as a colour, because e-ink maps all three tones
          // to one ink and because a fifth of readers cannot separate the hues.
          const text = column.id === "match" ? scoreLabel(book.match?.score) || cell.text : cell.text;
          const clickable = column.id === "match" && onMatch !== undefined;
          return (
            <div key={column.id} role="gridcell" aria-colindex={columnIndex + 2} id={`cell-${rowIndex}-${columnIndex}`} style={{ minWidth: 0, ...ring }}>
              <div
                onClick={clickable ? () => onMatch?.(book) : undefined}
                title={clickable ? `${cell.title}\n\nClick to see every candidate and search with better words.` : cell.title}
                style={{
                  padding: "3px 6px",
                  fontSize: "var(--font-size-12)",
                  textAlign: column.num === true ? "right" : "left",
                  opacity: tone === "quiet" ? 0.85 : 1,
                  color: tone === "quiet" ? undefined : toneColor(tone),
                  fontWeight: tone === "poor" ? 600 : undefined,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                  cursor: clickable ? "pointer" : undefined,
                }}
              >
                {text}
              </div>
            </div>
          );
        }

        const isEditing = editingField === field;
        const pending = pendingFor[field] !== undefined;
        const locked = reviewing || !editable.includes(field);
        const shown = value(book, field);
        return (
          <div key={column.id} role="gridcell" aria-colindex={columnIndex + 2} id={`cell-${rowIndex}-${columnIndex}`} style={{ minWidth: 0 }}>
            {isEditing ? (
              <input
                autoFocus
                aria-label={`${column.label} for ${book.title}`}
                value={draft}
                onChange={(event) => onDraft(event.target.value)}
                onBlur={onCommit}
                onKeyDown={(event) => {
                  if (event.key === "Enter") onCommit();
                  if (event.key === "Escape") onCancel();
                  if (event.key === "Tab") {
                    event.preventDefault();
                    onCommit();
                    onTab(event.shiftKey);
                  }
                }}
                style={{
                  width: "100%",
                  boxSizing: "border-box",
                  padding: "4px 6px",
                  border: "1px solid var(--color-accent)",
                  borderRadius: "var(--radius-8)",
                  background: "var(--color-bg)",
                  color: "var(--color-text)",
                  font: "inherit",
                  fontSize: "var(--font-size-12)",
                }}
              />
            ) : (
              <button
                type="button"
                // In the table, not in the browser's tab order: Tab means "next field" here, and five thousand cell
                // buttons in the tab order made it mean "walk through the library".
                tabIndex={-1}
                data-editor-cell=""
                disabled={locked}
                onClick={() => onStart(book, field)}
                style={{
                  all: "unset",
                  cursor: locked ? "default" : "text",
                  display: "block",
                  width: "100%",
                  padding: "4px 6px",
                  borderRadius: "var(--radius-8)",
                  fontSize: "var(--font-size-12)",
                  color: pending ? "var(--color-accent)" : "var(--color-text)",
                  background: pending
                    ? "color-mix(in srgb, var(--color-accent) 14%, transparent)"
                    : focused
                      ? "color-mix(in srgb, var(--color-text) 6%, transparent)"
                      : "transparent",
                  ...ring,
                  opacity: locked ? 0.65 : 1,
                  textAlign: column.num === true ? "right" : "left",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
                title={shown}
              >
                {shown === "" ? "—" : shown}
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
});
