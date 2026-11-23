// MusicCard — deliberately different from Book/VideoCard (memory: "Music deliberately different, artwork
// scarcity"): square artwork, no bookmark/progress chrome, an optional FLAC/format strip replaces the old
// inline FLAC pill (plan B2: "Drop the FLAC chip; album/track cards get a FLAC strip on the artwork").
// No tag pill either — same ruling as BookCard (Kevin, 22 Nov 2026): a card's kind is a glyph on the artwork,
// not another line of text under the artist.
import type { ComponentType } from "react";
import { CoverImage } from "./CoverImage";
import styles from "./MusicCard.module.css";

export interface MusicCardProps {
  title: string;
  artist: string;
  coverUrl: string;
  formatLabel?: string | undefined;
  /** The kind glyph on the artwork (Music2 for an album, ListMusic for a playlist). */
  kindIcon?: ComponentType<{ size?: number }> | undefined;
  onClick?: () => void;
}

export function MusicCard({ title, artist, coverUrl, formatLabel, kindIcon, onClick }: MusicCardProps) {
  return (
    <button type="button" className={styles.card} onClick={onClick}>
      <CoverImage src={coverUrl} alt="" ratio="1x1" radius="8" formatLabel={formatLabel} kindIcon={kindIcon} />
      <div className={styles.textBlock}>
        <p className={styles.title}>{title}</p>
        <p className={styles.artist}>{artist}</p>
      </div>
    </button>
  );
}
