// features/settings/api/diagnostics.ts — GET /api/diagnostics (internal/web/diagnostics.go), administrator-only.
//
// This is the server's own rolling view of itself: uptime, request and failure counts, memory, goroutines, how many
// libraries it holds, the slowest recent requests, the recent failures, and any background job that has been failing
// (internal/safely records those per task). It is read-only by design — the endpoint answers GET and nothing else —
// so the panel reports rather than repairs.
import { z } from "zod";
import { request } from "../../../lib/http";

const recordSchema = z.object({
  at: z.number(),
  method: z.string(),
  path: z.string(),
  status: z.number(),
  duration_ms: z.number(),
});

const backgroundFailureSchema = z.object({
  what: z.string(),
  at: z.number(),
  message: z.string(),
  count: z.number(),
  restarts: z.number(),
});

export const diagnosticsSchema = z.object({
  uptime_seconds: z.number(),
  requests: z.number(),
  failures: z.number(),
  heap_mb: z.number(),
  system_mb: z.number(),
  goroutines: z.number(),
  gc_cycles: z.number(),
  libraries: z.number(),
  slow: z.array(recordSchema),
  errors: z.array(recordSchema),
  background: z.array(backgroundFailureSchema),
});

export type Diagnostics = z.infer<typeof diagnosticsSchema>;
export type DiagnosticsRecord = z.infer<typeof recordSchema>;

export async function fetchDiagnostics(): Promise<Diagnostics> {
  return diagnosticsSchema.parse(await request<unknown>("/api/diagnostics"));
}

/** "3h 12m" — an uptime a person can read at a glance. Seconds are dropped above an hour, because at that scale
 *  they are noise; below it they are the only interesting part. */
export function formatUptime(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${Math.round(seconds % 60)}s`;
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) return `${days}d ${hours % 24}h`;
  return `${hours}h ${minutes % 60}m`;
}
