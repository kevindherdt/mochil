// MiniPlayerBar — the shell's docked playback bar.
//
// Two layouts out of one component, because the same bar is a different thing on a phone and on a desktop. On a phone
// it is the mock-up's strip — square cover, two lines, a hairline of progress, play and close — pinned above the dock,
// and every pixel of height is chrome the screen has already lost. On a desktop there is room for what a player bar is
// actually for: where you are in the track (a real slider, real times), moving between queued tracks, and volume
// setters. Kevin, 24 Nov: "the music bar looks very awkward on web desktop especially, compared to jellyfin".
//
// Presentational only (law L2): state and navigation live in features/player/MiniPlayer.tsx, which is why this file
// takes numbers and callbacks and never reads the player context. Controls that have nothing behind them are not
// rendered — the queue buttons only appear when there is a queue, because a skip button that cannot skip is a lie.
import { useRef } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { Pause, Play, SkipBack, SkipForward, Volume2, X } from "lucide-react";
import styles from "./MiniPlayerBar.module.css";

const SWIPE_UP_THRESHOLD = 32;

/** m:ss, or h:mm:ss past an hour — how every player writes a track position. */
function clock(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
  const whole = Math.floor(seconds);
  const hours = Math.floor(whole / 3600);
  const minutes = Math.floor((whole % 3600) / 60);
  const rest = whole % 60;
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(rest).padStart(2, "0")}`
    : `${minutes}:${String(rest).padStart(2, "0")}`;
}

export interface MiniPlayerBarProps {
  title: string;
  subtitle: string;
  coverUrl: string;
  /** 0–1. */
  fraction: number;
  isPlaying: boolean;
  /** Seconds. Only the desktop layout shows them. */
  position?: number | undefined;
  duration?: number | undefined;
  hasNext?: boolean | undefined;
  hasPrevious?: boolean | undefined;
  volume?: number | undefined;
  /** A desktop bar has room for the transport; a phone strip does not. */
  expanded?: boolean;
  onOpen: () => void;
  onTogglePlay: () => void;
  onStop: () => void;
  onSeek?: ((seconds: number) => void) | undefined;
  onNext?: (() => void) | undefined;
  onPrevious?: (() => void) | undefined;
  onVolume?: ((value: number) => void) | undefined;
}

export function MiniPlayerBar({
  title,
  subtitle,
  coverUrl,
  fraction,
  isPlaying,
  position = 0,
  duration = 0,
  hasNext = false,
  hasPrevious = false,
  volume = 1,
  expanded = false,
  onOpen,
  onTogglePlay,
  onStop,
  onSeek,
  onNext,
  onPrevious,
  onVolume,
}: MiniPlayerBarProps) {
  const startY = useRef<number | undefined>(undefined);

  const handlePointerDown = (event: ReactPointerEvent<HTMLButtonElement>) => {
    startY.current = event.clientY;
  };
  const handlePointerUp = (event: ReactPointerEvent<HTMLButtonElement>) => {
    const start = startY.current;
    startY.current = undefined;
    if (start !== undefined && start - event.clientY > SWIPE_UP_THRESHOLD) onOpen();
  };

  const identity = (
    <>
      <img className={expanded ? styles.coverWide : styles.cover} src={coverUrl} alt="" />
      <span className={styles.text}>
        <span className={styles.title}>{title}</span>
        <span className={styles.subtitle}>{subtitle}</span>
      </span>
    </>
  );

  if (expanded) {
    return (
      <div className={`${styles.bar} ${styles.wide}`}>
        <button type="button" className={styles.open} onClick={onOpen} title="Open the player">
          {identity}
        </button>

        <div className={styles.transport}>
          <div className={styles.buttons}>
            <button
              type="button"
              className={styles.step}
              aria-label="Previous track"
              onClick={onPrevious}
              disabled={onPrevious === undefined || !hasPrevious}
            >
              <SkipBack size={16} />
            </button>
            <button type="button" className={styles.play} aria-label={isPlaying ? "Pause" : "Play"} onClick={onTogglePlay}>
              {isPlaying ? <Pause size={18} /> : <Play size={18} />}
            </button>
            <button
              type="button"
              className={styles.step}
              aria-label="Next track"
              onClick={onNext}
              disabled={onNext === undefined || !hasNext}
            >
              <SkipForward size={16} />
            </button>
          </div>

          <div className={styles.timeline}>
            <span className={styles.time}>{clock(position)}</span>
            <input
              type="range"
              className={styles.slider}
              aria-label="Position in the track"
              min={0}
              max={duration > 0 ? Math.floor(duration) : 0}
              step={1}
              value={Math.min(Math.floor(position), duration > 0 ? Math.floor(duration) : 0)}
              onChange={(event) => onSeek?.(Number(event.target.value))}
              disabled={duration <= 0}
            />
            <span className={styles.time}>{clock(duration)}</span>
          </div>
        </div>

        <div className={styles.right}>
          <Volume2 size={16} className={styles.volumeIcon} aria-hidden="true" />
          <input
            type="range"
            className={`${styles.slider} ${styles.volume}`}
            aria-label="Volume"
            min={0}
            max={100}
            step={1}
            value={Math.round(volume * 100)}
            onChange={(event) => onVolume?.(Number(event.target.value) / 100)}
          />
          <button type="button" className={styles.step} aria-label="Stop and close the player" onClick={onStop}>
            <X size={16} />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.bar}>
      <button
        type="button"
        className={styles.open}
        onClick={onOpen}
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
      >
        {identity}
        <span className={styles.progress}>
          <span className={styles.progressFill} style={{ width: `${Math.round(fraction * 100)}%` }} />
        </span>
      </button>
      <button type="button" className={styles.control} aria-label={isPlaying ? "Pause" : "Play"} onClick={onTogglePlay}>
        {isPlaying ? <Pause size={18} /> : <Play size={18} />}
      </button>
      <button type="button" className={styles.control} aria-label="Stop" onClick={onStop}>
        <X size={18} />
      </button>
    </div>
  );
}
