// features/import/api/import.ts — what the drop folder holds, as the server sees it.
//
// The server reads the drop folder and proposes a book for each file: a status (new, new volume, possible duplicate,
// already in library), the title/author/series it guessed, and whatever conflicts it noticed. This module only describes
// that reply — nothing here writes anything. Copying is a separate, deliberate step (`managed-ingest`), and the reason
// Import exists as a screen at all is that nothing moves until a person has read the list.
import { z } from "zod";
import { request } from "../../../lib/http";

const proposal = z.object({
  title: z.string().optional().default(""),
  authors: z.array(z.string()).optional().default([]),
  series: z.string().optional().default(""),
  series_index: z.number().nullable().optional(),
  conflicts: z.array(z.string()).optional().default([]),
  evidence: z
    .array(z.object({ field: z.string(), confidence: z.string().optional().default("") }))
    .optional()
    .default([]),
});

export const importRow = z.object({
  source_path: z.string(),
  status: z.string(),
  proposal: proposal.optional(),
  existing: z.object({ title: z.string().optional().default("") }).nullable().optional(),
  copied: z.boolean().optional().default(false),
  /** The name the discovery pass itself worked out; the screen shows it before anyone edits, so the naming POST is only
   *  needed for rows a person changed. */
  suggested_target: z.string().optional().default(""),
  destination: z.string().optional().default(""),
});

export type ImportRow = z.infer<typeof importRow>;

const destination = z.object({ id: z.string(), name: z.string() });

const page = z.object({
  enabled: z.boolean().optional().default(true),
  incoming_directory: z.string().optional().default(""),
  managed_directory: z.string().optional().default(""),
  managed_library: z.object({ id: z.string(), name: z.string() }).nullable().optional(),
  destinations: z.array(destination).optional().default([]),
  items: z.array(importRow),
  next_cursor: z.string().optional().default(""),
  scanned: z.number().optional().default(0),
  /** The discovery walk hit its visit cap: the list is a partial answer, not the folder. */
  truncated: z.boolean().optional().default(false),
});

export type ImportPage = z.infer<typeof page>;

/**
 * The filters the server understands, in the order the screen shows them. `statuses` is the wire value of `?status=`;
 * "Everything" sends an empty one. Kept here rather than in the screen because it is part of the API's contract.
 */
export const IMPORT_FILTERS: { id: string; label: string; statuses: string }[] = [
  { id: "fresh", label: "New", statuses: "new,new_volume" },
  { id: "new", label: "New series", statuses: "new" },
  { id: "new_volume", label: "New volumes", statuses: "new_volume" },
  { id: "possible", label: "Possible duplicates", statuses: "possible" },
  { id: "in_library", label: "Already in library", statuses: "in_library" },
  { id: "all", label: "Everything", statuses: "" },
];

/** How a row's status reads to a person. Unknown values fall back to the wire value rather than disappearing. */
export const STATUS_TEXT: Record<string, string> = {
  new: "New",
  new_volume: "New volume",
  possible: "Possible duplicate",
  in_library: "Already in library",
};

/**
 * The naming presets the server accepts (internal/ingest/naming.go). Kept here rather than in the screen so a test can hold
 * them against the wire values: the screen once offered "folder", which the server rejects with 400 and which blanked the
 * whole list.
 */
export const NAMING_PRESETS: { id: string; label: string }[] = [
  { id: "default", label: "Author / Series / Title" },
  { id: "calibre", label: "Author / Title (Calibre)" },
  { id: "kavita", label: "Series / Volume (Kavita)" },
];

export interface ImportQuery {
  library: string;
  naming: string;
  filter: string;
  cursor?: string;
}

export async function fetchImportPage({ library, naming, filter, cursor }: ImportQuery): Promise<ImportPage> {
  const statuses = IMPORT_FILTERS.find((entry) => entry.id === filter)?.statuses ?? "";
  const params = new URLSearchParams({ library, naming, status: statuses, limit: "100" });
  if (cursor) params.set("cursor", cursor);
  return page.parse(await request<unknown>(`/api/metadata/editor/import?${params.toString()}`));
}

const library = z.object({ id: z.string(), name: z.string(), kind: z.string() });

/**
 * Only the kinds the server itself treats as publications (internal/web's publicationKind). An audiobooks or calibre
 * entry here would be offered by the screen and then refused with a 404 by the import endpoint, with no way back.
 */
export const PUBLICATION_KINDS = new Set(["books", "comics", "manga", "mixed"]);

export async function fetchImportLibraries() {
  const rows = z.array(library).parse(await request<unknown>("/api/libraries"));
  return rows.filter((entry) => PUBLICATION_KINDS.has(entry.kind));
}

export interface ImportEdit {
  title: string;
  authors: string[];
  series: string;
  seriesIndex: number | null;
}

const named = z.object({ source_path: z.string(), target: z.string() });

/**
 * What each book will be called inside the managed folder, from the fields as they stand — the same layouts discovery
 * uses. The server names at most 100 books per call, which is why the screen sends the page and not the whole folder.
 */
export async function fetchImportNames(naming: string, items: { source: string; edit: ImportEdit }[]): Promise<Map<string, string>> {
  if (items.length === 0) return new Map();
  const reply = z
    .object({ items: z.array(named) })
    .parse(
      await request<unknown>("/api/metadata/editor/import-names", {
        method: "POST",
        // `request` serialises the body itself; handing it a string would send a JSON-encoded string and the server
        // would answer 400 — which the screen would silently render as "needs a title".
        body: {
          naming,
          items: items.map(({ source, edit }) => ({
            source_path: source,
            title: edit.title,
            authors: edit.authors,
            series: edit.series,
            series_index: edit.seriesIndex,
          })),
        },
      }),
    );
  return new Map(reply.items.map((entry) => [entry.source_path, entry.target]));
}

/** A row's fields as the list offers them, before anyone edits. */
export function editFromRow(row: ImportRow): ImportEdit {
  return {
    title: row.proposal?.title ?? "",
    authors: row.proposal?.authors ?? [],
    series: row.proposal?.series ?? "",
    seriesIndex: row.proposal?.series_index ?? null,
  };
}

const previewEntry = z.object({
  source_path: z.string(),
  target_path: z.string().optional().default(""),
  ready: z.boolean().optional().default(false),
  issues: z
    .array(z.object({ code: z.string().optional().default(""), message: z.string().optional().default("") }))
    .optional()
    .default([]),
});

export const batchPreview = z.object({
  entries: z.array(previewEntry).optional().default([]),
  ready: z.boolean().optional().default(false),
  conflicts: z.number().optional().default(0),
});
export type BatchPreview = z.infer<typeof batchPreview>;

export const transaction = z.object({
  id: z.string(),
  state: z.string(),
  entries: z.array(z.object({ source_path: z.string(), target_path: z.string() })).optional().default([]),
  error: z.string().optional().default(""),
});
export type IngestTransaction = z.infer<typeof transaction>;

export interface CopyEntry {
  source: string;
  target: string;
}

/**
 * What copying would do, before it does it. The server checks every entry — source present and inside the drop folder, target
 * free, sizes sane — and answers per entry, which is the only honest way to offer a "copy" button over a hundred files.
 */
export async function previewCopy(entries: CopyEntry[]): Promise<BatchPreview> {
  return batchPreview.parse(
    await request<unknown>("/api/metadata/managed-ingest/preview", {
      method: "POST",
      body: { entries: entries.map((entry) => ({ source_path: entry.source, target_path: entry.target })) },
    }),
  );
}

/** Queues the copy. It is a transaction: it can be polled, and a failure leaves a record rather than half-copied shelves. */
export async function planCopy(entries: CopyEntry[]): Promise<IngestTransaction> {
  return transaction.parse(
    await request<unknown>("/api/metadata/managed-ingest/plan", {
      method: "POST",
      body: { entries: entries.map((entry) => ({ source_path: entry.source, target_path: entry.target })) },
    }),
  );
}

/** Runs a planned transaction. The server answers as soon as the work is queued; progress is the transaction's state. */
export async function executeCopy(id: string): Promise<IngestTransaction> {
  return transaction.parse(
    await request<unknown>(`/api/metadata/managed-ingest/${encodeURIComponent(id)}/execute`, {
      method: "POST",
      body: { confirmation: id },
    }),
  );
}

export async function fetchTransaction(id: string): Promise<IngestTransaction> {
  return transaction.parse(await request<unknown>(`/api/metadata/managed-ingest/${encodeURIComponent(id)}`));
}

/** The metadata a sidecar or an embed carries. The import screen knows these four fields and nothing else, so it sends
 *  exactly those: a made-up tag list or publisher would be written into the book's own metadata. */
export interface CopyMetadata {
  title: string;
  authors: string[];
  series: string;
  seriesIndex: number | null;
}

const writeResult = z.object({
  target_path: z.string(),
  sidecar: z.string().optional().default(""),
  status: z.string().optional().default(""),
  message: z.string().optional().default(""),
});
const writeResults = z.object({ results: z.array(writeResult).optional().default([]) });
export type CopyWriteResult = z.infer<typeof writeResult>;

function metadataBody(entries: { target: string; metadata: CopyMetadata }[]) {
  return {
    entries: entries.map((entry) => ({
      target_path: entry.target,
      metadata: {
        title: entry.metadata.title,
        authors: entry.metadata.authors,
        series: entry.metadata.series,
        series_index: entry.metadata.seriesIndex,
      },
    })),
  };
}

/** Writes a `metadata.opf` beside each copied book, which is what Calibre and KOReader read. */
export async function writeSidecars(entries: { target: string; metadata: CopyMetadata }[]): Promise<CopyWriteResult[]> {
  const reply = writeResults.parse(
    await request<unknown>("/api/metadata/managed-ingest/sidecars", { method: "POST", body: metadataBody(entries) }),
  );
  return reply.results;
}

/** Rewrites the EPUB's own metadata in place. Slower and more invasive than a sidecar, which is why it is a separate choice. */
export async function embedMetadata(entries: { target: string; metadata: CopyMetadata }[]): Promise<CopyWriteResult[]> {
  const reply = writeResults.parse(
    await request<unknown>("/api/metadata/managed-ingest/embed", { method: "POST", body: metadataBody(entries) }),
  );
  return reply.results;
}
