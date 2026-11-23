// GroupCard — one entry in the Series or Authors browse list.
//
// A series is drawn as a stack: the front cover plus two offset sheets behind it, because a single book cover with
// "Ancient Egypt · 7 books" underneath reads as a duplicate book card (that is what Kevin saw). The count is then
// confirmation, not the only clue. Authors are a single cover — one person is not a pile.
import type { ComponentType } from "react";
import { CoverImage } from "./CoverImage";
import styles from "./GroupCard.module.css";

export interface GroupCardProps {
  name: string;
  count: number;
  coverUrl?: string | undefined;
  /** Glyph on the artwork — UserRound for an author. Series get the stack instead, so they pass nothing. */
  kindIcon?: ComponentType<{ size?: number }> | undefined;
  /** True when the group holds several books (a series), which draws the stacked covers. */
  stack?: boolean;
  /** 2:3 for a book series/author; 1:1 for an album/artist (square artwork, same as MusicCard). */
  ratio?: "2x3" | "1x1";
  /** Overrides the default "N books" line — an album says "N tracks", an artist says "N albums" or "N tracks". */
  countLabel?: string;
  onClick?: () => void;
}

export function GroupCard({ name, count, coverUrl, kindIcon, stack = false, ratio = "2x3", countLabel, onClick }: GroupCardProps) {
  return (
    <button type="button" className={styles.card} onClick={onClick}>
      {stack ? (
        <div className={styles.stack}>
          <span className={`${styles.sheet} ${styles.sheetBack}`} aria-hidden="true" />
          <span className={`${styles.sheet} ${styles.sheetMiddle}`} aria-hidden="true" />
          <div className={styles.front}>
            <CoverImage src={coverUrl ?? ""} alt="" ratio={ratio} radius="8" />
          </div>
        </div>
      ) : (
        <CoverImage src={coverUrl ?? ""} alt="" ratio={ratio} radius="8" kindIcon={kindIcon} />
      )}
      <div className={styles.textBlock}>
        <p className={styles.name}>{name}</p>
        <p className={styles.count}>{countLabel ?? (count === 1 ? "1 book" : `${count} books`)}</p>
      </div>
    </button>
  );
}
