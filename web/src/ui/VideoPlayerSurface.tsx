// ui/VideoPlayerSurface — the drawing of a video player: the video element's frame, the scrim bars, the scrubber, the
// transport row and the centre play button. Purely presentational (law L2): it owns no source, no progress and no
// state, and every value it shows and every action it offers arrives as a prop.
//
// The feature that plays the film (features/library/VideoPlayback.tsx) passes the <video> element in as a child, so the
// element's ref and its <track> children stay with the code that manages them while the chrome lives here in ui/ where
// screens and features are allowed to get their layout from.
//
// Colours are declared as local custom properties rather than raw values on a background: a player's ground is black
// and its chrome is white over a scrim whatever palette the app uses (the same reasoning that allows #fff over cover
// art), and law L3 wants each such value in one named place.
import type { ReactNode } from "react";
import { ArrowLeft, Maximize, Minimize, Pause, Play, RotateCcw, RotateCw, Settings2 } from "lucide-react";
import styles from "./VideoPlayerSurface.module.css";

export interface VideoPlayerSurfaceProps {
  /** The <video> element, with its <track> children. Rendered full-bleed under the chrome. */
  video: ReactNode;
  title: string;
  ready: boolean;
  playing: boolean;
  controlsVisible: boolean;
  fullscreen: boolean;
  /** Seconds. */
  position: number;
  /** Seconds. Zero until the duration is known. */
  duration: number;
  /** How much of the film is buffered, as a percentage — drawn behind the played portion. */
  bufferedPercent: number;
  /** Audio and subtitle menus, built by the feature (they open popovers, which is the feature's business). */
  menus?: ReactNode;
  /** Cue-appearance custom properties (`--cue-*`), from lib/subtitlePreferences. Applied to the wrapper so the ::cue
   *  rule below picks them up; changing a setting restyles the captions without touching the track elements. */
  cueVars?: Record<string, string>;
  onBack: () => void;
  onTogglePlay: () => void;
  onSeekBy: (seconds: number) => void;
  onSeekTo: (seconds: number) => void;
  onSeekStart: () => void;
  onSeekEnd: () => void;
  onToggleFullscreen: () => void;
  /** Opens the playback settings sheet (subtitle appearance, speed). */
  onOpenSettings: () => void;
  /** Rendered last, inside the player box — the settings sheet positions itself against the picture, not the document. */
  overlay?: ReactNode;
  /** The caption block, drawn by us rather than by the browser (see ui/CaptionOverlay). */
  captions?: ReactNode;
  /** Called on any pointer movement; the feature decides whether that reveals the controls. */
  onWake: () => void;
}

/** "1:04:09" — hours only when there are any, so a sitcom does not read as "0:21:30". */
function formatClock(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
  const total = Math.floor(seconds);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (value: number) => String(value).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}

export function VideoPlayerSurface({
  video,
  title,
  ready,
  playing,
  controlsVisible,
  fullscreen,
  position,
  duration,
  bufferedPercent,
  menus,
  cueVars,
  onBack,
  onTogglePlay,
  onSeekBy,
  onSeekTo,
  onSeekStart,
  onSeekEnd,
  onToggleFullscreen,
  onOpenSettings,
  overlay,
  captions,
  onWake,
}: VideoPlayerSurfaceProps) {
  const playedPercent = duration > 0 ? (position / duration) * 100 : 0;

  return (
    <div
      className={controlsVisible ? styles.player : `${styles.player} ${styles.idle}`}
      style={cueVars}
      onPointerMove={onWake}
      onPointerDown={onWake}
    >
      {video}

      {!ready && <div className={styles.spinner}>Opening…</div>}

      {!playing && ready && (
        <button type="button" className={styles.centrePlay} onClick={onTogglePlay} aria-label="Play">
          <Play size={40} fill="currentColor" />
        </button>
      )}

      {captions}

      <div className={styles.topBar}>
        <button type="button" className={styles.iconButton} onClick={onBack} aria-label="Back">
          <ArrowLeft size={20} />
        </button>
        <div className={styles.titleBlock}>
          <span className={styles.title}>{title}</span>
        </div>
        <div className={styles.topActions}>
          {menus}
          <button type="button" className={styles.iconButton} onClick={onOpenSettings} aria-label="Playback settings">
            <Settings2 size={20} />
          </button>
        </div>
      </div>

      <div className={styles.bottomBar}>
        <div className={styles.scrubber}>
          <div className={styles.track}>
            <div className={styles.buffered} style={{ width: `${bufferedPercent}%` }} />
            <div className={styles.played} style={{ width: `${playedPercent}%` }} />
            <input
              className={styles.range}
              type="range"
              min={0}
              max={duration > 0 ? duration : 0}
              step={0.1}
              value={position}
              aria-label="Seek"
              onChange={(event) => {
                onSeekStart();
                onSeekTo(Number(event.target.value));
              }}
              onPointerUp={onSeekEnd}
              onKeyUp={onSeekEnd}
              onBlur={onSeekEnd}
            />
          </div>
          <div className={styles.times}>
            <span>{formatClock(position)}</span>
            <span>{formatClock(duration)}</span>
          </div>
        </div>

        {/* Three columns, not one centred row: the transport stays optically centred whatever sits at the right,
            and fullscreen stops being a fourth glyph glued to "forward 15". */}
        <div className={styles.transportRow}>
          <div />
          <div className={styles.transport}>
            <button type="button" className={styles.iconButton} onClick={() => onSeekBy(-15)} aria-label="Back 15 seconds">
              <RotateCcw size={22} />
            </button>
            <button
              type="button"
              className={styles.playButton}
              onClick={onTogglePlay}
              aria-label={playing ? "Pause" : "Play"}
            >
              {playing ? <Pause size={24} fill="currentColor" /> : <Play size={24} fill="currentColor" />}
            </button>
            <button type="button" className={styles.iconButton} onClick={() => onSeekBy(15)} aria-label="Forward 15 seconds">
              <RotateCw size={22} />
            </button>
          </div>
          <div className={styles.transportEnd}>
            <button type="button" className={styles.iconButton} onClick={onToggleFullscreen} aria-label="Fullscreen">
              {fullscreen ? <Minimize size={20} /> : <Maximize size={20} />}
            </button>
          </div>
        </div>
      </div>

      {overlay}
    </div>
  );
}
