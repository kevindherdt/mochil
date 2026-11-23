// Text/Heading — the type scale, applied (plan §4.3).
import type { ElementType, ReactNode } from "react";
import type { FontSizeKey } from "../lib/tokenMeta";
import styles from "./Text.module.css";

export interface TextProps {
  children: ReactNode;
  size?: FontSizeKey;
  muted?: boolean;
  tone?: "default" | "error";
  as?: ElementType;
}

export function Text({ children, size = "14", muted, tone = "default", as: Tag = "p" }: TextProps) {
  const classes = [styles.text];
  if (muted) classes.push(styles.muted);
  if (tone === "error") classes.push(styles.error);
  return (
    <Tag className={classes.join(" ")} style={{ fontSize: `var(--font-size-${size})` }}>
      {children}
    </Tag>
  );
}

export interface HeadingProps {
  children: ReactNode;
  size?: FontSizeKey;
  /** Which heading level to render — kept separate from `size` so visual scale and document outline don't have
      to match (a section can be visually small but still the correct h2 for screen readers). */
  level?: 1 | 2 | 3 | 4;
}

export function Heading({ children, size = "22", level = 2 }: HeadingProps) {
  const Tag = (`h${level}` as const) as ElementType;
  return (
    <Tag className={styles.heading} style={{ fontSize: `var(--font-size-${size})` }}>
      {children}
    </Tag>
  );
}
