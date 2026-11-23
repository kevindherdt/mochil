// MediaList — the list counterpart of MediaGrid. Items are plain data (title/subtitle/cover/progress) rather than
// a card component, because a list row is not a card: if it grew its own per-family variants the two layouts would
// drift. Screens pass the same fields they pass to a card.
import type { ComponentType } from "react";
import { Check } from "lucide-react";
import styles from "./MediaList.module.css";

export interface MediaListItem {
  id: string;
  title: string;
  subtitle?: string | undefined;
  coverUrl: string;
  /** 0–1. Drawn as the thin accent bar under the subtitle — never as a percentage label. */
  progress?: number | undefined;
  finished?: boolean | undefined;
  onClick?: () => void;
}

export interface MediaListProps {
  items: MediaListItem[];
  /** Unused visually for now, kept so a caller can pass the shelf's kind glyph when a row grows one. */
  kindIcon?: ComponentType<{ size?: number }> | undefined;
}

export function MediaList({ items }: MediaListProps) {
  return (
    <div className={styles.list} role="list">
      {items.map((item) => (
        <button key={item.id} type="button" role="listitem" className={styles.row} onClick={item.onClick}>
          <span className={styles.thumb}>
            <img className={styles.thumbImage} src={item.coverUrl} alt="" loading="lazy" />
          </span>
          <span className={styles.text}>
            <p className={styles.title}>{item.title}</p>
            {item.subtitle && <p className={styles.subtitle}>{item.subtitle}</p>}
            {item.progress !== undefined && !item.finished && (
              <span className={styles.bar}>
                <span className={styles.barFill} style={{ width: `${Math.round(item.progress * 100)}%` }} />
              </span>
            )}
          </span>
          {item.finished && (
            <span className={styles.done} aria-label="Finished">
              <Check size={16} strokeWidth={3} />
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
