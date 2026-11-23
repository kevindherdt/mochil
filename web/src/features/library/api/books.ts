// features/library/api/books.ts — typed endpoint + zod schema for /api/books, one of the 8 contract-fixtured
// endpoints. Only the fields the cards actually need are declared — the real response carries more (path,
// size, modified, etc.) that these screens have no use for yet.
import { z } from "zod";
import { request, requestEnvelope } from "../../../lib/http";

const readingProgressSchema = z.object({
  fraction: z.number(),
  updated_at: z.number(),
  finished: z.boolean().optional(),
  status: z.string().optional(),
});

const bookSchema = z.object({
  id: z.string(),
  title: z.string(),
  authors: z.array(z.string()),
  series: z.string().optional(),
  series_index: z.number().optional(),
  library_id: z.string(),
  library_kind: z.string(),
  /** Container format ("epub", "cbz", "flac", "mkv"). Music cards print it on the artwork, so it has to be real. */
  format: z.string().optional(),
  cover_url: z.string(),
  reading_progress: readingProgressSchema.optional(),
});

export type BookInfo = z.infer<typeof bookSchema>;

/** The detail screen's fields on top of the card's — `/api/books/{library}/{id}` (`findBook` needs both: a book
    id alone isn't unique across libraries), just more of the same response. `description` is server-sourced HTML
    (calibre/metadata descriptions commonly are), so the screen must treat it as markup, not plain text. */
const bookDetailSchema = bookSchema.extend({
  description: z.string().optional(),
  publisher: z.string().optional(),
  published: z.string().optional(),
  read_url: z.string(),
  size: z.number().optional(),
});

export type BookDetail = z.infer<typeof bookDetailSchema>;

export async function fetchBookDetail(libraryId: string, id: string): Promise<BookDetail> {
  const raw = await request<unknown>(`/api/books/${encodeURIComponent(libraryId)}/${encodeURIComponent(id)}`);
  return bookDetailSchema.parse(raw);
}

export interface BookFilter {
  /** Only this series (drill-down from the series group list) — the server already supports `?series=`. */
  series?: string;
  /** Only this author (drill-down from the author group list) — `?author=`. */
  author?: string;
  /** Free-text search across title/author — the server's own `?q=` matching, not a client-side filter. */
  q?: string;
  /** Reading status, answered by the server from the account's own positions (see internal/web/reading_status_filter.go). */
  status?: BookStatus;
}

export type BookStatus = "in_progress" | "unread" | "read";

/** Newest first across every library — Discover's "Recently added" rail. Same endpoint and schema as the shelf,
    with no `library`, which the server reads as "all libraries". */
export async function fetchRecentBooks(limit = 20): Promise<BookInfo[]> {
  const params = new URLSearchParams({ sort: "recent", limit: String(limit) });
  const raw = await request<unknown>(`/api/books?${params}`);
  return z.array(bookSchema).parse(raw);
}

export interface BookPage {
  items: BookInfo[];
  /** The server's cursor for the next window of this exact view, or null at the end of the shelf. */
  nextCursor: string | null;
  /** Size of the whole filtered shelf, from X-Total-Count. */
  total: number | null;
}

/** One window of a shelf, plus the paging cursor. The cursor is the server's own token — it encodes the view it
    was minted for, so a cursor from a different filter is rejected by the server rather than silently skipping
    books, which is why the caller must not cache one across filters. */
export async function fetchBooksPage(libraryId: string, filter: BookFilter = {}, cursor?: string, limit = 120): Promise<BookPage> {
  const params = new URLSearchParams({ library: libraryId, limit: String(limit) });
  if (filter.series !== undefined) params.set("series", filter.series);
  if (filter.author !== undefined) params.set("author", filter.author);
  if (filter.q !== undefined && filter.q !== "") params.set("q", filter.q);
  if (filter.status !== undefined) params.set("status", filter.status);
  if (cursor !== undefined) params.set("cursor", cursor);
  const envelope = await requestEnvelope<unknown>(`/api/books?${params}`);
  return { items: z.array(bookSchema).parse(envelope.data), nextCursor: envelope.nextCursor, total: envelope.total };
}

/** First window only, for callers that don't page (Discover's rails). */
export async function fetchBooks(libraryId: string, filter: BookFilter = {}, limit = 120): Promise<BookInfo[]> {
  const params = new URLSearchParams({ library: libraryId, limit: String(limit) });
  if (filter.series !== undefined) params.set("series", filter.series);
  if (filter.author !== undefined) params.set("author", filter.author);
  if (filter.q !== undefined && filter.q !== "") params.set("q", filter.q);
  if (filter.status !== undefined) params.set("status", filter.status);
  const raw = await request<unknown>(`/api/books?${params}`);
  return z.array(bookSchema).parse(raw);
}
