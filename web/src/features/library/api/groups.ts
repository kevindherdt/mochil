// features/library/api/groups.ts — typed endpoints for /api/series and /api/authors (the grouped browse lists).
// Shapes verified against the LIVE dev box, not just the fixtures: /api/authors is one of the 8 contract-fixtured
// endpoints (contracts/api/authors.json), /api/series is not, so its shape was read from a real signed-in call
// (`[{library_id, library_name, name, authors[], bucket, count, cover_url}]`) before this schema was written.
import { z } from "zod";
import { request } from "../../../lib/http";

const groupSchema = z.object({
  library_id: z.string(),
  library_name: z.string(),
  name: z.string(),
  /** Only series carry authors (the representative book's credits); authors have none. */
  authors: z.array(z.string()).optional(),
  bucket: z.string().optional(),
  count: z.number(),
  cover_url: z.string().optional(),
  unsorted: z.boolean().optional(),
});

export type GroupInfo = z.infer<typeof groupSchema>;
export type GroupKind = "series" | "authors";

export const GROUP_PAGE_SIZE = 120;

export interface GroupPage {
  items: GroupInfo[];
  /** True when the server returned a full page — i.e. there is probably another one. The real list size is in
      X-Total-Count, which `lib/http` deliberately doesn't expose, so a full page is the honest signal here. */
  hasMore: boolean;
}

export async function fetchGroups(kind: GroupKind, libraryId: string, offset = 0, limit = GROUP_PAGE_SIZE): Promise<GroupPage> {
  const params = new URLSearchParams({ library: libraryId, limit: String(limit), offset: String(offset) });
  const raw = await request<unknown>(`/api/${kind}?${params}`);
  const items = z.array(groupSchema).parse(raw);
  return { items, hasMore: items.length === limit };
}
