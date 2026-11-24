// features/player/PlayerContext.tsx — the one `<audio>` element in the whole app (plan §5.6: shell slots, not
// per-screen state). It is mounted once, above the router, so playback survives navigating away from the track
// that started it — the mini player (MiniPlayer.tsx) is the shell's `player` slot, and AudioDetailScreen is the
// "Now Playing" screen the mockup calls the big player; both read this context instead of owning an element.
//
// Scope, honestly: this plays what `/api/audio` already streams (music/podcasts/audiobooks — `stream_url` is
// unconditional there, unlike video's DirectPlay gating). There is still no scrubbing by drag (the progress bar is
// display-only), and no server-side resume position for audio (the book reader's `/api/reading` position tracking
// has no audio equivalent yet). Those are real gaps, not silently skipped — see docs/rewrite-progress.md.
//
// **Queue**: a track list handed in by an album/series/episode-list can now advance by itself. It lives here rather
// than in a screen because playback outlives the screen that started it — the whole point of one element above the
// router. Deliberately client-side: the server's `/api/audio-queue` store only accepts *music* libraries
// (internal/web/audio_collection_http.go rejects anything else with 400), so syncing a podcast or audiobook queue
// through it is not possible today, and a queue that worked for only one kind would be worse than one that works
// for all of them in this browser.
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";

export interface NowPlayingTrack {
  id: string;
  libraryId: string;
  title: string;
  subtitle: string;
  coverUrl: string;
  formatLabel?: string | undefined;
  streamUrl: string;
}

export interface PlayerState {
  track: NowPlayingTrack | undefined;
  isPlaying: boolean;
  /** Seconds. */
  position: number;
  duration: number;
  play: (track: NowPlayingTrack) => void;
  /** Replaces the queue and starts at `startIndex` (an album's "Play all", a series' first book). */
  playQueue: (tracks: NowPlayingTrack[], startIndex?: number) => void;
  /** Appends one track to the queue without changing what is playing. */
  enqueue: (track: NowPlayingTrack) => void;
  /** The tracks after the current one, for a queue view. */
  queue: NowPlayingTrack[];
  togglePlay: () => void;
  /** Relative seek, clamped to [0, duration] — the mock-up's -15/+15 buttons. */
  skip: (deltaSeconds: number) => void;
  /** Absolute seek, clamped the same way. An audiobook chapter is a position, not an offset from here. */
  seekTo: (seconds: number) => void;
  /** Move within the queue. Both are no-ops without a queue, and both are honest about it through hasNext/hasPrevious. */
  next: () => void;
  previous: () => void;
  hasNext: boolean;
  hasPrevious: boolean;
  /** 0–1. The element's own volume — there is no server-side volume, and none is needed. */
  volume: number;
  setVolume: (value: number) => void;
  stop: () => void;
}

const PlayerContext = createContext<PlayerState | undefined>(undefined);

export function PlayerProvider({ children }: { children: ReactNode }) {
  const audioRef = useRef<HTMLAudioElement>(null);
  // Three pieces of state instead of a track + a queue + a "which one is current" ref: a queue *and* a cursor, plus
  // the single track that was played on its own. `track` is derived, so "advance" is just moving the cursor and
  // there is no second copy of the truth to keep in sync.
  const [queue, setQueue] = useState<NowPlayingTrack[]>([]);
  const [cursor, setCursor] = useState(-1);
  const [standalone, setStandalone] = useState<NowPlayingTrack | undefined>(undefined);
  const track = standalone ?? queue[cursor];
  const [isPlaying, setIsPlaying] = useState(false);
  const [position, setPosition] = useState(0);
  const [duration, setDuration] = useState(0);

  const play = useCallback((next: NowPlayingTrack) => {
    // Same track again is a no-op: togglePlay handles pause/resume, and re-setting it would restart playback.
    setStandalone((current) => (current?.id === next.id ? current : next));
    setQueue([]);
    setCursor(-1);
    setPosition(0);
    setDuration(0);
    setIsPlaying(true);
  }, []);

  const playQueue = useCallback((tracks: NowPlayingTrack[], startIndex = 0) => {
    if (tracks[startIndex] === undefined) return;
    setStandalone(undefined);
    setQueue(tracks);
    setCursor(startIndex);
    setPosition(0);
    setDuration(0);
    setIsPlaying(true);
  }, []);

  const enqueue = useCallback((next: NowPlayingTrack) => {
    setQueue((current) => {
      if (current.some((item) => item.id === next.id)) return current;
      return [...current, next];
    });
  }, []);

  /** What `onEnded` does: move to the next queued track, or stop. Nothing wraps around — a finished album stops. */
  const advance = useCallback(() => {
    setCursor((current) => {
      if (current === -1 || current + 1 >= queue.length) {
        setIsPlaying(false);
        return current;
      }
      setPosition(0);
      setDuration(0);
      setIsPlaying(true);
      return current + 1;
    });
  }, [queue.length]);

  const togglePlay = useCallback(() => setIsPlaying((was) => !was), []);

  const next = useCallback(() => {
    setCursor((current) => {
      if (current === -1 || current + 1 >= queue.length) return current;
      setPosition(0);
      setDuration(0);
      setIsPlaying(true);
      return current + 1;
    });
  }, [queue.length]);

  const previous = useCallback(() => {
    setCursor((current) => {
      if (current <= 0) return current;
      setPosition(0);
      setDuration(0);
      setIsPlaying(true);
      return current - 1;
    });
  }, []);

  const [volume, setVolumeState] = useState(1);
  const setVolume = useCallback((value: number) => {
    const clamped = Math.max(0, Math.min(1, value));
    setVolumeState(clamped);
    if (audioRef.current !== null) audioRef.current.volume = clamped;
  }, []);

  const skip = useCallback((deltaSeconds: number) => {
    const element = audioRef.current;
    if (element === null) return;
    element.currentTime = Math.max(0, Math.min(element.duration || 0, element.currentTime + deltaSeconds));
  }, []);

  const seekTo = useCallback((seconds: number) => {
    const element = audioRef.current;
    if (element === null) return;
    element.currentTime = Math.max(0, Math.min(element.duration || seconds, seconds));
  }, []);

  const stop = useCallback(() => {
    setStandalone(undefined);
    setQueue([]);
    setCursor(-1);
    setIsPlaying(false);
    setPosition(0);
    setDuration(0);
  }, []);

  // The element is the source of truth for play state, not the other way: a user hitting a system media key or
  // the element simply finishing playback has to be reflected back into isPlaying, not only requests we made.
  useEffect(() => {
    const element = audioRef.current;
    if (element === null || track === undefined) return;
    if (isPlaying) void element.play().catch(() => setIsPlaying(false));
    else element.pause();
  }, [isPlaying, track]);

  const hasNext = cursor !== -1 && cursor + 1 < queue.length;
  const hasPrevious = cursor > 0;

  const value = useMemo<PlayerState>(
    () => ({
      track,
      isPlaying,
      position,
      duration,
      play,
      playQueue,
      enqueue,
      queue,
      togglePlay,
      skip,
      seekTo,
      next,
      previous,
      hasNext,
      hasPrevious,
      volume,
      setVolume,
      stop,
    }),
    [
      track,
      isPlaying,
      position,
      duration,
      play,
      playQueue,
      enqueue,
      queue,
      togglePlay,
      skip,
      seekTo,
      next,
      previous,
      hasNext,
      hasPrevious,
      volume,
      setVolume,
      stop,
    ],
  );

  return (
    <PlayerContext.Provider value={value}>
      {children}
      <audio
        ref={audioRef}
        src={track?.streamUrl}
        onTimeUpdate={(event) => setPosition(event.currentTarget.currentTime)}
        onLoadedMetadata={(event) => setDuration(event.currentTarget.duration)}
        onEnded={advance}
        style={{ display: "none" }}
      />
    </PlayerContext.Provider>
  );
}

// react-refresh/only-export-components would rather this hook lived in its own file; co-locating a context's
// provider with its one reader is the more common React pattern (same call as ShellVariant.tsx) and not worth a
// second file for two lines.
// eslint-disable-next-line react-refresh/only-export-components
export function usePlayer(): PlayerState {
  const value = useContext(PlayerContext);
  if (value === undefined) throw new Error("usePlayer() used outside <PlayerProvider>");
  return value;
}
