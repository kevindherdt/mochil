// Rail — a horizontally scrolling shelf. It sizes its children itself (fixed card width) so callers just map cards
// in; see Rail.module.css for why that lives here rather than in each screen.
import type { ReactNode } from "react";
import styles from "./Rail.module.css";

export interface RailProps {
  children: ReactNode;
  /** Accessible name for the row, since a rail usually sits under a visible heading that isn't part of it. */
  label: string;
}

export function Rail({ children, label }: RailProps) {
  return (
    <div className={styles.rail} role="list" aria-label={label}>
      {children}
    </div>
  );
}
