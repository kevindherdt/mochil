// features/library/api/libraries.ts — typed endpoint + zod schema for /api/libraries, one of the 8
// contract-fixtured endpoints (internal/web/contract_fixtures_test.go) — the real server's response shape is
// pinned by that fixture, so this schema should never drift from reality silently.
import { z } from "zod";
import { request } from "../../../lib/http";

const librarySchema = z.object({
  id: z.string(),
  name: z.string(),
  kind: z.string(),
  count: z.number(),
  metadata_provider: z.string().optional(),
});

export type LibraryInfo = z.infer<typeof librarySchema>;

export async function fetchLibraries(): Promise<LibraryInfo[]> {
  const raw = await request<unknown>("/api/libraries");
  return z.array(librarySchema).parse(raw);
}
