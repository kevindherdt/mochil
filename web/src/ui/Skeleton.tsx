// Skeleton — a loading placeholder shaped like the real layout it stands in for (plan §4.3).
import styles from "./Skeleton.module.css";

export interface SkeletonProps {
  /** CSS length (e.g. "100%", "3rem") — describes the shape of the content being loaded, not a design token. */
  width?: string;
  height?: string;
}

export function Skeleton({ width = "100%", height = "1.2em" }: SkeletonProps) {
  return <div className={styles.skeleton} style={{ width, height }} aria-hidden="true" />;
}
