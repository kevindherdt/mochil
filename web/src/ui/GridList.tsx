// GridList — a vertical list of cards that becomes two or three columns when its container is wide enough.
// Added for the Settings hub (Phase 3's "two-column settings"): eleven panel cards in one 928 px strip inside a
// 1440 px window read as a phone screen stretched sideways. The column rules live in the CSS module beside this
// file (law L3 keeps raw column counts and media/container queries out of feature code).
import type { ReactNode } from "react";
import styles from "./GridList.module.css";

export function GridList({ children }: { children: ReactNode }) {
  return (
    <div className={styles.grid}>
      <div className={styles.inner}>{children}</div>
    </div>
  );
}
