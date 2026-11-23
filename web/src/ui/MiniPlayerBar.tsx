// MiniPlayerBar — the shell's docked playback bar (mock-up "small music player look"): square cover, title/artist,
// a thin progress line, play/pause and stop. Presentational only (law L2 — the feature component that knows
// about the player's state lives in features/player/MiniPlayer.tsx and wires this). Kept deliberately short —
// Kevin, 22 Nov: "i didnt ask u to make mini player bigger vertically" — it only needs its icons sized right, not
// an extra row of text. The mock-up's skip-next and waveform/visualizer glyphs are left out rather than faked:
// there is no queue to skip through and nothing here actually analyses the audio.
//
// Opens the big player either by a tap or by swiping the bar upward (the gesture every music app answers "slide
// to expand" with) — Kevin: "make it possible to slide from mini to big player". A swipe is a plain vertical
// drag past a small threshold, not a physics simulation: the bar doesn't visually follow the finger on the way
// up (nothing to follow to — "big" isn't a transformed version of "mini"), it only decides whether the gesture
// was a swipe-up or an ordinary tap.
import { useRef } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { Pause, Play, X } from "lucide-react";
import styles from "./MiniPlayerBar.module.css";

const SWIPE_UP_THRESHOLD = 32;

export interface MiniPlayerBarProps {
  title: string;
  subtitle: string;
  coverUrl: string;
  /** 0–1. */
  fraction: number;
  isPlaying: boolean;
  onOpen: () => void;
  onTogglePlay: () => void;
  onStop: () => void;
}

export function MiniPlayerBar({ title, subtitle, coverUrl, fraction, isPlaying, onOpen, onTogglePlay, onStop }: MiniPlayerBarProps) {
  const startY = useRef<number | undefined>(undefined);

  const handlePointerDown = (event: ReactPointerEvent<HTMLButtonElement>) => {
    startY.current = event.clientY;
  };
  const handlePointerUp = (event: ReactPointerEvent<HTMLButtonElement>) => {
    const start = startY.current;
    startY.current = undefined;
    if (start !== undefined && start - event.clientY > SWIPE_UP_THRESHOLD) onOpen();
  };

  return (
    <div className={styles.bar}>
      <button
        type="button"
        className={styles.open}
        onClick={onOpen}
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
      >
        <img className={styles.cover} src={coverUrl} alt="" />
        <span className={styles.text}>
          <span className={styles.title}>{title}</span>
          <span className={styles.subtitle}>{subtitle}</span>
          <span className={styles.progress}>
            <span className={styles.progressFill} style={{ width: `${Math.round(fraction * 100)}%` }} />
          </span>
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
