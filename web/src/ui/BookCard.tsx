// BookCard — shared by books, audiobooks, light novels and Storyteller (shelf-card parity). Cover (2:3, radius
// 12) → title → author. No tag pill: Kevin's ruling (22 Nov 2026) is that a card's *kind* is an icon on the
// artwork, not a line of text under the author — "just a book icon on the cover to reduce text clutter". That
// supersedes ui-overhaul-plan.md's "cover → title → author → one pill" for this card family; the mock-ups'
// `Narrated / Whispersync` / `Light Novel` pills are gone.
// Reading state stays per Kevin's ruling: a thin accent bar while in progress, a small green check when finished.
import type { ComponentType } from "react";
import { Bookmark, Headphones } from "lucide-react";
import { CoverImage } from "./CoverImage";
import coverStyles from "./CoverImage.module.css";
import styles from "./BookCard.module.css";

export interface BookCardProps {
  title: string;
  author: string;
  coverUrl: string;
  /** The kind glyph on the artwork, e.g. BookOpen / Headphones / BookImage. */
  kindIcon?: ComponentType<{ size?: number }> | undefined;
  /** Series position, rendered as a small "Vol. N" plate (top-right, alongside the bookmark). */
  seriesIndex?: number | undefined;
  /** 0–1 reading progress. Omit (or pass `finished`) when there's nothing to show. */
  progress?: number | undefined;
  finished?: boolean | undefined;
  /** A short label printed on the artwork: "1080p" for a film, "FLAC" for a track. It replaces the progress
      strip while it's set, the same rule CoverImage already applies for MusicCard. */
  formatLabel?: string | undefined;
  /** Audio edition exists (narrated/Whispersync) — shown as a small headphones badge, top-right. */
  hasAudio?: boolean;
  saved?: boolean;
  onToggleSave?: () => void;
  onClick?: () => void;
}

export function BookCard({
  title,
  author,
  coverUrl,
  kindIcon,
  seriesIndex,
  progress,
  finished,
  formatLabel,
  hasAudio,
  saved,
  onToggleSave,
  onClick,
}: BookCardProps) {
  return (
    <button type="button" className={styles.card} onClick={onClick}>
      <CoverImage
        src={coverUrl}
        alt=""
        ratio="2x3"
        radius="8"
        kindIcon={kindIcon}
        progress={finished ? undefined : progress}
        finished={finished}
        formatLabel={formatLabel}
        topRight={
          <>
            {seriesIndex !== undefined && <span className={coverStyles.badge}>Vol. {seriesIndex}</span>}
            {hasAudio && <Headphones size={13} color="#fff" />}
            {onToggleSave && (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation();
                  onToggleSave();
                }}
                aria-label={saved ? "Remove bookmark" : "Save"}
                aria-pressed={saved}
                style={{ background: "none", border: "none", padding: 0, cursor: "pointer", color: "#fff", display: "flex" }}
              >
                <Bookmark size={15} fill={saved ? "currentColor" : "none"} />
              </button>
            )}
          </>
        }
      />
      <div className={styles.textBlock}>
        <p className={styles.title}>{title}</p>
        <p className={styles.author}>{author}</p>
      </div>
    </button>
  );
}
