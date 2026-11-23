// ui/CaptionOverlay — subtitle lines drawn by us, in our own box, so a viewer's size/colour/background/edge choices
// are the ones that reach the screen and so the captions can sit above the transport row instead of under it (the
// browser's own cue layer puts them where the video element's bottom edge is, which on a portrait phone is exactly
// where our buttons are).
//
// Presentational only: the lines arrive as props from features/library/useActiveCues, and the four look-settings arrive
// as `--cue-*` custom properties on the wrapper (lib/subtitlePreferences) — which work here, unlike on `::cue`, because
// this is ordinary DOM rather than a user-agent shadow tree.
import type { CSSProperties } from "react";
import styles from "./CaptionOverlay.module.css";

export interface CaptionOverlayProps {
  lines: string[];
  /** Raised while the transport row is on screen, so captions never sit under a button. */
  lifted: boolean;
  style?: CSSProperties;
}

export function CaptionOverlay({ lines, lifted, style }: CaptionOverlayProps) {
  if (lines.length === 0) return null;
  return (
    <div className={lifted ? `${styles.captions} ${styles.lifted}` : styles.captions} style={style} aria-live="off">
      {lines.map((line, index) => (
        <span key={index} className={styles.line}>
          {line}
        </span>
      ))}
    </div>
  );
}
