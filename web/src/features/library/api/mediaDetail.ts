// features/library/api/mediaDetail.ts — single-item lookups for the detail screens. The server has no
// `/api/videos/{id}` or `/api/audio/{id}`: a video/audio item is only reachable by listing its library (optionally
// filtered by `q`/`show`/`season`) and finding the id in the page. `findInPages` does that paging walk once, so
// every detail fetch below is the same shape: ask for a page, check the total, ask for the next page if the item
// wasn't in this one. For movies and shows (a handful of items) this is one request; for a 1388-track music
// library it is at most three — bounded by the real total the server reports, never a guess.
//
// `/api/books/{id}` is a real single-item endpoint (findBook resolves across every library by id), so the book
// detail fetch below doesn't need this at all.
import { z } from "zod";
import { requestEnvelope } from "../../../lib/http";

async function findInPages<T>(
  page: (offset: number, limit: number) => Promise<{ items: T[]; total: number | null }>,
  matches: (item: T) => boolean,
  limit = 500,
): Promise<T | undefined> {
  let offset = 0;
  for (;;) {
    const { items, total } = await page(offset, limit);
    const found = items.find(matches);
    if (found !== undefined) return found;
    offset += items.length;
    if (items.length === 0 || (total !== null && offset >= total)) return undefined;
  }
}

const videoStreamSchema = z.looseObject({ height: z.number().optional() });

/** An audio or subtitle track, as the probe reported it. `text` is the one that decides whether a subtitle can be
 *  offered at all: the server converts text tracks to WebVTT and 404s image-based ones (PGS), so listing those would
 *  be offering a control that always fails. */
export const mediaTrackSchema = z.looseObject({
  index: z.number(),
  codec: z.string().optional(),
  language: z.string().optional(),
  title: z.string().optional(),
  channels: z.number().optional(),
  default: z.boolean().optional(),
  forced: z.boolean().optional(),
  text: z.boolean().optional(),
});

export type MediaTrack = z.infer<typeof mediaTrackSchema>;

const videoDetailSchema = z.looseObject({
  id: z.string(),
  title: z.string(),
  year: z.number().optional(),
  overview: z.string().optional(),
  show: z.string().optional(),
  season: z.number().optional(),
  episode: z.number().optional(),
  artwork_url: z.string().optional(),
  backdrop_url: z.string().optional(),
  stream_url: z.string().optional(),
  hls_url: z.string().optional(),
  video_streams: z.array(videoStreamSchema).default([]),
  audio_streams: z.array(mediaTrackSchema).default([]),
  subtitle_streams: z.array(mediaTrackSchema).default([]),
  metadata_override: z
    .looseObject({ title: z.string().optional(), overview: z.string().optional(), artwork_url: z.string().optional(), year: z.number().optional() })
    .optional(),
  progress: z.looseObject({ position_seconds: z.number(), duration_seconds: z.number() }).optional(),
  // Playback needs these two: `duration_seconds` for the fallback when there is no stored progress yet, and
  // `revision` so a saved position is pinned to this exact version of the file.
  duration_seconds: z.number().optional(),
  revision: z.string().default(""),
});

export type VideoDetail = z.infer<typeof videoDetailSchema>;

const showDetailSchema = z.looseObject({
  name: z.string(),
  source_name: z.string().optional(),
  item_id: z.string(),
  year: z.number().optional(),
  overview: z.string().optional(),
  artwork_url: z.string().optional(),
  backdrop_url: z.string().optional(),
  seasons: z.array(z.looseObject({ number: z.number(), count: z.number() })).default([]),
  metadata_override: z
    .looseObject({ title: z.string().optional(), overview: z.string().optional(), artwork_url: z.string().optional(), year: z.number().optional() })
    .optional(),
});

export type ShowDetail = z.infer<typeof showDetailSchema>;

const audioDetailSchema = z.looseObject({
  id: z.string(),
  title: z.string(),
  artist: z.string().optional(),
  album_artist: z.string().optional(),
  album: z.string().optional(),
  container: z.string().optional(),
  track: z.number().optional(),
  artwork_url: z.string().optional(),
  stream_url: z.string().optional(),
  description: z.string().optional(),
  // Real chapters, when the file carries them (an m4b with chapter markers): `audio.Item.Chapters`, probed by the
  // scanner and sent on the view. Most mp3 audiobooks have none, which is why this is optional rather than assumed.
  chapters: z
    .array(
      z.looseObject({
        title: z.string(),
        start_seconds: z.number(),
        end_seconds: z.number(),
      }),
    )
    .optional(),
});

export type AudioDetail = z.infer<typeof audioDetailSchema>;
export type AudioChapter = { title: string; start: number; end: number };

async function videoPage(libraryId: string, offset: number, limit: number): Promise<{ items: VideoDetail[]; total: number | null }> {
  const params = new URLSearchParams({ library: libraryId, limit: String(limit), offset: String(offset) });
  const envelope = await requestEnvelope<unknown>(`/api/videos?${params}`);
  return { items: z.array(videoDetailSchema).parse(envelope.data), total: envelope.total };
}

export async function fetchVideoDetail(libraryId: string, id: string): Promise<VideoDetail | undefined> {
  return findInPages((offset, limit) => videoPage(libraryId, offset, limit), (item) => item.id === id);
}

export async function fetchShowDetail(libraryId: string, itemId: string): Promise<ShowDetail | undefined> {
  const params = new URLSearchParams({ library: libraryId, limit: "500" });
  return findInPages(
    async (offset) => {
      params.set("offset", String(offset));
      const envelope = await requestEnvelope<unknown>(`/api/video-shows?${params}`);
      return { items: z.array(showDetailSchema).parse(envelope.data), total: envelope.total };
    },
    (show) => show.item_id === itemId,
  );
}

/** Episodes of one season (or every season, if `season` is omitted) — a direct, un-paged filter on `/api/videos`,
    unlike the id lookups above: a show rarely has more than a season's worth of rows, and the plan calls for
    exactly the season the reader picked, not the whole library scanned to find it. */
export async function fetchEpisodes(libraryId: string, showSourceName: string, season?: number): Promise<VideoDetail[]> {
  const params = new URLSearchParams({ library: libraryId, show: showSourceName, limit: "500" });
  if (season !== undefined) params.set("season", String(season));
  const envelope = await requestEnvelope<unknown>(`/api/videos?${params}`);
  return z.array(videoDetailSchema).parse(envelope.data);
}

export async function fetchAudioDetail(libraryId: string, id: string): Promise<AudioDetail | undefined> {
  return findInPages(
    async (offset, limit) => {
      const params = new URLSearchParams({ library: libraryId, limit: String(limit), offset: String(offset) });
      const envelope = await requestEnvelope<unknown>(`/api/audio?${params}`);
      return { items: z.array(audioDetailSchema).parse(envelope.data), total: envelope.total };
    },
    (item) => item.id === id,
  );
}
