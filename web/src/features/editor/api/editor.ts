// features/editor/api/editor.ts — the library editor's metadata table.
//
// The server side already exists and is the same API the legacy editor used (internal/web/library_editor.go):
//
//   GET  /api/metadata/editor/books?library=&q=&filter=&sort=&dir=&offset=&limit=
//        → { items[], total, library_total, counts{}, collections[], columns[], hardcover_sync }
//   POST /api/metadata/editor/edit   { library, changes: {bookId: {field: value}}, preview: true|false }
//        preview → { changes: [{book, title, field, from, to}] }   (nothing written)
//        apply   → { changed, undo }                               (undo token)
//   POST /api/metadata/editor/undo   { library, undo }
//
// Searching, filtering and sorting are all done server-side, over the whole library rather than the page in front of
// you. That is the difference between an editor and a table, and it is why none of the three happens client-side here.
import { z } from "zod";
import { request } from "../../../lib/http";

/** A candidate from the metadata provider, as the server sends it (enrichment.Suggestion plus a score). */
export interface EditorMatch {
  title: string;
  authors: string[];
  year: number;
  overview: string;
  provider: string;
  providerId: string;
  score: number;
}

export interface EditorBook {
  id: string;
  title: string;
  authors: string[];
  series: string;
  seriesIndex: number | undefined;
  format: string;
  tags: string[];
  publisher: string;
  published: string;
  language: string;
  /** Already stars, 0–5: the server divides the stored 0–10 before sending. Never halve it again. */
  rating: number;
  description: string;
  identifiers: string;
  titleSort: string;
  authorSort: string;
  path: string;
  edited: boolean;
  issues: string[];
  custom: Record<string, string>;
  // Read-only columns, straight from the server's row (see references/api.md).
  collections: string[];
  /** 0–1. */
  progress: number;
  /** 0 means "not counted yet", not "zero words". */
  words: number;
  pages: number;
  isbnFound: string;
  isbnNote: string;
  match: EditorMatch | undefined;
  matchClear: boolean;
  matchNote: string;
  hardcover: string;
  hardcoverNote: string;
}

const bookSchema = z.looseObject({
  id: z.string(),
  title: z.string().optional(),
  authors: z.array(z.string()).nullable().optional(),
  series: z.string().optional(),
  series_index: z.number().nullable().optional(),
  format: z.string().optional(),
  tags: z.array(z.string()).nullable().optional(),
  publisher: z.string().optional(),
  published: z.string().optional(),
  language: z.string().optional(),
  rating: z.number().optional(),
  description: z.string().optional(),
  identifiers: z.string().optional(),
  title_sort: z.string().optional(),
  author_sort: z.string().optional(),
  path: z.string().optional(),
  edited: z.boolean().optional(),
  issues: z.array(z.string()).nullable().optional(),
  custom: z.record(z.string(), z.string()).nullable().optional(),
  collections: z.array(z.string()).nullable().optional(),
  progress: z.number().nullable().optional(),
  words: z.number().nullable().optional(),
  pages: z.number().nullable().optional(),
  isbn_found: z.string().nullable().optional(),
  isbn_note: z.string().nullable().optional(),
  match: z
    .looseObject({
      title: z.string().optional(),
      authors: z.array(z.string()).nullable().optional(),
      year: z.number().nullable().optional(),
      overview: z.string().nullable().optional(),
      provider: z.string().nullable().optional(),
      provider_id: z.string().nullable().optional(),
      score: z.number().nullable().optional(),
    })
    .nullable()
    .optional(),
  match_clear: z.boolean().nullable().optional(),
  match_note: z.string().nullable().optional(),
  hardcover: z.string().nullable().optional(),
  hardcover_note: z.string().nullable().optional(),
});

const columnSchema = z.looseObject({
  name: z.string().optional(),
  label: z.string().optional(),
});

const transferTargetSchema = z.looseObject({ id: z.string().optional(), name: z.string().optional() });

const pageSchema = z.looseObject({
  items: z.array(bookSchema).nullable().optional(),
  total: z.number().optional(),
  library_total: z.number().optional(),
  counts: z.record(z.string(), z.number()).nullable().optional(),
  columns: z.array(columnSchema).nullable().optional(),
  transfer_to: z.array(transferTargetSchema).nullable().optional(),
});

export interface EditorPage {
  items: EditorBook[];
  total: number;
  libraryTotal: number;
  counts: Record<string, number>;
  columns: { name: string; label: string }[];
  transferTo: TransferTarget[];
}

/** Another library books can be moved or copied into. */
export interface TransferTarget {
  id: string;
  name: string;
}

/** The fields the table lets you write. The same names the server validates; anything else is rejected. */
export const EDITOR_FIELDS = [
  "title",
  "authors",
  "series",
  "series_index",
  "publisher",
  "published",
  "language",
  "tags",
  "identifiers",
  "description",
  "title_sort",
  "author_sort",
] as const;

export type EditorField = (typeof EDITOR_FIELDS)[number];

/** Human labels, used by the bulk-edit picker and the details pane, so neither invents its own wording. */
export const FIELD_LABELS: Record<EditorField, string> = {
  title: "Title",
  authors: "Authors",
  series: "Series",
  series_index: "Series position",
  publisher: "Publisher",
  published: "Year",
  language: "Language",
  tags: "Tags",
  identifiers: "Identifiers",
  description: "Description",
  title_sort: "Title sort",
  author_sort: "Author sort",
};

/** Fields Calibre owns. Writing them from Mochil is refused with a 409 (internal/web/library_editor.go's
 *  `calibreOwnedFields`) because Calibre would overwrite them on its next save — so the table must not offer them. */
export const CALIBRE_OWNED_FIELDS: readonly EditorField[] = ["title", "authors", "series", "series_index", "identifiers"];

/** Which of a library's fields this editor may write. A Calibre library keeps its own metadata. */
export function editableFieldsFor(sourceType: string): readonly EditorField[] {
  return sourceType === "calibre" ? EDITOR_FIELDS.filter((field) => !CALIBRE_OWNED_FIELDS.includes(field)) : EDITOR_FIELDS;
}

/** "1996-12-15T18:29:17+00:00" → "1996". The stored value is a full timestamp; the column reads as a year, like
 *  Calibre's. Anything that does not look like a date is passed through, because some books carry "2015" or "" there. */
export function yearOf(published: string): string {
  const match = /^(\d{4})-\d{2}-\d{2}/.exec(published);
  if (match === null) return published;
  const year = match[1] ?? "";
  // The server writes Go's zero date for a book with no publication date; "0101" is not a year anyone wants to see.
  return Number(year) < 1000 ? "" : year;
}

/** The server's sort key for a column — not always the field name ("author", "number"). See library_editor_sort.go. */
export function sortKeyFor(column: string): string {
  if (column === "authors") return "author";
  if (column === "series_index") return "number";
  return column;
}

/** A book's editable value for one field, in the form the field displays and round-trips (authors with " & "). */
export function valueOf(book: EditorBook, field: EditorField): string {
  switch (field) {
    case "title":
      return book.title;
    case "authors":
      return book.authors.join(" & ");
    case "series":
      return book.series;
    case "series_index":
      return book.seriesIndex === undefined ? "" : String(book.seriesIndex);
    case "publisher":
      return book.publisher;
    case "published":
      return yearOf(book.published);
    case "language":
      return book.language;
    case "tags":
      return book.tags.join(", ");
    case "identifiers":
      return book.identifiers;
    case "description":
      return book.description;
    case "title_sort":
      return book.titleSort;
    case "author_sort":
      return book.authorSort;
  }
}

export interface EditorQuery {
  library: string;
  q: string;
  filter: string;
  /** A collection id, or "" for the whole library. */
  collection: string;
  sort: string;
  dir: "asc" | "desc";
  offset: number;
  limit: number;
}

export async function fetchEditorBooks(query: EditorQuery): Promise<EditorPage> {
  const params = new URLSearchParams({
    library: query.library,
    offset: String(query.offset),
    limit: String(query.limit),
    sort: query.sort,
    dir: query.dir,
  });
  if (query.q !== "") params.set("q", query.q);
  if (query.filter !== "") params.set("filter", query.filter);
  if (query.collection !== "") params.set("collection", query.collection);
  const raw = await request<unknown>(`/api/metadata/editor/books?${params}`);
  const parsed = pageSchema.parse(raw);
  return {
    total: parsed.total ?? 0,
    libraryTotal: parsed.library_total ?? parsed.total ?? 0,
    counts: parsed.counts ?? {},
    columns: (parsed.columns ?? []).map((column) => ({
      name: column.name ?? "",
      label: column.label ?? column.name ?? "",
    })),
    transferTo: (parsed.transfer_to ?? []).map((target) => ({ id: target.id ?? "", name: target.name ?? "" })),
    items: (parsed.items ?? []).map((book): EditorBook => ({
      id: book.id,
      title: book.title ?? "",
      authors: book.authors ?? [],
      series: book.series ?? "",
      seriesIndex: book.series_index ?? undefined,
      format: book.format ?? "",
      tags: book.tags ?? [],
      publisher: book.publisher ?? "",
      published: book.published ?? "",
      language: book.language ?? "",
      rating: book.rating ?? 0,
      description: book.description ?? "",
      identifiers: book.identifiers ?? "",
      titleSort: book.title_sort ?? "",
      authorSort: book.author_sort ?? "",
      path: book.path ?? "",
      edited: book.edited === true,
      issues: book.issues ?? [],
      custom: book.custom ?? {},
      collections: book.collections ?? [],
      progress: book.progress ?? 0,
      words: book.words ?? 0,
      pages: book.pages ?? 0,
      isbnFound: book.isbn_found ?? "",
      isbnNote: book.isbn_note ?? "",
      match:
        book.match === null || book.match === undefined
          ? undefined
          : {
              title: book.match.title ?? "",
              authors: book.match.authors ?? [],
              year: book.match.year ?? 0,
              overview: book.match.overview ?? "",
              provider: book.match.provider ?? "",
              providerId: book.match.provider_id ?? "",
              score: book.match.score ?? 0,
            },
      matchClear: book.match_clear === true,
      matchNote: book.match_note ?? "",
      hardcover: book.hardcover ?? "",
      hardcoverNote: book.hardcover_note ?? "",
    })),
  };
}

/** `changes` is keyed by book id, then field name — exactly what the server expects. */
export type EditorChanges = Record<string, Partial<Record<EditorField, string>>>;

export interface EditorChange {
  book: string;
  title: string;
  field: string;
  from: string;
  to: string;
}

const changeSchema = z.looseObject({
  book: z.string().optional(),
  title: z.string().optional(),
  field: z.string().optional(),
  from: z.string().optional(),
  to: z.string().optional(),
});

/** What applying the pending changes would do, without doing it. */
export async function previewChanges(libraryId: string, changes: EditorChanges, reset: string[] = []): Promise<EditorChange[]> {
  const raw = await request<{ changes?: unknown }>("/api/metadata/editor/edit", {
    method: "POST",
    body: { library: libraryId, changes, reset, preview: true },
  });
  return z
    .array(changeSchema)
    .parse(raw.changes ?? [])
    .map((change) => ({
      book: change.book ?? "",
      title: change.title ?? "",
      field: change.field ?? "",
      from: change.from ?? "",
      to: change.to ?? "",
    }));
}

export interface ApplyResult {
  changed: number;
  undo: string;
}

export async function applyChanges(libraryId: string, changes: EditorChanges, reset: string[] = []): Promise<ApplyResult> {
  const raw = await request<{ changed?: number; undo?: string }>("/api/metadata/editor/edit", {
    method: "POST",
    body: { library: libraryId, changes, reset, preview: false },
  });
  return { changed: raw.changed ?? 0, undo: raw.undo ?? "" };
}

export async function undoChanges(libraryId: string, token: string): Promise<void> {
  await request<unknown>("/api/metadata/editor/undo", { method: "POST", body: { library: libraryId, undo: token } });
}

/** Count of pending field edits, for the save bar. */
export function changeCount(changes: EditorChanges): number {
  return Object.values(changes).reduce((total, fields) => total + Object.keys(fields).length, 0);
}

/** Set one field on many books at once — Calibre's bulk edit. Returns a new map; the caller keeps it in state. */
export function setOnBooks(changes: EditorChanges, books: string[], field: EditorField, value: string): EditorChanges {
  const next: EditorChanges = { ...changes };
  for (const book of books) next[book] = { ...(next[book] ?? {}), [field]: value };
  return next;
}

export interface EditorAll {
  items: EditorBook[];
  total: number;
  libraryTotal: number;
  counts: Record<string, number>;
  columns: { name: string; label: string }[];
  transferTo: TransferTarget[];
}

/** Loads every matching row in 1 000-row chunks (the server's maximum page) so the whole library is one scrollable
 *  spreadsheet instead of pages. A few requests for a few thousand books; the legacy editor did the same. */
export async function fetchAllEditorBooks(query: Omit<EditorQuery, "offset" | "limit">): Promise<EditorAll> {
  const CHUNK = 1000;
  const items: EditorBook[] = [];
  let last: Awaited<ReturnType<typeof fetchEditorBooks>> | undefined;
  for (let offset = 0; ; offset += CHUNK) {
    const page = await fetchEditorBooks({ ...query, offset, limit: CHUNK });
    items.push(...page.items);
    last = page;
    if (items.length >= page.total) break;
  }
  // Every page carries the same totals, counts and column definitions; read them from the final one.
  const summary = last as NonNullable<typeof last>;
  return {
    items,
    total: summary.total,
    libraryTotal: summary.libraryTotal,
    counts: summary.counts,
    columns: summary.columns,
    transferTo: summary.transferTo,
  };
}

/** Per-field validation, the same rules the server enforces (ported from the legacy editor-model.js). */
export function validateField(field: EditorField, value: string): string | undefined {
  const text = value.trim();
  if (field === "title" && text === "") return "A title cannot be empty.";
  if (field === "series_index" && text !== "") {
    const number = Number(text);
    if (!Number.isFinite(number) || number <= 0) return "A volume number must be a positive number.";
  }
  return undefined;
}

/** Set one field on many books. Values are trimmed, the way the legacy editor did it. */
export function setAllOn(books: EditorBook[], field: EditorField, value: string): EditorChanges {
  const next: EditorChanges = {};
  for (const book of books) next[book.id] = { [field]: value.trim() };
  return next;
}

/** Give the books, in the order given, the numbers start, start+step, … */
export function renumberBooks(books: EditorBook[], start = 1, step = 1): EditorChanges {
  const clean = (number: number) => String(Math.round(number * 1000) / 1000);
  const next: EditorChanges = {};
  books.forEach((book, index) => {
    next[book.id] = { series_index: clean(start + index * step) };
  });
  return next;
}

/** Swap text inside a field. With `prefixOnly` only text at the start is touched — what stripping "Series - " needs. */
export function replaceInBooks(
  books: EditorBook[],
  field: EditorField,
  find: string,
  replacement: string,
  { prefixOnly = false, matchCase = false }: { prefixOnly?: boolean; matchCase?: boolean } = {},
): EditorChanges {
  if (find === "") return {};
  const escaped = find.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const pattern = new RegExp(prefixOnly ? `^${escaped}` : escaped, prefixOnly ? (matchCase ? "" : "i") : matchCase ? "g" : "gi");
  const next: EditorChanges = {};
  for (const book of books) {
    const before = valueOf(book, field);
    const after = before.replace(pattern, () => replacement).replace(/\s{2,}/g, " ").trim();
    if (after !== before) next[book.id] = { [field]: after };
  }
  return next;
}

/** Merge freshly computed changes over the pending ones, per book and per field. */
export function mergeChanges(base: EditorChanges, incoming: EditorChanges): EditorChanges {
  const next: EditorChanges = { ...base };
  for (const [book, fields] of Object.entries(incoming)) {
    next[book] = { ...(next[book] ?? {}), ...fields };
  }
  return next;
}

/** "isbn:123, hardcover:abc" with one scheme replaced or added; the other identifiers survive. */
export function identifierWith(existing: string, scheme: string, value: string): string {
  const parts = existing
    .split(",")
    .map((part) => part.trim())
    .filter((part) => part !== "")
    .filter((part) => !part.toLowerCase().startsWith(`${scheme.toLowerCase()}:`));
  if (value !== "") parts.push(`${scheme}:${value}`);
  return parts.join(", ");
}

/** Does the book already carry an ISBN in its identifiers? */
export function isbnOf(book: EditorBook): string {
  for (const part of book.identifiers.split(",")) {
    const [scheme, value] = part.split(":");
    if (scheme !== undefined && scheme.trim().toLowerCase() === "isbn" && (value ?? "").trim() !== "") return (value ?? "").trim();
  }
  return "";
}

/** What taking a certain match does to one book: fill the fields it has nothing in, and add the link. Never replaces a
 *  value the book already says — that is the difference between "Use matches" and rewriting metadata. */
export function matchFill(book: EditorBook, editable: readonly EditorField[]): Partial<Record<EditorField, string>> {
  const match = book.match;
  if (match === undefined || !book.matchClear) return {};
  const fields: Partial<Record<EditorField, string>> = {};
  const take = (field: EditorField, from: string, to: string) => {
    if (!editable.includes(field)) return;
    if (to !== "" && to !== from && from === "") fields[field] = to;
  };
  take("title", book.title, match.title);
  take("authors", book.authors.join(" & "), match.authors.join(" & "));
  take("published", yearOf(book.published), match.year > 0 ? String(match.year) : "");
  take("description", book.description, match.overview);
  if (editable.includes("identifiers") && match.providerId !== "" && /hardcover/i.test(match.provider)) {
    const identifiers = identifierWith(book.identifiers, "hardcover", match.providerId);
    if (identifiers !== book.identifiers) fields.identifiers = identifiers;
  }
  return fields;
}

/** Bulk "Use clear matches": every book with a certain match, as pending edits. */
export function matchFills(books: EditorBook[], editable: readonly EditorField[]): EditorChanges {
  const next: EditorChanges = {};
  for (const book of books) {
    const fields = matchFill(book, editable);
    if (Object.keys(fields).length > 0) next[book.id] = fields;
  }
  return next;
}

/** Bulk "Use found ISBNs": only books that have none and were given one by the Find ISBNs job. */
export function isbnFills(books: EditorBook[], editable: readonly EditorField[]): EditorChanges {
  if (!editable.includes("identifiers")) return {};
  const next: EditorChanges = {};
  for (const book of books) {
    if (book.isbnFound === "" || isbnOf(book) !== "") continue;
    next[book.id] = { identifiers: identifierWith(book.identifiers, "isbn", book.isbnFound) };
  }
  return next;
}

// ---- collections ----
export interface EditorCollection {
  id: string;
  name: string;
  /** A smart collection is a saved rule; its members are worked out, not picked. */
  smart: boolean;
  count: number;
}

const collectionSchema = z.looseObject({
  id: z.string().optional(),
  name: z.string().optional(),
  smart: z.boolean().nullable().optional(),
  count: z.number().nullable().optional(),
});

export async function fetchCollections(libraryId: string): Promise<EditorCollection[]> {
  const raw = await request<{ collections?: unknown }>(
    `/api/metadata/editor/collections?library=${encodeURIComponent(libraryId)}`,
  );
  return z
    .array(collectionSchema)
    .parse(raw.collections ?? [])
    .map((entry) => ({ id: entry.id ?? "", name: entry.name ?? "", smart: entry.smart === true, count: entry.count ?? 0 }))
    .filter((entry) => entry.id !== "");
}

/** Put books in a collection, or take them out. Passing a name with no id creates the collection. */
export async function setCollectionMembers(
  libraryId: string,
  target: { id?: string; name?: string },
  ids: string[],
  remove: boolean,
): Promise<void> {
  await request<unknown>("/api/metadata/editor/collections", {
    method: "POST",
    body: { library: libraryId, collection: target.id ?? "", name: target.name ?? "", ids, remove },
  });
}

// ---- move or copy to another library ----
export interface TransferPreview {
  ready: number;
  blocked: number;
  to: string;
  books: { id: string; title: string; reason: string }[];
}

export async function previewTransfer(
  libraryId: string,
  to: string,
  ids: string[],
  mode: "move" | "copy",
): Promise<TransferPreview> {
  const raw = await request<unknown>("/api/metadata/editor/transfer", {
    method: "POST",
    body: { library: libraryId, to, ids, mode, preview: true },
  });
  const parsed = z
    .looseObject({
      ready: z.number().optional(),
      blocked: z.number().optional(),
      to: z.string().optional(),
      books: z
        .array(z.looseObject({ id: z.string().optional(), title: z.string().optional(), reason: z.string().optional() }))
        .nullable()
        .optional(),
    })
    .parse(raw);
  return {
    ready: parsed.ready ?? 0,
    blocked: parsed.blocked ?? 0,
    to: parsed.to ?? "",
    books: (parsed.books ?? []).map((book) => ({ id: book.id ?? "", title: book.title ?? "", reason: book.reason ?? "" })),
  };
}

export async function applyTransfer(
  libraryId: string,
  to: string,
  ids: string[],
  mode: "move" | "copy",
): Promise<{ token: string; rescan: string[] }> {
  const raw = await request<{ token?: string; rescan?: string[] }>("/api/metadata/editor/transfer", {
    method: "POST",
    body: { library: libraryId, to, ids, mode, preview: false },
  });
  return { token: raw.token ?? "", rescan: raw.rescan ?? [] };
}

// ---- background jobs (Find ISBNs, Find matches, the AI asks) ----
export type EditorTool = "isbn" | "isbn-ask" | "matches" | "match-ask" | "check" | "count";

export interface JobState {
  status: "idle" | "queued" | "running" | "ready" | "failed" | "canceled";
  current: number;
  total: number;
  message: string;
  error: string;
}

const jobSchema = z.looseObject({
  status: z.string().optional(),
  current: z.number().nullable().optional(),
  total: z.number().nullable().optional(),
  message: z.string().nullable().optional(),
  error: z.string().nullable().optional(),
});

function jobPath(tool: EditorTool): string {
  return `/api/metadata/editor/${tool}`;
}

/** Start a job on the selection (ids) or the whole library (no ids). It keeps running if this page closes. */
export async function startJob(libraryId: string, tool: EditorTool, ids: string[]): Promise<void> {
  await request<unknown>(jobPath(tool), { method: "POST", body: { library: libraryId, ids } });
}

export async function jobState(libraryId: string, tool: EditorTool): Promise<JobState> {
  const raw = jobSchema.parse(await request<unknown>(`${jobPath(tool)}?library=${encodeURIComponent(libraryId)}`));
  const status = raw.status ?? "idle";
  return {
    status: (["idle", "queued", "running", "ready", "failed", "canceled"].includes(status) ? status : "idle") as JobState["status"],
    current: raw.current ?? 0,
    total: raw.total ?? 0,
    message: raw.message ?? "",
    error: raw.error ?? "",
  };
}

export async function stopJob(libraryId: string, tool: EditorTool): Promise<void> {
  await request<unknown>(`${jobPath(tool)}?library=${encodeURIComponent(libraryId)}`, { method: "DELETE" });
}

/** Which rows to draw for a box scrolled `scrollTop` pixels down and `viewport` pixels tall, plus blank space above and
 *  below them. Ported from the legacy editor's virtual-rows.js so a 20 000-row library scrolls like a desktop app. */
export function windowFor({
  scrollTop,
  viewport,
  rowHeight,
  count,
  overscan = 12,
}: {
  scrollTop: number;
  viewport: number;
  rowHeight: number;
  count: number;
  overscan?: number;
}): { start: number; end: number; before: number; after: number } {
  const height = Math.max(1, rowHeight);
  const start = Math.max(0, Math.floor(Math.max(0, scrollTop) / height) - overscan);
  const end = Math.min(count, Math.ceil((Math.max(0, scrollTop) + viewport) / height) + overscan);
  return { start, end: Math.max(start, end), before: start * height, after: Math.max(0, (count - Math.max(start, end)) * height) };
}
