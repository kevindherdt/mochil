// features/about/api/version.ts — typed endpoint + zod schema for /api/version (plan §5.2). Deliberately NOT
// one of the 8 contract-fixtured endpoints (internal/web/contract_fixtures_test.go's header explains why:
// started/uptime/pid are real-clock values that can never be pinned to a deterministic fixture) — the schema
// below is validated by hand against a live `curl /api/version` instead.
import { z } from "zod";
import { request } from "../../../lib/http";

const versionSchema = z.object({
  product: z.string(),
  version: z.string(),
  commit: z.string().optional(),
  build_time: z.string().optional(),
  binary_time: z.string().optional(),
  started: z.string(),
  uptime: z.string(),
  go: z.string().optional(),
  pid: z.number(),
});

export type VersionInfo = z.infer<typeof versionSchema>;

export async function fetchVersion(): Promise<VersionInfo> {
  const raw = await request<unknown>("/api/version");
  return versionSchema.parse(raw);
}
