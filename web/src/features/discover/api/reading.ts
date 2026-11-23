// features/discover/api/reading.ts — /api/reading, one of the 8 contract-fixtured endpoints: the signed-in
// account's in-progress books, each carrying the same book fields a card needs plus the fraction read.
import { z } from "zod";
import { request } from "../../../lib/http";

const readingItemSchema = z.object({
  id: z.string(),
  title: z.string(),
  authors: z.array(z.string()).default([]),
  series: z.string().optional(),
  series_index: z.number().optional(),
  library_id: z.string(),
  library_kind: z.string(),
  cover_url: z.string(),
  fraction: z.number(),
  updated_at: z.number(),
});

export type ReadingItem = z.infer<typeof readingItemSchema>;

export async function fetchContinueReading(limit = 20): Promise<ReadingItem[]> {
  const params = new URLSearchParams({ limit: String(limit) });
  const raw = await request<unknown>(`/api/reading?${params}`);
  return z.array(readingItemSchema).parse(raw);
}
