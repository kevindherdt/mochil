// features/library/screens/PlaylistsScreen.tsx — the two playlist routes.
//
// Thin by design: the list and the tracklist live in features/library/PlaylistsView, because both are composed from
// the same primitives a shelf uses and neither owns layout. These screens exist to read the route, hand the player a
// queue, and navigate — which is what a screen is allowed to do under the layering rule.
import { useNavigate, useParams } from "@tanstack/react-router";
import { usePlayer, type NowPlayingTrack } from "../../player/PlayerContext";
import { Page, Stack } from "../../../ui";
import { PlaylistDetail, PlaylistsList } from "../PlaylistsView";
import type { Playlist } from "../api/playlists";

export function PlaylistsScreen() {
  const navigate = useNavigate();
  return (
    <Page>
      <Stack gap={16}>
        <PlaylistsList onOpen={(name) => void navigate({ to: "/playlists/$name", params: { name } })} />
      </Stack>
    </Page>
  );
}

export function PlaylistScreen() {
  const { name } = useParams({ strict: false }) as { name?: string };
  const navigate = useNavigate();
  const player = usePlayer();

  // The playlist is the queue, in its own order: "Play all" then Next walks it. A track with no stream URL cannot be
  // played, and dropping it silently would renumber everything after it — so it is kept out of the queue and the
  // playlist itself stays exactly as the server has it.
  const play = (playlist: Playlist, startIndex: number) => {
    const playable = playlist.tracks
      .filter((track) => track.streamUrl !== undefined)
      .map(
        (track): NowPlayingTrack => ({
          id: track.itemId,
          libraryId: track.libraryId,
          title: track.title,
          subtitle: track.artist !== "" ? track.artist : track.album,
          coverUrl: track.coverUrl,
          streamUrl: track.streamUrl as string,
        }),
      );
    if (playable.length === 0) return;
    const start = playlist.tracks
      .slice(0, startIndex)
      .filter((track) => track.streamUrl !== undefined).length;
    player.playQueue(playable, Math.min(start, playable.length - 1));
  };

  if (name === undefined) return null;
  return (
    <Page>
      <Stack gap={16}>
        <PlaylistDetail name={name} onBack={() => void navigate({ to: "/playlists" })} onPlay={play} />
      </Stack>
    </Page>
  );
}
