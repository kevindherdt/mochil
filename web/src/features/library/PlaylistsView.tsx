// features/library/PlaylistsView.tsx — the account's playlists, and the tracks inside one.
//
// Playlists are account-wide in the API (`/api/audio-playlists` is keyed by account, not by library), so these views
// are too: one list, wherever the tracks came from. Only music libraries can contribute entries, which is why the
// shelf's button appears for music only.
//
// Two behaviours here are choices, not defaults:
//
//  * **Reordering is up/down buttons over a whole-list PUT.** The server's contract is "these are the entries, in
//    this order", so a move is one request that cannot half-apply. Drag would need a drag primitive this app does
//    not have, and would be unusable with a keyboard — the editor already set the keyboard bar.
//  * **Every write quotes the revision it read**, and the revision the server answers with replaces it. Without the
//    second half, two moves in a row send the second with a stale ETag and fail with 412 against the screen's own
//    previous write, which reads like a conflict with another device and is not one.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowDown, ArrowUp, ListMusic, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Async, Button, Card, GroupCard, Inline, MediaGrid, Stack, Text, TextField } from "../../ui";
import {
  appendToPlaylist,
  createPlaylist,
  deletePlaylist,
  fetchPlaylist,
  fetchPlaylistSummaries,
  fetchPlaylists,
  PlaylistConflictError,
  PlaylistNameError,
  PlaylistNameTakenError,
  renamePlaylist,
  replacePlaylist,
  type Playlist,
  type PlaylistEntry,
} from "./api/playlists";

const playlistsKey = ["playlists"] as const;

/** What a dialog should say about a failed write: the server's own reason when it has one, in plain words. */
function describe(error: unknown): string {
  if (error instanceof PlaylistNameError) return error.message;
  if (error instanceof PlaylistConflictError) return "This playlist changed on another device. Reload to see its current tracks.";
  if (error instanceof Error && error.message !== "") return error.message;
  return "That did not save.";
}

export function PlaylistsList({ onOpen }: { onOpen: (name: string) => void }) {
  const client = useQueryClient();
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [problem, setProblem] = useState<string | null>(null);
  const query = useQuery({ queryKey: playlistsKey, queryFn: fetchPlaylists, staleTime: 30_000 });

  const create = useMutation({
    mutationFn: createPlaylist,
    onSuccess: async (_result, created) => {
      setProblem(null);
      setName("");
      setCreating(false);
      await client.invalidateQueries({ queryKey: playlistsKey });
      onOpen(created);
    },
    onError: (error: unknown) => setProblem(describe(error)),
  });

  return (
    <Stack gap={16}>
      <Inline gap={8} align="center">
        <Button
          variant="primary"
          onClick={() => {
            setProblem(null);
            setCreating((value) => !value);
          }}
        >
          <Plus size={15} /> New playlist
        </Button>
        {!creating && (
          <Text size="13" muted>
            A playlist holds tracks from any music shelf.
          </Text>
        )}
      </Inline>

      {creating && (
        <Card>
          <Stack gap={8}>
            <TextField
              label="Playlist name"
              value={name}
              autoFocus
              onChange={(event) => setName(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && name.trim() !== "") create.mutate(name.trim());
                if (event.key === "Escape") setCreating(false);
              }}
            />
            <Inline gap={8} align="center">
              <Button variant="primary" disabled={name.trim() === "" || create.isPending} onClick={() => create.mutate(name.trim())}>
                {create.isPending ? "Creating…" : "Create"}
              </Button>
              <Button variant="secondary" onClick={() => setCreating(false)}>
                Cancel
              </Button>
              {problem !== null && (
                <Text size="13" tone="error">
                  {problem}
                </Text>
              )}
            </Inline>
          </Stack>
        </Card>
      )}

      <Async query={query} empty={(lists) => lists.length === 0} emptyMessage="No playlists yet. Make one, then add tracks from any album.">
        {(lists) => (
          <MediaGrid>
            {lists.map((list) => (
              <GroupCard
                key={list.name}
                name={list.name}
                count={list.tracks.length}
                countLabel={list.tracks.length === 1 ? "1 track" : `${list.tracks.length} tracks`}
                coverUrl={list.tracks[0]?.coverUrl ?? ""}
                kindIcon={ListMusic}
                onClick={() => onOpen(list.name)}
              />
            ))}
          </MediaGrid>
        )}
      </Async>
    </Stack>
  );
}

export function PlaylistDetail({
  name,
  onBack,
  onPlay,
}: {
  name: string;
  onBack: () => void;
  onPlay: (playlist: Playlist, startIndex: number) => void;
}) {
  const client = useQueryClient();
  const [problem, setProblem] = useState<string | null>(null);
  const [renaming, setRenaming] = useState(false);
  const [draftName, setDraftName] = useState(name);
  const query = useQuery({ queryKey: ["playlist", name], queryFn: () => fetchPlaylist(name), staleTime: 0 });

  // One mutation for every edit, because they all send the same whole list with the same revision: the difference
  // between a move, a removal and an undo is only which array is sent.
  const write = useMutation({
    mutationFn: (entries: PlaylistEntry[]) => replacePlaylist(name, entries, query.data?.revision ?? null),
    // No revision juggling here, and that is deliberate. A *slow* refetch cannot be clicked through: this handler is
    // awaited, so the mutation stays pending and the buttons stay disabled. A *failed* one does not leave a stale list
    // to edit either — this screen is the one <Async> consumer that passes `failOnRefetchError`, so a failed refetch
    // swaps in the error panel and Retry is the only way forward, which brings a fresh revision with it. (Everywhere
    // else keeps the loaded data and says it could not refresh; here a stale revision makes the next edit fail.)
    // An earlier draft adopted the PUT's ETag into the cache for that window; a test written to
    // exercise it showed the window does not exist, so the code is gone rather than kept "just in case".
    onSuccess: async () => {
      setProblem(null);
      await query.refetch();
      await client.invalidateQueries({ queryKey: playlistsKey });
    },
    onError: async (error: unknown) => {
      setProblem(describe(error));
      await query.refetch();
    },
  });

  const rename = useMutation({
    mutationFn: (next: string) => renamePlaylist(name, next, query.data?.revision ?? null),
    onSuccess: async () => {
      setRenaming(false);
      await client.invalidateQueries({ queryKey: playlistsKey });
      await client.invalidateQueries({ queryKey: ["playlist", name] });
      onBack();
      setProblem(null);
    },
    onError: (error: unknown) => setProblem(describe(error)),
  });

  const remove = useMutation({
    mutationFn: () => deletePlaylist(name, query.data?.revision ?? null),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: playlistsKey });
      onBack();
    },
    onError: (error: unknown) => setProblem(describe(error)),
  });

  const entries = (playlist: Playlist): PlaylistEntry[] => playlist.tracks.map((track) => ({ libraryId: track.libraryId, itemId: track.itemId }));

  const move = (playlist: Playlist, from: number, to: number) => {
    if (to < 0 || to >= playlist.tracks.length) return;
    const next = entries(playlist);
    const [moved] = next.splice(from, 1);
    if (moved === undefined) return;
    next.splice(to, 0, moved);
    write.mutate(next);
  };

  return (
    <Stack gap={12}>
      <Inline gap={8} align="center">
        <Button variant="secondary" onClick={onBack}>
          ← All playlists
        </Button>
        <Text size="16">{name}</Text>
        <Inline gap={4} align="center">
          <Button variant="secondary" onClick={() => setRenaming((value) => !value)}>
            Rename
          </Button>
          <Button variant="secondary" disabled={remove.isPending} onClick={() => remove.mutate()}>
            <Trash2 size={15} /> Delete
          </Button>
        </Inline>
      </Inline>

      {renaming && (
        <Card>
          <Stack gap={8}>
            <TextField label="New name" value={draftName} autoFocus onChange={(event) => setDraftName(event.target.value)} />
            <Inline gap={8}>
              <Button
                variant="primary"
                disabled={draftName.trim() === "" || draftName.trim() === name || rename.isPending}
                onClick={() => rename.mutate(draftName.trim())}
              >
                Save name
              </Button>
              <Button variant="secondary" onClick={() => setRenaming(false)}>
                Cancel
              </Button>
            </Inline>
          </Stack>
        </Card>
      )}

      {problem !== null && (
        <Text size="13" tone="error">
          {problem}
        </Text>
      )}

      <Async query={query} failOnRefetchError empty={(playlist) => playlist.tracks.length === 0} emptyMessage="This playlist has no tracks yet. Open an album and add one.">
        {(playlist) => (
          <Stack gap={8}>
            <Inline gap={8} align="center">
              <Button variant="primary" onClick={() => onPlay(playlist, 0)}>
                Play all
              </Button>
              <Text size="13" muted>
                {playlist.tracks.length === 1 ? "1 track" : `${playlist.tracks.length} tracks`}
              </Text>
            </Inline>
            <Stack gap={4}>
              {playlist.tracks.map((track, index) => (
                <Card key={`${track.libraryId}/${track.itemId}/${index}`}>
                  <Inline gap={8} align="center" justify="between">
                    <Inline gap={8} align="center">
                      <Text size="13" muted>
                        {index + 1}
                      </Text>
                      <Button variant="secondary" onClick={() => onPlay(playlist, index)}>
                        {track.title}
                      </Button>
                      <Text size="13" muted>
                        {track.artist !== "" ? track.artist : track.album}
                      </Text>
                    </Inline>
                    <Inline gap={4} align="center">
                      <Button variant="secondary" aria-label={`Move ${track.title} up`} disabled={index === 0 || write.isPending} onClick={() => move(playlist, index, index - 1)}>
                        <ArrowUp size={15} />
                      </Button>
                      <Button
                        variant="secondary"
                        aria-label={`Move ${track.title} down`}
                        disabled={index === playlist.tracks.length - 1 || write.isPending}
                        onClick={() => move(playlist, index, index + 1)}
                      >
                        <ArrowDown size={15} />
                      </Button>
                      <Button
                        variant="secondary"
                        aria-label={`Remove ${track.title}`}
                        disabled={write.isPending}
                        onClick={() => write.mutate(entries(playlist).filter((_, at) => at !== index))}
                      >
                        <Trash2 size={15} />
                      </Button>
                    </Inline>
                  </Inline>
                </Card>
              ))}
            </Stack>
          </Stack>
        )}
      </Async>
    </Stack>
  );
}

/** The "add to playlist" control for one track. Loaded lazily by the caller so a tracklist does not fetch every
 *  playlist name for a menu nobody opened. */
export function AddToPlaylistMenu({ entry, label = "Add to playlist" }: { entry: PlaylistEntry; label?: string }) {
  const client = useQueryClient();
  const [problem, setProblem] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const summaries = useQuery({ queryKey: [...playlistsKey, "summary"], queryFn: fetchPlaylistSummaries, enabled: pickerOpen });

  const add = useMutation({
    mutationFn: async (name: string) => {
      // A name that does not exist yet is created first: one gesture instead of "create, then add", and the server's
      // own 409 covers the race where someone else made it in between.
      if (summaries.data?.every((list) => list.name !== name) ?? true) {
        try {
          await createPlaylist(name);
        } catch (error) {
          // Taken is fine here — the playlist exists, which is what this call wanted. Anything else (an unusable
          // name, a dead session) is a real failure and belongs in front of the person, not swallowed.
          if (!(error instanceof PlaylistNameTakenError)) throw error;
        }
      }
      await appendToPlaylist(name, entry);
    },
    onSuccess: async () => {
      setProblem(null);
      setNewName("");
      setPickerOpen(false);
      await client.invalidateQueries({ queryKey: playlistsKey });
    },
    onError: (error: unknown) => setProblem(describe(error)),
  });

  return (
    <Stack gap={4}>
      <Button variant="secondary" aria-label={label} onClick={() => setPickerOpen((value) => !value)}>
        <Plus size={15} /> Add to playlist
      </Button>
      {pickerOpen && (
        <Card>
          <Stack gap={4}>
            <Async query={summaries}>
              {(lists) => (
                <Stack gap={4}>
                  {lists.map((list) => (
                    <Button
                      key={list.name}
                      variant="secondary"
                      aria-label={`Add to ${list.name}`}
                      disabled={add.isPending}
                      onClick={() => add.mutate(list.name)}
                    >
                      {list.name} · {list.count === 1 ? "1 track" : `${list.count} tracks`}
                    </Button>
                  ))}
                </Stack>
              )}
            </Async>
            <Inline gap={4} align="center">
              <TextField label="New playlist" value={newName} onChange={(event) => setNewName(event.target.value)} />
              <Button variant="primary" disabled={newName.trim() === "" || add.isPending} onClick={() => add.mutate(newName.trim())}>
                Create and add
              </Button>
            </Inline>
            {problem !== null && (
              <Text size="13" tone="error">
                {problem}
              </Text>
            )}
          </Stack>
        </Card>
      )}
    </Stack>
  );
}
