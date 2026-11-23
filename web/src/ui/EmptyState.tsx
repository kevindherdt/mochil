// EmptyState — one muted line + the one action that fixes it (plan §4.2, L5).
import type { ReactNode } from "react";
import { Button } from "./Button";
import styles from "./EmptyState.module.css";

export interface EmptyStateProps {
  message: string;
  action?: { label: string; onClick: () => void } | undefined;
  icon?: ReactNode;
}

export function EmptyState({ message, action, icon }: EmptyStateProps) {
  return (
    <div className={styles.state}>
      {icon}
      <p>{message}</p>
      {action && <Button variant="secondary" onClick={action.onClick}>{action.label}</Button>}
    </div>
  );
}
