// Toolbar — one row of controls at one height (a search field, a few menus, a button or two).
//
// Kevin, on the editor: "the buttons/fields to type in and the tools button can be resized to be in line like the
// others in size". They were four sizes: a 44px touch-height search box and buttons, a 32px menu trigger, a 28px chip
// row. A toolbar that is one row on screen should be one row in the eye, so this sets them all to the trigger's 32px
// and leaves the rest to the controls' own styles.
//
// Only heights are imposed: a Button stays a Button, a Menu stays a Menu. Screens handing their controls a height
// themselves is the thing this exists to stop (plan L2/L3 — screens own no layout).
import type { ReactNode } from "react";
import type { SpaceKey } from "../lib/tokenMeta";
import styles from "./Toolbar.module.css";

export interface ToolbarProps {
  children: ReactNode;
  /** Gap between controls, from the space scale. */
  gap?: SpaceKey;
}

export function Toolbar({ children, gap = 12 }: ToolbarProps) {
  return (
    <div className={styles.toolbar} style={{ gap: `var(--space-${gap})` }}>
      {children}
    </div>
  );
}
