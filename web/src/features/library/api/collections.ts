// features/library/api/collections.ts — user-made collections ("to read", "book club") and smart ones.
//
// The collections API existed with no rewrite UI at all. Two endpoints matter here:
//   GET /api/collections?library=<id>        → summaries: id, name, count, up to three cover URLs, optional rule
//   GET /api/collections/<id>/books          → the resolved books, with X-Total-Count for paging
// A "smart" collection carries a rule instead of members (it is a saved search) — the API resolves both the same
// way, so this client never has to know which kind it is looking at.
import { z } from "zod";
import { request, requestEnvelope } from "../../../lib/http";

const collectionSchema = z.looseObject({
  id: z.string(),
  name: z.string(),
  library_id: z.string().optional(),
  count: z.number(),
  covers: z.array(z.string()).nullable().optional(),
  rule: z.looseObject({}).nullable().optional(),
});

export interface CollectionSummary {
  id: string;
  name: string;
  count: number;
  coverUrl: string;
  smart: boolean;
}

export async function fetchCollections(libraryId: string): Promise<CollectionSummary[]> {
  const raw = await request<unknown>(`/api/collections?library=${encodeURIComponent(libraryId)}`);
  return z
    .array(collectionSchema)
    .parse(raw ?? [])
    .map((collection): CollectionSummary => ({
      id: collection.id,
      name: collection.name,
      count: collection.count,
      // The server hands back up to three cover paths; the card wants one, and the first is the natural pick.
      coverUrl: (collection.covers ?? [])[0] ?? "",
      smart: collection.rule !== undefined && collection.rule !== null,
    }));
}

const bookSchema = z.looseObject({
  id: z.string(),
  library_id: z.string(),
  library_kind: z.string().optional(),
  title: z.string(),
  authors: z.array(z.string()).default([]),
  series: z.string().optional(),
  series_index: z.number().optional(),
  cover_url: z.string().optional(),
  reading_progress: z
    .looseObject({ fraction: z.number().optional(), finished: z.boolean().optional() })
    .nullable()
    .optional(),
});

export type CollectionBook = z.infer<typeof bookSchema>;

export async function fetchCollectionBooks(collectionId: string, limit = 300): Promise<{ books: CollectionBook[]; total: number | null }> {
  const envelope = await requestEnvelope<unknown>(
    `/api/collections/${encodeURIComponent(collectionId)}/books?limit=${limit}&offset=0`,
  );
  return { books: z.array(bookSchema).parse(envelope.data ?? []), total: envelope.total };
}
