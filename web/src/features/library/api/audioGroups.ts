// features/library/api/audioGroups.ts — "Albums"/"Artists" for music and "Series"/"Authors" for audiobooks, the
// audio equivalent of api/groups.ts's /api/series and /api/authors. Backed by the server's existing
// /api/audio-groups + /api/audio-group-tracks (the same endpoints the Android app's grouped browse already uses —
// `androidAudioGroupingAllowed` in internal/web/android_audio.go is the source of truth for which `group_type`
// values a library kind accepts: music → albums|artists, audiobooks → series|authors; anything else 404s, so
// podcasts stay a flat shelf for now rather than guessing at a grouping the server can't answer).
//
// Known limitation, not hidden: tapping an "Artists" group lists every track by that artist flat (natural title
// order), not nested further into that artist's albums. The server has a third endpoint for that
// (/api/audio-artist-albums) — a real "artist → albums → tracks" drill is a reasonable follow-up, not built here.
import { z } from "zod";
import { request } from "../../../lib/http";
import { audioItemSchema, toShelfAudioItem, type ShelfItem } from "./media";

export type AudioGroupKind = "albums" | "artists" | "series" | "authors";

export interface AudioGroupInfo {
  id: string;
  name: string;
  subtitle: string;
  count: number;
  coverUrl: string | undefined;
}

const audioGroupSchema = z.object({
  id: z.string(),
  name: z.string(),
  subtitle: z.string().optional(),
  count: z.number(),
  artwork_item_id: z.string().optional(),
  artwork_library_id: z.string().optional(),
});

export interface AudioGroupPage {
  items: AudioGroupInfo[];
  /** Same heuristic as api/groups.ts: a full page probably has a next one; the server's X-Total-Count isn't
      surfaced by `request()`, and a "probably more" signal is enough for a Show-more button. */
  hasMore: boolean;
}

export const AUDIO_GROUP_PAGE_SIZE = 60; // internal/web/android_audio.go's boundedAndroidAudioPage default

function groupArtworkUrl(group: z.infer<typeof audioGroupSchema>): string | undefined {
  if (group.artwork_library_id === undefined || group.artwork_item_id === undefined) return undefined;
  return `/api/audio-artwork/${encodeURIComponent(group.artwork_library_id)}/${encodeURIComponent(group.artwork_item_id)}?size=card`;
}

export async function fetchAudioGroups(
  libraryId: string,
  groupType: AudioGroupKind,
  q = "",
  offset = 0,
  limit = AUDIO_GROUP_PAGE_SIZE,
): Promise<AudioGroupPage> {
  const params = new URLSearchParams({ library: libraryId, group_type: groupType, limit: String(limit), offset: String(offset) });
  if (q !== "") params.set("q", q);
  const raw = await request<unknown>(`/api/audio-groups?${params}`);
  const items = z.array(audioGroupSchema).parse(raw).map((group): AudioGroupInfo => ({
    id: group.id,
    name: group.name,
    subtitle: group.subtitle ?? "",
    count: group.count,
    coverUrl: groupArtworkUrl(group),
  }));
  return { items, hasMore: items.length === limit };
}

/** Every track in one group, in the server's own order (album/series: disc+track number; artist/author: title) —
    not paged, since a tracklist the reader just picked is small enough to show whole. */
export async function fetchAudioGroupTracks(libraryId: string, groupType: AudioGroupKind, groupId: string): Promise<ShelfItem[]> {
  const params = new URLSearchParams({ library: libraryId, group_type: groupType, group_id: groupId });
  const raw = await request<unknown>(`/api/audio-group-tracks?${params}`);
  return z.array(audioItemSchema).parse(raw).map(toShelfAudioItem);
}
