// IconButton — a square, label-less control for a header's action cluster (search / cover-list / AI). `label` is
// required because the button has no visible text and must still be announced.
import type { ComponentType } from "react";
import styles from "./IconButton.module.css";

export interface IconButtonProps {
  icon: ComponentType<{ size?: number }>;
  /** The accessible name — required, since there is no visible text. */
  label: string;
  active?: boolean;
  onClick?: () => void;
}

export function IconButton({ icon: Icon, label, active = false, onClick }: IconButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={active || undefined}
      className={active ? `${styles.button} ${styles.active}` : styles.button}
      onClick={onClick}
    >
      <Icon size={18} />
    </button>
  );
}
