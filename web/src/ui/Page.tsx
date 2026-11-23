// Page — the outer padding/width constraint every screen's content sits inside (plan §4.3).
import type { ReactNode } from "react";
import styles from "./Page.module.css";

export function Page({ children, wide, fill }: { children: ReactNode; wide?: boolean; fill?: boolean }) {
  const classes = [styles.page, wide ? styles.wide : "", fill ? styles.fill : ""].filter(Boolean).join(" ");
  return <div className={classes}>{children}</div>;
}
