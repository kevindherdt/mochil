// CoverImage — the artwork box every card family shares: 2:3 (books/video) or 1:1 (music), a top-right badge
// slot (bookmark, Vol. N, audio-available), a bottom scrim slot (reading/format strip + accent bar), and a
// finished-reading badge (plan docs/ui-overhaul-plan.md "Card anatomy" + Kevin's reading-state ruling: "if there
// is reading progress it shows the progress bar line in the card, also if the book is read a tiny badge with a
// green checkmark is enough and doesn't clutter the cards too much while keep it visible" — no percentage text,
// no second line of overlay copy, just the bar or the badge).
import { useState, type ComponentType, type ReactNode } from "react";
import { Check } from "lucide-react";
import styles from "./CoverImage.module.css";
import { Skeleton } from "./Skeleton";

export interface CoverImageProps {
  src: string;
  alt: string;
  ratio?: "2x3" | "1x1";
  topRight?: ReactNode | undefined;
  /** What kind of item this is (book / audiobook / manga / …), as a small glyph in the cover's top-left corner.
      Kevin's ruling: a card carries no tag pill under the title — an icon on the artwork says the same thing
      without adding a line of text to every card in a row. */
  kindIcon?: ComponentType<{ size?: number }> | undefined;
  /** 0–1. Renders a thin accent bar at the bottom of the artwork — nothing else (no "72% read" label). */
  progress?: number | undefined;
  /** Renders a small green check badge instead of a progress bar — mutually exclusive with `progress`. */
  finished?: boolean | undefined;
  /** A one-line format/status string on a bottom scrim, e.g. "FLAC" or "4K HDR · Direct Play". */
  formatLabel?: string | undefined;
  /** Corner step for the artwork. Cards that sit on their own surface use 8 (an inset poster inside a bordered
      card); a bare artwork tile uses 12 (the mockups round posters at 12 at the card's own edge). */
  radius?: "8" | "12";
}

export function CoverImage({ src, alt, ratio = "2x3", topRight, kindIcon: KindIcon, progress, finished, formatLabel, radius = "12" }: CoverImageProps) {
  const radiusClass = radius === "8" ? styles.radius8 : styles.radius12;
  // Images arrive over the network and can render top-to-bottom for a moment on a slow connection or a large
  // cover (Kevin, 22 Nov 2026: "I can literally sometimes see the cover loading like it's a static page line by
  // line"). A shimmering placeholder covers that instead of exposing the raw partial decode; onError also clears
  // it; a broken image must not pulse forever.
  const [loaded, setLoaded] = useState(false);
  return (
    <div className={`${styles.cover} ${radiusClass} ${ratio === "2x3" ? styles.ratio2x3 : styles.ratio1x1}`}>
      {!loaded && (
        <div className={styles.skeletonLayer}>
          <Skeleton width="100%" height="100%" />
        </div>
      )}
      <img
        className={`${styles.image} ${loaded ? styles.imageLoaded : styles.imageLoading}`}
        src={src}
        alt={alt}
        loading="lazy"
        onLoad={() => setLoaded(true)}
        onError={() => setLoaded(true)}
      />
      {KindIcon && (
        <span className={styles.kindIcon} aria-hidden="true">
          <KindIcon size={13} />
        </span>
      )}
      {topRight && <div className={styles.topRight}>{topRight}</div>}
      {formatLabel && (
        <div className={styles.scrim}>
          {formatLabel}
          {progress !== undefined && (
            <div className={styles.scrimBar}>
              <div className={styles.scrimBarFill} style={{ width: `${Math.round(progress * 100)}%` }} />
            </div>
          )}
        </div>
      )}
      {!formatLabel && progress !== undefined && (
        <div className={styles.scrim} style={{ padding: "var(--space-8)" }}>
          <div className={styles.scrimBar}>
            <div className={styles.scrimBarFill} style={{ width: `${Math.round(progress * 100)}%` }} />
          </div>
        </div>
      )}
      {finished && (
        <div className={styles.finishedBadge} aria-label="Finished">
          <Check size={13} strokeWidth={3} />
        </div>
      )}
    </div>
  );
}
