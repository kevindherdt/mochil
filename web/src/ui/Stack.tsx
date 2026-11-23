// Stack — vertical flex layout with a token-scale gap, align and justify (plan §4.3).
import type { CSSProperties, ReactNode } from "react";
import { type SpaceKey } from "../lib/tokenMeta";
import styles from "./Stack.module.css";

export interface StackProps {
  children: ReactNode;
  /** Gap between children, from the space scale (plan §4.1: 4/8/12/16/24/32/48 — nothing else exists). */
  gap?: SpaceKey;
  align?: CSSProperties["alignItems"];
  justify?: CSSProperties["justifyContent"];
}

export function Stack({ children, gap = 16, align, justify }: StackProps) {
  const style: CSSProperties = { gap: `var(--space-${gap})` };
  if (align !== undefined) style.alignItems = align;
  if (justify !== undefined) style.justifyContent = justify;
  return (
    <div className={styles.stack} style={style}>
      {children}
    </div>
  );
}
