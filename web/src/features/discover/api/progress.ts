// features/discover/api/progress.ts — the two rails Discover was missing: what you are *listening* to and what you
// are *watching*.
//
// Kevin: "on the discover page it still only shows continue reading, but no continue listening (podcasts and
// audiobooks etc.. everything audio except for music) and then no continue watching either". The progress stores
// already existed and already had a "continue" filter — `/api/audio-progress?continue=1` and
// `/api/video-progress?continue=1` both answer only un-dismissed, un-finished, positively-positioned items
// (internal/web/audio_progress_http.go, video_http.go's videoProgressViews). Discover simply never asked.
//
// Music is excluded from Continue listening on purpose: this server's music progress is *listening history*
// (the endpoint takes a `music_library` parameter and treats it as exactly that), not "half-heard album" — a rail of
// the last thirty songs you played would be history, not continuation.
import { z } from "zod";
import { request } from "../../../lib/http";

const audioProgressSchema = z.looseObject({
  id: z.string(),
  library_id: z.string(),
  library_kind: z.string().optional(),
  title: z.string(),
  artist: z.string().optional(),
  album: z.string().optional(),
  artwork_url: z.string().optional(),
  stream_url: z.string().optional(),
  duration_seconds: z.number().optional(),
  progress: z
    .looseObject({
      position_seconds: z.number().optional(),
      duration_seconds: z.number().optional(),
      finished: z.boolean().optional(),
    })
    .nullable()
    .optional(),
});

export interface ContinueAudio {
  id: string;
  libraryId: string;
  /** "audiobooks", "podcasts" — used for the card glyph and for the tap destination. */
  kind: string;
  title: string;
  subtitle: string;
  coverUrl: string;
  streamUrl: string | undefined;
  fraction: number;
}

export async function fetchContinueListening(limit = 12): Promise<ContinueAudio[]> {
  const raw = await request<unknown>(`/api/audio-progress?continue=1&limit=${limit}`);
  return z
    .array(audioProgressSchema)
    .parse(raw ?? [])
    .filter((item) => (item.library_kind ?? "").toLowerCase() !== "music")
    .map((item): ContinueAudio => {
      const position = item.progress?.position_seconds ?? 0;
      const duration = item.progress?.duration_seconds ?? item.duration_seconds ?? 0;
      return {
        id: item.id,
        libraryId: item.library_id,
        kind: item.library_kind ?? "",
        title: item.title,
        subtitle: item.artist ?? item.album ?? "",
        coverUrl: item.artwork_url ?? "",
        streamUrl: item.stream_url,
        // The progress store already dropped finished items; a zero duration (an un-probed file) shows no bar
        // rather than a full one, which is why this can be 0.
        fraction: duration > 0 ? Math.min(1, position / duration) : 0,
      };
    });
}

const videoProgressSchema = z.looseObject({
  id: z.string(),
  library_id: z.string(),
  library_kind: z.string().optional(),
  title: z.string(),
  year: z.number().optional(),
  show: z.string().optional(),
  season: z.number().optional(),
  episode: z.number().optional(),
  artwork_url: z.string().optional(),
  backdrop_url: z.string().optional(),
  metadata_override: z
    .looseObject({ title: z.string().optional(), artwork_url: z.string().optional() })
    .nullable()
    .optional(),
  progress: z.looseObject({ position_seconds: z.number().optional(), duration_seconds: z.number().optional() }).nullable().optional(),
});

export interface ContinueVideo {
  id: string;
  libraryId: string;
  title: string;
  subtitle: string;
  coverUrl: string;
  fraction: number;
}

export async function fetchContinueWatching(limit = 12): Promise<ContinueVideo[]> {
  const raw = await request<unknown>(`/api/video-progress?continue=1&limit=${limit}`);
  return z
    .array(videoProgressSchema)
    .parse(raw ?? [])
    .map((item): ContinueVideo => {
      const position = item.progress?.position_seconds ?? 0;
      const duration = item.progress?.duration_seconds ?? 0;
      const episode = item.episode === undefined ? "" : item.season === undefined ? `Episode ${item.episode}` : `S${item.season} · E${item.episode}`;
      return {
        id: item.id,
        libraryId: item.library_id,
        title: item.metadata_override?.title ?? item.title,
        subtitle: [item.show, episode].filter((part) => part !== undefined && part !== "").join(" · "),
        coverUrl: item.metadata_override?.artwork_url ?? item.artwork_url ?? "",
        fraction: duration > 0 ? Math.min(1, position / duration) : 0,
      };
    });
}
