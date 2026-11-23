// features/player/MiniPlayer.tsx — wires the global player into the shell's `player` slot (plan §5.6: shell
// slots, not per-screen state). Presentation lives in ui/MiniPlayerBar.tsx (law L2: a feature component may not
// own a CSS module); this file only knows what state to read and where "open" navigates to.
import { useNavigate, useParams } from "@tanstack/react-router";
import { MiniPlayerBar } from "../../ui";
import { usePlayer } from "./PlayerContext";

export function MiniPlayer() {
  const player = usePlayer();
  const navigate = useNavigate();
  const routeParams = useParams({ strict: false }) as { libraryId?: string; itemId?: string };

  if (player.track === undefined) return null;
  const { track } = player;

  // Hidden on the track's own Now Playing screen: that screen already shows everything this bar would, and two
  // transports on one screen at once is exactly the confusion Kevin flagged (22 Nov: "clicking on the music
  // player the small one and big one shouldnt be visible at the same time ofcourse").
  const onOwnNowPlayingScreen = routeParams.libraryId === track.libraryId && routeParams.itemId === track.id;
  if (onOwnNowPlayingScreen) return null;

  return (
    <MiniPlayerBar
      title={track.title}
      subtitle={track.subtitle}
      coverUrl={track.coverUrl}
      fraction={player.duration > 0 ? player.position / player.duration : 0}
      isPlaying={player.isPlaying}
      onOpen={() => void navigate({ to: "/library/audio/$libraryId/$itemId", params: { libraryId: track.libraryId, itemId: track.id } })}
      onTogglePlay={player.togglePlay}
      onStop={player.stop}
    />
  );
}
