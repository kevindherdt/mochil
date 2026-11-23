// PlaybackControls — the transport row the mock-up's big player shows: -15 / big play-pause circle / +15, with
// elapsed/total time above a progress line. Display-only progress (no drag-to-seek yet — AudioDetailScreen's
// header documents that gap); -15/+15 are real seeks, wired to the one global <audio> element.
import { Pause, Play, RotateCcw, RotateCw } from "lucide-react";
import { formatTime } from "./formatTime";
import styles from "./PlaybackControls.module.css";


export interface PlaybackControlsProps {
  isPlaying: boolean;
  position: number;
  duration: number;
  onTogglePlay: () => void;
  onSkip: (deltaSeconds: number) => void;
}

export function PlaybackControls({ isPlaying, position, duration, onTogglePlay, onSkip }: PlaybackControlsProps) {
  const fraction = duration > 0 ? position / duration : 0;
  return (
    <div className={styles.controls}>
      <div className={styles.timeline}>
        <span className={styles.bar}>
          <span className={styles.barFill} style={{ width: `${Math.round(fraction * 100)}%` }} />
        </span>
        <span className={styles.times}>
          <span>{formatTime(position)}</span>
          <span>{formatTime(duration)}</span>
        </span>
      </div>
      <div className={styles.row}>
        <button type="button" className={styles.skip} aria-label="Back 15 seconds" onClick={() => onSkip(-15)}>
          <RotateCcw size={20} />
        </button>
        <button type="button" className={styles.play} aria-label={isPlaying ? "Pause" : "Play"} onClick={onTogglePlay}>
          {isPlaying ? <Pause size={28} /> : <Play size={28} />}
        </button>
        <button type="button" className={styles.skip} aria-label="Forward 15 seconds" onClick={() => onSkip(15)}>
          <RotateCw size={20} />
        </button>
      </div>
    </div>
  );
}
