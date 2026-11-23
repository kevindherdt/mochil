// Inline — horizontal flex layout with a token-scale gap, align, justify and wrap (plan §4.3).
import type { CSSProperties, ReactNode } from "react";
import type { SpaceKey } from "../lib/tokenMeta";
import styles from "./Inline.module.css";

export interface InlineProps {
  children: ReactNode;
  /** Gap between children, from the space scale (plan §4.1). */
  gap?: SpaceKey;
  align?: CSSProperties["alignItems"];
  justify?: CSSProperties["justifyContent"];
  wrap?: boolean;
}

export function Inline({ children, gap = 8, align = "center", justify, wrap }: InlineProps) {
  const style: CSSProperties = { gap: `var(--space-${gap})`, alignItems: align };
  if (justify !== undefined) style.justifyContent = justify;
  if (wrap) style.flexWrap = "wrap";
  return (
    <div className={styles.inline} style={style}>
      {children}
    </div>
  );
}
