// ui/TaskList — one row per background task: what it found, how much of it is left, and the actions on it.
//
// A primitive rather than a piece of the editor because law L2 says a feature component owns no layout, and the row
// this draws (counts on the left, actions on the right, one row per task) is the whole of its layout. It knows nothing
// about metadata: it is handed rows and callbacks, the way Menu and FilterChips are, so the editor's row actions live
// with the editor and the shape lives here.
//
// Every action is a real button, so Tab reaches it and Enter presses it; the visible word is short ("Review") and the
// accessible name says which row it acts on ("Review Find ISBNs"), because four identical "Review"s in a list tell a
// screen reader nothing.
import type { ReactNode } from "react";
import { Button } from "./Button";
import { Inline } from "./Inline";
import { Text } from "./Text";
import styles from "./TaskList.module.css";

export interface TaskRow {
  /** The task's id in the URL the actions call. Also the row's key. */
  tool: string;
  title: string;
  state: "idle" | "running" | "finished";
  found: number;
  dismissed: number;
  /** ISO timestamp of the last run; "" when it has never run. */
  ranAt: string;
  /** Only tools that start a job may offer Re-run; a read (the cleanup rules) does not. */
  canRerun: boolean;
  /**
   * A tool whose answer is computed when it is asked for, not stored (the cleanup rules). It has no findable count until
   * someone reviews it, so it must not be gated on one: a row reading "0 found" with Review switched off would hide the
   * feature completely, which is exactly what it did before this flag existed.
   */
  preview: boolean;
}

export interface TaskListProps {
  rows: TaskRow[];
  /** The row whose action is in flight. Only that row's buttons dim: dimming all three made the panel flash on
   *  every click, which read as the whole thing redrawing. */
  busyTool?: string | undefined;
  onReview: (tool: string) => void;
  /** No books: "the rest of this run", which the server reads as all of them. */
  onDiscardRest: (tool: string, books: string[]) => void;
  onRestore: (tool: string) => void;
  onRerun: (tool: string) => void;
  /** Shown above the rows when a request was refused. */
  problem?: ReactNode;
}

const STATE_WORD: Record<TaskRow["state"], string> = { idle: "idle", running: "running", finished: "finished" };

/** "10:41" for a timestamp, "" when there is not one — a row that never ran must not print "Invalid Date". */
function ranClock(iso: string): string {
  if (iso === "") return "";
  const at = new Date(iso);
  if (Number.isNaN(at.getTime())) return "";
  return at.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function TaskList({ rows, busyTool, onReview, onDiscardRest, onRestore, onRerun, problem }: TaskListProps) {
  return (
    <div className={styles.panel}>
      {problem !== undefined && (
        <div className={styles.problem} role="alert">
          {problem}
        </div>
      )}
      {rows.length === 0 ? (
        <div className={styles.empty}>
          <Text size="12" muted>
            No tool has run for this library yet. Start one above and its results appear here.
          </Text>
        </div>
      ) : (
        <ul className={styles.rows} aria-label="Metadata tasks">
          {rows.map((row) => {
            const left = Math.max(0, row.found - row.dismissed);
            const running = row.state === "running";
            const rowBusy = busyTool === row.tool;
            const clock = ranClock(row.ranAt);
            return (
              <li key={row.tool} className={styles.row}>
                <div className={styles.name}>
                  <Text size="13">{row.title}</Text>
                  <Inline gap={4} align="center" wrap>
                    <Text size="11" muted>
                      {row.preview
                        ? row.dismissed > 0
                          ? `${row.dismissed} dismissed`
                          : "checked when you review"
                        : row.state === "idle" && row.found === 0
                          ? "nothing found yet"
                          : `${row.found} found · ${row.dismissed} dismissed`}
                    </Text>
                    <span className={`${styles.state} ${styles[row.state]}`}>{STATE_WORD[row.state]}</span>
                    {clock !== "" && (
                      <Text size="11" muted>
                        · run {clock}
                      </Text>
                    )}
                  </Inline>
                </div>
                <Inline gap={8} align="center">
                  <Button
                    // Secondary, not primary. A primary button on all three rows made an idle panel look like three things
                    // waiting to be reviewed — the counts say that, the colour should not.
                    variant="secondary"
                    // Nothing left to review is not a failure, it is a finished answer: the button says so by being off.
                    disabled={rowBusy || running || (!row.preview && left === 0)}
                    aria-label={`Review ${row.title}`}
                    title={
                      row.preview
                        ? `Work out what ${row.title} would change.`
                        : row.found === 0
                          ? `Nothing found yet: run ${row.title} first.`
                          : left === 0
                            ? "Everything this run found has been discarded."
                            : `Review what is left of ${row.title}.`
                    }
                    onClick={() => onReview(row.tool)}
                  >
                    {row.preview ? "Review" : `Review ${left}`}
                  </Button>
                  {/* A preview tool has nothing to discard until its answer has been looked at, and then the discards are
                      per book in the review panel. Offering the action here would promise something it cannot do. */}
                  {!row.preview && (
                  <Button
                    variant="secondary"
                    disabled={rowBusy || running || left === 0}
                    aria-label={`Discard the rest of ${row.title}`}
                    title="Put every remaining result of this task aside. Another tool's results for the same books stay."
                    onClick={() => onDiscardRest(row.tool, [])}
                  >
                    Discard rest
                  </Button>
                  )}
                  {row.canRerun && (
                    <Button
                      variant="secondary"
                      disabled={rowBusy || running}
                      aria-label={`Run ${row.title} again`}
                      title="Start this tool again. It keeps running if this page closes."
                      onClick={() => onRerun(row.tool)}
                    >
                      Re-run
                    </Button>
                  )}
                  {row.dismissed > 0 && (
                    <Button
                      variant="secondary"
                      disabled={rowBusy}
                      aria-label={`Bring back ${row.title}`}
                      title="Bring back everything this task had put aside. The other tasks are untouched."
                      onClick={() => onRestore(row.tool)}
                    >
                      Restore
                    </Button>
                  )}
                </Inline>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
