// StaleNotice — the strip <Async> shows when a *refetch* fails but the last good data is still on screen.
//
// A refresh that fails is not the same event as a load that fails. The list the person was reading is still valid;
// replacing it with a full error panel throws away their place, their scroll position and anything selected inside
// it — for one dropped request that a moment later would have succeeded. So the data stays, and this says so,
// with the one action that helps.
import { Button } from "./Button";
import { Text } from "./Text";
import styles from "./StaleNotice.module.css";

export interface StaleNoticeProps {
  onRetry?: (() => void) | undefined;
  message?: string;
}

export function StaleNotice({ onRetry, message = "Couldn't refresh — this is the last version that loaded." }: StaleNoticeProps) {
  return (
    <div className={styles.notice} role="status">
      <Text size="13" tone="warning">
        {message}
      </Text>
      {onRetry && (
        <Button variant="secondary" onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  );
}
