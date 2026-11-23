// features/library/api/media.ts — the non-book shelves. This server does **not** serve movies, TV or music from
// /api/books: `/api/books` is the *catalogue* (epub/cbz/pdf and audiobook files that were indexed as books), while
// video and audio live in their own stores behind their own endpoints. The first version of the video/music slice
// assumed books served everything, so those shelves rendered "No titles in this shelf yet." for a library holding
// 1388 tracks — the counts in /api/libraries are the proof, and they disagree with an empty /api/books.
//
//   video kind tv|anime → /api/video-shows   (one card per show: name, year, episode count, progress counts)
//   video kind movies    → /api/videos        (one card per film: title, year, poster, position/duration)
//   audio kinds          → /api/audio         (one card per track: title, artist, album, container, artwork)
//
// Everything is normalised to `ShelfItem` so the screens stay unaware of which endpoint answered.
import { z } from "zod";
import { requestEnvelope } from "../../../lib/http";

export interface ShelfItem {
  id: string;
  title: string;
  /** Year, artist or episode count — whatever this shelf uses as the second line. */
  subtitle: string;
  coverUrl: string;
  /** 0..1, only when the shelf has real progress for the item. */
  progress?: number | undefined;
  finished?: boolean | undefined;
  /** Resolution ("1080p") or container ("FLAC") printed on the artwork. */
  formatLabel?: string | undefined;
  /** Audio only: present so a tap can hand this straight to the global player (features/player/PlayerContext)
      without a second fetch. Undefined for video/show items, which still navigate to a detail screen. */
  streamUrl?: string | undefined;
}

export interface ShelfPage {
  items: ShelfItem[];
  total: number | null;
}

/** Video streams and metadata overrides are deep; only the fields a card prints are parsed, and the rest is
    tolerated so a richer server response can't break a shelf (`looseObject` on the containers). */
const videoItemSchema = z.looseObject({
  id: z.string(),
  title: z.string(),
  year: z.number().optional(),
  container: z.string().optional(),
  duration_seconds: z.number().optional(),
  video_streams: z.array(z.looseObject({ height: z.number().optional() })).default([]),
  artwork_url: z.string().optional(),
  metadata_override: z.looseObject({ title: z.string().optional(), artwork_url: z.string().optional(), year: z.number().optional() }).optional(),
  progress: z.looseObject({ position_seconds: z.number(), duration_seconds: z.number() }).optional(),
  // The three fields playback needs. `stream_url` is present only for DirectPlay files and `hls_url` for anything the
  // server transcodes — exactly one of them shows up, which is why the player takes both as optional.
  stream_url: z.string().optional(),
  hls_url: z.string().optional(),
  // Changes whenever the file is replaced; sent with progress so an old position cannot attach to a new file.
  revision: z.string().default(""),
  overview: z.string().optional(),
});

const showSchema = z.looseObject({
  name: z.string(),
  source_name: z.string().optional(),
  item_id: z.string(),
  count: z.number(),
  artwork_url: z.string().optional(),
  year: z.number().optional(),
  seasons: z.array(z.looseObject({ number: z.number(), count: z.number() })).default([]),
  metadata_override: z.looseObject({ title: z.string().optional(), artwork_url: z.string().optional(), year: z.number().optional() }).optional(),
});

export const audioItemSchema = z.looseObject({
  id: z.string(),
  title: z.string(),
  artist: z.string().optional(),
  album_artist: z.string().optional(),
  album: z.string().optional(),
  container: z.string().optional(),
  track: z.number().optional(),
  artwork_url: z.string().optional(),
  duration_seconds: z.number().optional(),
  stream_url: z.string().optional(),
  // Podcasts only: which feed an episode came from, and when it was published. Presence of `feed_id` is what
  // tells a podcast episode apart from an audiobook track on the same endpoint.
  feed_id: z.string().optional(),
  modified: z.string().optional(),
  // The account's own progress for this item, when it has any. This is what makes "unplayed" more than a guess:
  // the server filters on the same field (`matchesPlayedFilter`, internal/web/audio_http.go).
  progress: z
    .looseObject({
      position_seconds: z.number().optional(),
      duration_seconds: z.number().optional(),
      finished: z.boolean().optional(),
    })
    .optional(),
});

/** Shared by the flat shelf below and api/audioGroups.ts's group-tracks fetch — both endpoints answer the same
    audioView shape, and a track found via "Albums" must look identical to the same track found via "All tracks". */
export function toShelfAudioItem(item: z.infer<typeof audioItemSchema>): ShelfItem {
  const position = item.progress?.position_seconds ?? 0;
  const duration = item.progress?.duration_seconds ?? item.duration_seconds ?? 0;
  const fraction = duration > 0 ? position / duration : 0;
  return {
    id: item.id,
    title: item.title,
    // Artist for music, album for audiobooks (where the "artist" is usually the narrator), whichever exists.
    subtitle: item.artist ?? item.album ?? item.album_artist ?? "",
    coverUrl: item.artwork_url ?? "",
    formatLabel: item.container === undefined ? undefined : item.container.toUpperCase(),
    streamUrl: item.stream_url,
    // Same two thresholds the catalogue uses: a few seconds in is not progress, and 98% is finished. Kept
    // consistent so a half-heard podcast episode draws the same bar a half-read book does.
    progress: fraction > 0.001 && fraction < 0.98 ? fraction : undefined,
    finished: item.progress?.finished === true || fraction >= 0.98,
  };
}

/** "1080p" from the first video stream — real probe data, which is why the video card can show a quality label
    instead of the tag pill the mock-up rules out. Exported: the video detail screen prints the same label. */
export function resolutionLabel(item: { video_streams: { height?: number | undefined }[] }): string | undefined {
  const height = item.video_streams.find((stream) => typeof stream.height === "number")?.height;
  if (height === undefined) return undefined;
  // The usual ladder, rounded down: a 800-line encode is "720p" in every interface a person has ever used.
  if (height >= 2000) return "4K";
  if (height >= 1000) return "1080p";
  if (height >= 700) return "720p";
  if (height >= 400) return "480p";
  return `${height}p`;
}

function posterUrl(item: {
  artwork_url?: string | undefined;
  metadata_override?: { artwork_url?: string | undefined } | undefined;
}): string {
  // A matched provider poster wins over the one extracted from the file, matching what the legacy UI shows.
  return item.metadata_override?.artwork_url ?? item.artwork_url ?? "";
}

export async function fetchVideoItems(
  libraryId: string,
  options: { q?: string; offset?: number; limit?: number } = {},
): Promise<ShelfPage> {
  const { q, offset = 0, limit = 120 } = options;
  const params = new URLSearchParams({ library: libraryId, limit: String(limit), offset: String(offset) });
  if (q !== undefined && q !== "") params.set("q", q);
  const envelope = await requestEnvelope<unknown>(`/api/videos?${params}`);
  const items = z.array(videoItemSchema).parse(envelope.data).map((item): ShelfItem => {
    const duration = item.progress?.duration_seconds ?? item.duration_seconds ?? 0;
    const fraction = duration > 0 ? (item.progress?.position_seconds ?? 0) / duration : 0;
    return {
      id: item.id,
      // A matched provider title wins over the filename the scanner parsed.
      title: item.metadata_override?.title ?? item.title,
      subtitle: (item.metadata_override?.year ?? item.year)?.toString() ?? "",
      coverUrl: posterUrl(item),
      // 0.1% watched is not progress; the legacy player writes a position the moment playback starts.
      progress: fraction > 0.001 && fraction < 1 ? fraction : undefined,
      finished: fraction >= 0.98,
      formatLabel: resolutionLabel(item),
    };
  });
  return { items, total: envelope.total };
}

export async function fetchShows(
  libraryId: string,
  options: { q?: string; offset?: number; limit?: number } = {},
): Promise<ShelfPage> {
  const { q, offset = 0, limit = 120 } = options;
  const params = new URLSearchParams({ library: libraryId, limit: String(limit), offset: String(offset) });
  if (q !== undefined && q !== "") params.set("q", q);
  const envelope = await requestEnvelope<unknown>(`/api/video-shows?${params}`);
  const items = z.array(showSchema).parse(envelope.data).map((show): ShelfItem => {
    const seasons = show.seasons.length;
    const episodes = show.count === 1 ? "1 episode" : `${show.count} episodes`;
    return {
      id: show.item_id,
      title: show.metadata_override?.title ?? show.name,
      // Season count only when it's plural: "1 season" on every anime is noise.
      subtitle: [show.year !== undefined ? String(show.year) : "", seasons > 1 ? `${seasons} seasons` : ""]
        .filter((part) => part !== "")
        .join(" · ") || episodes,
      coverUrl: posterUrl(show),
      formatLabel: undefined,
    };
  });
  return { items, total: envelope.total };
}

/** `sort` and `played` are the server's own parameters (internal/web/audio_http.go — sortAudioItems and
 *  matchesPlayedFilter), not something this client filters: a podcast shelf is 1836 episodes, so hiding played
 *  ones has to happen before paging, or the first page is still full of them. */
export interface AudioShelfOptions {
  q?: string | undefined;
  offset?: number | undefined;
  limit?: number | undefined;
  sort?: "title" | "recent" | "oldest" | undefined;
  played?: "played" | "unplayed" | "continue" | undefined;
  /** One podcast show's episodes (apiAudio's `feed` parameter). */
  feed?: string | undefined;
}

export async function fetchAudioItems(libraryId: string, options: AudioShelfOptions = {}): Promise<ShelfPage> {
  const { q, offset = 0, limit = 120, sort, played, feed } = options;
  const params = new URLSearchParams({ library: libraryId, limit: String(limit), offset: String(offset) });
  if (q !== undefined && q !== "") params.set("q", q);
  if (sort !== undefined) params.set("sort", sort);
  if (played !== undefined) params.set("played", played);
  if (feed !== undefined) params.set("feed", feed);
  const envelope = await requestEnvelope<unknown>(`/api/audio?${params}`);
  const items = z.array(audioItemSchema).parse(envelope.data).map(toShelfAudioItem);
  return { items, total: envelope.total };
}
