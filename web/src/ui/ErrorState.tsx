// ErrorState — one muted line (coloured as an error) + the one action that fixes it: retry (plan §4.2, L5).
import { Button } from "./Button";
import styles from "./EmptyState.module.css";

export interface ErrorStateProps {
  message?: string;
  onRetry?: (() => void) | undefined;
}

export function ErrorState({ message = "Something went wrong.", onRetry }: ErrorStateProps) {
  return (
    <div className={styles.state}>
      <p className={styles.errorMessage}>{message}</p>
      {onRetry && <Button variant="secondary" onClick={onRetry}>Try again</Button>}
    </div>
  );
}
