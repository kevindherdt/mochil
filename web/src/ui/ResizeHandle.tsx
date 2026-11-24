// ResizeHandle — the line between two columns, which is also the thing you drag to change their widths.
//
// It lives in `ui/` rather than inside a screen because a screen may not own CSS (law L2), and because more than one
// table wants it. The pointer events are passed through: what resizing means belongs to the table that owns the widths.
import type { PointerEvent } from "react";
import styles from "./ResizeHandle.module.css";

export interface ResizeHandleProps {
  onPointerDown: (event: PointerEvent<HTMLSpanElement>) => void;
  onDoubleClick?: () => void;
  /** Placed at the right edge of a cell by the caller; this only sizes and paints the target. */
  title?: string;
}

export function ResizeHandle({ onPointerDown, onDoubleClick, title = "Drag to resize · double-click to reset" }: ResizeHandleProps) {
  return (
    <span
      role="separator"
      aria-orientation="vertical"
      aria-label={title}
      onPointerDown={onPointerDown}
      onDoubleClick={onDoubleClick}
      title={title}
      className={styles.handle}
      style={{ position: "absolute", right: -5, top: 0, height: "100%", width: 9, cursor: "col-resize", touchAction: "none" }}
    />
  );
}
