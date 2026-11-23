// features/activity/api/jobs.ts — the server's job board (`/api/jobs`, internal/web/jobs_http.go).
//
// Every queue the server runs reports through this one shape: background scans, podcast downloads, video and audio
// transcoding, audio analysis. It is the only place a person can see *why* something is not appearing yet — a book
// stuck behind a scan, an episode downloading, a film still transcoding — which is why it belongs in the app rather
// than only in the logs.
import { z } from "zod";
import { request } from "../../../lib/http";

const jobSchema = z.looseObject({
  queue: z.string(),
  key: z.string(),
  status: z.string(),
  queued_at: z.string().optional(),
  started_at: z.string().optional(),
  finished_at: z.string().optional(),
  progress_current: z.number().optional(),
  progress_total: z.number().optional(),
  progress_message: z.string().optional(),
  error: z.string().optional(),
});

export type Job = z.infer<typeof jobSchema>;

export async function fetchJobs(): Promise<Job[]> {
  const raw = await request<unknown>("/api/jobs");
  return z.array(jobSchema).parse(raw ?? []);
}

/** A job the server is still working on. `queued`/`running` are the two states the queues use for "not done". */
export function isActive(job: Job): boolean {
  return job.status === "queued" || job.status === "running" || job.status === "started";
}

/** 0..1 when the server reports both counters, otherwise undefined (a queue with no byte counts). */
export function jobProgress(job: Job): number | undefined {
  const total = job.progress_total ?? 0;
  const current = job.progress_current ?? 0;
  if (total <= 0) return undefined;
  return Math.max(0, Math.min(1, current / total));
}
