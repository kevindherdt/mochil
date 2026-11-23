// VideoCard — shared by Movies and TV (shelf-card parity). No pill (unlike BookCard): a format/status strip on
// the artwork (e.g. "4K HDR · Direct Play") and one muted metadata line below the title does the same job.
import { Bookmark } from "lucide-react";
import { CoverImage } from "./CoverImage";
import styles from "./VideoCard.module.css";

export interface VideoCardProps {
  title: string;
  /** e.g. "Movie · Sci-Fi" or "TV Series · 1 Season". */
  meta: string;
  coverUrl: string;
  /** e.g. "4K HDR" or "4K HDR · Direct Play" — read from the real item, never invented. */
  formatLabel?: string;
  progress?: number;
  saved?: boolean;
  onToggleSave?: () => void;
  onClick?: () => void;
}

export function VideoCard({ title, meta, coverUrl, formatLabel, progress, saved, onToggleSave, onClick }: VideoCardProps) {
  return (
    <button type="button" className={styles.card} onClick={onClick}>
      <CoverImage
        src={coverUrl}
        alt=""
        ratio="2x3"
        radius="8"
        formatLabel={formatLabel}
        progress={progress}
        topRight={
          onToggleSave && (
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
          )
        }
      />
      <div className={styles.textBlock}>
        <p className={styles.title}>{title}</p>
        <p className={styles.meta}>{meta}</p>
      </div>
    </button>
  );
}
