// MediaGrid — the single card-grid rule (plan §4.3). Responds to its own container width via container queries
// (law L7), not the viewport, so the same grid works inside a phone sheet or a wide desktop panel.
import type { ReactNode } from "react";
import styles from "./MediaGrid.module.css";

export function MediaGrid({ children }: { children: ReactNode }) {
  return (
    <div className={styles.grid}>
      <div className={styles.inner}>{children}</div>
    </div>
  );
}
