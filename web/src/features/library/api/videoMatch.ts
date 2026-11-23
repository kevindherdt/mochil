// features/library/api/videoMatch.ts — hand-matching one film or show to TMDB.
//
// Two halves, both already on the server and neither wired to /next:
//   GET  /api/video-metadata-search/<lib>?q=&show=1   → candidate titles (new: internal/web/video_metadata_search_http.go)
//   PUT  /api/video-metadata/<lib>/<itemId>           → save the chosen override
//   DELETE /api/video-metadata/<lib>/<itemId>         → forget it
//   PUT/DELETE /api/video-show-metadata/<lib>?show=  → the same for a whole series
//
// The batch matcher (`/api/video-metadata-match`, ?apply=1) is deliberately not used by the UI: it is the "big
// button" Kevin asked to get rid of ("it should be out of the way somewhere or being able to click on the
// series/movie/ etc and have a edit button like a pencil"). It stays available for scripts.
import { z } from "zod";
import { request } from "../../../lib/http";

export interface MatchCandidate {
  providerId: string;
  title: string;
  year: number | undefined;
  overview: string;
  posterUrl: string;
  backdropUrl: string;
}

const candidateSchema = z.looseObject({
  provider_id: z.string(),
  title: z.string(),
  year: z.number().optional(),
  overview: z.string().optional(),
  poster_url: z.string().optional(),
  backdrop_url: z.string().optional(),
});

export async function searchMatches(libraryId: string, query: string, show: boolean): Promise<MatchCandidate[]> {
  const params = new URLSearchParams({ q: query });
  if (show) params.set("show", "1");
  const raw = await request<{ results?: unknown }>(`/api/video-metadata-search/${encodeURIComponent(libraryId)}?${params}`);
  return z
    .array(candidateSchema)
    .parse(raw.results ?? [])
    .map((row): MatchCandidate => ({
      providerId: row.provider_id,
      title: row.title,
      year: row.year,
      overview: row.overview ?? "",
      posterUrl: row.poster_url ?? "",
      backdropUrl: row.backdrop_url ?? "",
    }));
}

export interface VideoOverride {
  title: string;
  year: number | undefined;
  overview: string;
  artworkUrl: string;
  backdropUrl: string;
  providerId: string;
}

const overrideSchema = z
  .looseObject({
    title: z.string().optional(),
    year: z.number().optional(),
    overview: z.string().optional(),
    artwork_url: z.string().optional(),
    backdrop_url: z.string().optional(),
    provider_id: z.string().optional(),
    provider: z.string().optional(),
  })
  .nullable()
  .optional();

function toOverride(value: z.infer<typeof overrideSchema>): VideoOverride | undefined {
  if (value === null || value === undefined) return undefined;
  return {
    title: value.title ?? "",
    year: value.year,
    overview: value.overview ?? "",
    artworkUrl: value.artwork_url ?? "",
    backdropUrl: value.backdrop_url ?? "",
    providerId: value.provider_id ?? "",
  };
}

/** The override currently saved for one item, or undefined when it has none. */
export async function readItemOverride(libraryId: string, itemId: string): Promise<VideoOverride | undefined> {
  const raw = await request<{ metadata?: unknown }>(`/api/video-metadata/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}`);
  return toOverride(overrideSchema.parse(raw.metadata));
}

export async function readShowOverride(libraryId: string, show: string): Promise<VideoOverride | undefined> {
  const raw = await request<{ metadata?: unknown }>(`/api/video-show-metadata/${encodeURIComponent(libraryId)}?show=${encodeURIComponent(show)}`);
  return toOverride(overrideSchema.parse(raw.metadata));
}

function body(override: VideoOverride) {
  return {
    title: override.title,
    ...(override.year === undefined ? {} : { year: override.year }),
    overview: override.overview,
    artwork_url: override.artworkUrl,
    backdrop_url: override.backdropUrl,
    provider_id: override.providerId,
    provider: "TMDB",
  };
}

export async function saveItemOverride(libraryId: string, itemId: string, override: VideoOverride): Promise<void> {
  await request<unknown>(`/api/video-metadata/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}`, {
    method: "PUT",
    body: body(override),
  });
}

export async function clearItemOverride(libraryId: string, itemId: string): Promise<void> {
  await request<unknown>(`/api/video-metadata/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}`, { method: "DELETE" });
}

export async function saveShowOverride(libraryId: string, show: string, override: VideoOverride): Promise<void> {
  await request<unknown>(`/api/video-show-metadata/${encodeURIComponent(libraryId)}?show=${encodeURIComponent(show)}`, {
    method: "PUT",
    body: body(override),
  });
}

export async function clearShowOverride(libraryId: string, show: string): Promise<void> {
  await request<unknown>(`/api/video-show-metadata/${encodeURIComponent(libraryId)}?show=${encodeURIComponent(show)}`, { method: "DELETE" });
}

/** A candidate as the override it would become. */
export function candidateToOverride(candidate: MatchCandidate): VideoOverride {
  return {
    title: candidate.title,
    year: candidate.year,
    overview: candidate.overview,
    artworkUrl: candidate.posterUrl,
    backdropUrl: candidate.backdropUrl,
    providerId: candidate.providerId,
  };
}
