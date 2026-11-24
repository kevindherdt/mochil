// features/library/api/playlists.ts — the account's own playlists.
//
// This API never went away: `/api/audio-playlists` is account-scoped and revision-guarded, and the Android app
// still uses it. The legacy music page was the only *web* client, so retiring that page took playlist creation,
// renaming, reordering and deletion off the web while leaving the machinery in place. This module is the client
// for what the server already answers:
//
//   GET    /api/audio-playlists                 → every playlist, each with its resolved tracks
//   GET    /api/audio-playlists?view=summary    → the same list, thinner ({version, playlists:[{name, item_count, ...}]})
//   POST   /api/audio-playlists {name}          → create (409 = name taken, 400 = unusable name)
//   GET    /api/audio-playlists/{name}          → one playlist + ETag
//   PATCH  /api/audio-playlists/{name} {name}   → rename       (If-Match required, 412 = changed elsewhere)
//   POST   /api/audio-playlists/{name} {entry}  → append one track
//   PUT    /api/audio-playlists/{name} [entry]  → replace the whole list in order (If-Match required)
//   DELETE /api/audio-playlists/{name}          → delete       (If-Match required)
//
// Every write quotes the revision it read. That is the server's contract, not a nicety: an edit that guesses the
// revision either overwrites what another device wrote or fails with 412, and the 412 is by far the better outcome —
// which is why `PlaylistConflictError` is a distinct type the screens can report instead of a generic failure.
//
// Only *music* libraries can hold playlist entries (`validMusicEntry` rejects anything else with 400), so the
// screens only offer this for music shelves rather than presenting a menu that always fails.
import { z } from "zod";
import { HttpError, request, requestEnvelope } from "../../../lib/http";
import { audioItemSchema } from "./media";

const playlistItemSchema = audioItemSchema.extend({
  library_id: z.string(),
  library_kind: z.string().optional(),
  stream_url: z.string().optional(),
});

const playlistSchema = z.object({
  name: z.string(),
  items: z.array(playlistItemSchema).default([]),
});

const summarySchema = z.object({
  version: z.number().optional(),
  playlists: z
    .array(
      z.object({
        name: z.string(),
        item_count: z.number().default(0),
        artwork_urls: z.array(z.string()).default([]),
      }),
    )
    .default([]),
});

/** One entry as the server stores it: which library, which track. Nothing else is accepted. */
export interface PlaylistEntry {
  libraryId: string;
  itemId: string;
}

export interface PlaylistTrack extends PlaylistEntry {
  title: string;
  artist: string;
  album: string;
  coverUrl: string;
  streamUrl: string | undefined;
  durationSeconds: number | undefined;
}

export interface Playlist {
  name: string;
  tracks: PlaylistTrack[];
  /** The ETag this was read at — null only if the server answered without one, which would make writes a guess. */
  revision: string | null;
}

export interface PlaylistSummary {
  name: string;
  count: number;
  coverUrl: string;
}

/** The name is taken (409) or unusable (400). Distinct so a dialog can say which, instead of "something failed". */
export class PlaylistNameError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PlaylistNameError";
  }
}

/** Someone else changed the playlist (412) or it is gone (404): the screen refetches instead of retrying blind. */
export class PlaylistConflictError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PlaylistConflictError";
  }
}

function toTrack(item: z.infer<typeof playlistItemSchema>): PlaylistTrack {
  return {
    libraryId: item.library_id,
    itemId: item.id,
    title: item.title,
    artist: item.artist ?? item.album_artist ?? "",
    album: item.album ?? "",
    coverUrl: item.artwork_url ?? "",
    streamUrl: item.stream_url,
    durationSeconds: item.duration_seconds,
  };
}

function toPlaylist(raw: unknown, revision: string | null): Playlist {
  const parsed = playlistSchema.parse(raw);
  return { name: parsed.name, revision, tracks: parsed.items.map(toTrack) };
}

/** The whole list, with tracks. Used by the shelf, which draws a cover ring from the first four tracks. */
export async function fetchPlaylists(): Promise<Playlist[]> {
  const raw = await request<unknown>("/api/audio-playlists");
  return z.array(playlistSchema).parse(raw ?? []).map((list) => ({ name: list.name, revision: null, tracks: list.items.map(toTrack) }));
}

/** Names and counts only — for the "add to playlist" picker, which must not download every tracklist. */
export async function fetchPlaylistSummaries(): Promise<PlaylistSummary[]> {
  const raw = await request<unknown>("/api/audio-playlists?view=summary");
  return summarySchema.parse(raw).playlists.map((list) => ({
    name: list.name,
    count: list.item_count,
    // The server sends up to four covers so a card can show a stack; the first is the one this app draws.
    coverUrl: list.artwork_urls[0] ?? "",
  }));
}

export async function fetchPlaylist(name: string): Promise<Playlist> {
  const envelope = await requestEnvelope<unknown>(`/api/audio-playlists/${encodeURIComponent(name)}`);
  return toPlaylist(envelope.data, envelope.etag);
}

export async function createPlaylist(name: string): Promise<void> {
  try {
    await request(`/api/audio-playlists`, { method: "POST", body: { name } });
  } catch (error) {
    throw asNameError(error);
  }
}

export async function renamePlaylist(from: string, to: string, revision: string | null): Promise<void> {
  try {
    await request(`/api/audio-playlists/${encodeURIComponent(from)}`, {
      method: "PATCH",
      body: { name: to },
      headers: revisionHeaders(revision),
    });
  } catch (error) {
    throw asNameError(error);
  }
}

export async function deletePlaylist(name: string, revision: string | null): Promise<void> {
  try {
    await request(`/api/audio-playlists/${encodeURIComponent(name)}`, {
      method: "DELETE",
      headers: revisionHeaders(revision),
    });
  } catch (error) {
    throw asConflictError(error);
  }
}

/** Appends one track. The server answers 400 for an item that is not a track in a music library. */
export async function appendToPlaylist(name: string, entry: PlaylistEntry): Promise<void> {
  try {
    await request(`/api/audio-playlists/${encodeURIComponent(name)}`, {
      method: "POST",
      body: { library_id: entry.libraryId, item_id: entry.itemId },
    });
  } catch (error) {
    throw asNameError(error);
  }
}

/**
 * Replaces the whole list, in order — the one call that reorders and the one that removes a track.
 *
 * Returns the revision the server answered with, so the next move quotes the new one instead of the revision this
 * screen read before its own write. Without that, two moves in a row would send the second with a stale ETag and
 * fail with 412 against itself.
 */
export async function replacePlaylist(name: string, entries: PlaylistEntry[], revision: string | null): Promise<string | null> {
  try {
    const envelope = await requestEnvelope(`/api/audio-playlists/${encodeURIComponent(name)}`, {
      method: "PUT",
      body: entries.map((entry) => ({ library_id: entry.libraryId, item_id: entry.itemId })),
      headers: revisionHeaders(revision),
    });
    return envelope.etag;
  } catch (error) {
    throw asConflictError(error);
  }
}

/** `If-Match` for a write. An empty ETag is sent as-is: the server treats a missing revision as "overwrite", which is
 *  what the create-then-immediately-edit path needs, and quoting a made-up value would fail every time. */
function revisionHeaders(revision: string | null): Record<string, string> {
  return revision === null ? {} : { "If-Match": revision };
}

function asNameError(error: unknown): unknown {
  if (error instanceof HttpError && (error.status === 409 || error.status === 400)) {
    return new PlaylistNameError(error.message || "That name cannot be used.");
  }
  return error;
}

function asConflictError(error: unknown): unknown {
  if (error instanceof HttpError && (error.status === 412 || error.status === 404)) {
    return new PlaylistConflictError(error.message || "This playlist changed elsewhere.");
  }
  return error;
}
