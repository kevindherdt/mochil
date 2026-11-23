// features/editor/EditorTools.tsx — the bulk metadata tools: find first, then use what was found.
//
// Per-book lookups are useless on a library of thousands, so this follows the legacy editor's two stages:
//
//   1. FIND — a background job on the server, scoped to the selection or the whole library. It keeps running if the page
//      closes, it can be stopped, and starting it again continues where a stop left off. Four of them: find ISBNs, find
//      Hardcover matches, and the two AI asks that only decide the cases the plain lookups were unsure about.
//   2. USE — stage the results as ordinary pending edits ("Use clear matches" fills only empty fields, "Use found ISBNs"
//      only books without one), which then go through the normal review → save → undo path.
//
// Nothing here writes metadata by itself. A tool that wrote directly would be a tool nobody dares run on 4 800 books.
import { useEffect, useMemo, useState } from "react";
import { Button, Heading, Inline, Stack, Text } from "../../ui";
import {
  isbnFills,
  jobState,
  matchFills,
  startJob,
  stopJob,
  type EditorBook,
  type EditorChanges,
  type EditorField,
  type EditorTool as ToolId,
  type JobState,
} from "./api/editor";

export interface HermesAbilities {
  /** A helper model is configured; without one the asks cannot run at all. */
  helper: boolean;
  isbn: boolean;
  match: boolean;
}

interface ToolDefinition {
  id: ToolId;
  label: string;
  hint: string;
  needs?: "isbn" | "match";
}

const TOOLS: ToolDefinition[] = [
  {
    id: "matches",
    label: "Find Hardcover matches",
    hint: "Looks each book up by ISBN, else by title. Slow — about a minute per sixty books — and safe to stop.",
  },
  {
    id: "isbn",
    label: "Find ISBNs",
    hint: "Reads each book for the ISBN printed in it. Nothing changes until you use what it finds.",
  },
  {
    id: "match-ask",
    label: "Ask Hermes about matches",
    hint: "For books the match search was unsure about. She may only pick one of Hardcover's own candidates, and only when confident.",
    needs: "match",
  },
  {
    id: "isbn-ask",
    label: "Ask Hermes about ISBNs",
    hint: "For books where the ISBN search was unsure or found none. She may only pick a number that is really in the book.",
    needs: "isbn",
  },
];

export function EditorTools({
  libraryId,
  books,
  selected,
  editable,
  hermes,
  onStage,
  refetch,
}: {
  libraryId: string;
  books: EditorBook[];
  selected: string[];
  editable: readonly EditorField[];
  hermes: HermesAbilities;
  onStage: (changes: EditorChanges) => void;
  /** react-query's refetch — a stable identity, so the polling effect does not restart on every render. */
  refetch: () => Promise<unknown>;
}) {
  const [open, setOpen] = useState(false);
  const [running, setRunning] = useState<ToolId | undefined>(undefined);
  const [job, setJob] = useState<JobState | undefined>(undefined);
  const [problem, setProblem] = useState<string | undefined>(undefined);
  // What the last job said when it finished. A job can end in under a second, and the progress line only exists while a
  // job is running, so without this the server's own message ("2 already have an ISBN") would never be read.
  const [summary, setSummary] = useState<string | undefined>(undefined);

  const available = TOOLS.filter((tool) => tool.needs === undefined || (hermes.helper && hermes[tool.needs]));
  const scope = selected.length > 0 ? selected : undefined;
  const scopeBooks = scope === undefined ? books : books.filter((book) => scope.includes(book.id));

  const matchChanges = useMemo(() => matchFills(scopeBooks, editable), [scopeBooks, editable]);
  const isbnChanges = useMemo(() => isbnFills(scopeBooks, editable), [scopeBooks, editable]);
  const matchCount = Object.keys(matchChanges).length;
  const isbnCount = Object.keys(isbnChanges).length;

  // Re-attach to a job that is still running on the server (it survives the page being closed).
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      for (const tool of available) {
        try {
          const state = await jobState(libraryId, tool.id);
          if (!cancelled && (state.status === "running" || state.status === "queued")) {
            setRunning(tool.id);
            setJob(state);
            return;
          }
        } catch {
          // A tool the server does not know about is simply not running.
        }
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- availability is derived from hermes, not a dependency of attaching
  }, [libraryId]);

  // While a job runs, ask the server where it is, once a second.
  useEffect(() => {
    if (running === undefined) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const tick = async () => {
      let state: JobState | undefined;
      try {
        state = await jobState(libraryId, running);
      } catch {
        state = undefined;
      }
      if (cancelled) return;
      if (state === undefined) {
        timer = setTimeout(tick, 2000);
        return;
      }
      setJob(state);
      if (state.status === "ready" || state.status === "failed" || state.status === "canceled" || state.status === "idle") {
        const label = TOOLS.find((tool) => tool.id === running)?.label ?? "The tool";
        // A stopped job's last message is the book it was in the middle of, which reads like it is still running.
        const stopped = state.status === "canceled";
        setSummary(
          stopped
            ? `Stopped after ${state.current} of ${state.total}. What was found is kept, and starting again carries on.`
            : state.error !== ""
              ? state.error
              : state.message !== ""
                ? state.message
                : `${label} finished.`,
        );
        setRunning(undefined);
        await refetch();
        return;
      }
      timer = setTimeout(tick, 1000);
    };
    timer = setTimeout(tick, 600);
    return () => {
      cancelled = true;
      if (timer !== undefined) clearTimeout(timer);
    };
  }, [running, libraryId, refetch]);

  async function start(tool: ToolId) {
    setProblem(undefined);
    setSummary(undefined);
    try {
      await startJob(libraryId, tool, selected);
      setRunning(tool);
      setJob(undefined);
      setOpen(true);
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "That could not be started.");
    }
  }

  async function stop() {
    if (running === undefined) return;
    try {
      await stopJob(libraryId, running);
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "That could not be stopped.");
    }
  }

  const scopeLabel = selected.length > 0 ? `${selected.length} selected` : "the whole library";

  return (
    <Stack gap={8}>
      <Inline gap={8} align="center" wrap>
        <Button variant="secondary" onClick={() => setOpen(!open)}>
          Tools
        </Button>
        {running !== undefined && (
          <Text size="11" muted>
            {TOOLS.find((tool) => tool.id === running)?.label ?? "Working"} · {job?.current ?? 0}
            {job !== undefined && job.total > 0 ? ` of ${job.total}` : ""}
            {job?.message !== undefined && job.message !== "" ? ` · ${job.message}` : ""}
          </Text>
        )}
        {running !== undefined && (
          <Button variant="secondary" onClick={() => void stop()}>
            Stop
          </Button>
        )}
        {running === undefined && summary !== undefined && <Text size="11" muted>{summary}</Text>}
        <div style={{ flex: 1 }} />
        {matchCount > 0 && (
          <Button variant="secondary" onClick={() => onStage(matchChanges)}>
            Use clear matches ({matchCount})
          </Button>
        )}
        {isbnCount > 0 && (
          <Button variant="secondary" onClick={() => onStage(isbnChanges)}>
            Use found ISBNs ({isbnCount})
          </Button>
        )}
      </Inline>

      {open && (
        <div style={{ border: "1px solid var(--color-line)", borderRadius: "var(--radius-12)", padding: "var(--space-16)", background: "var(--color-surface)" }}>
          <Stack gap={12}>
            <Stack gap={4}>
              <Heading size="16">Look things up</Heading>
              <Text size="11" muted>
                These run on the server against {scopeLabel} and keep going if you close this page. Nothing is written
                until you stage the results and save.
              </Text>
            </Stack>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: "var(--space-8)" }}>
              {available.map((tool) => (
                <button
                  key={tool.id}
                  type="button"
                  disabled={running !== undefined}
                  onClick={() => void start(tool.id)}
                  style={{
                    textAlign: "left",
                    padding: "8px 10px",
                    borderRadius: "var(--radius-10, 10px)",
                    border: "1px solid var(--color-line)",
                    background: running === tool.id ? "color-mix(in srgb, var(--color-accent) 12%, transparent)" : "transparent",
                    color: "var(--color-text)",
                    font: "inherit",
                    opacity: running !== undefined && running !== tool.id ? 0.5 : 1,
                    cursor: running !== undefined ? "default" : "pointer",
                  }}
                >
                  <div style={{ fontSize: "var(--font-size-12)" }}>{tool.label}</div>
                  <div style={{ fontSize: "10px", opacity: 0.6 }}>{tool.hint}</div>
                </button>
              ))}
            </div>
            {available.length < TOOLS.length && (
              <Text size="11" muted>
                The two “Ask Hermes” tools are hidden because no helper model is configured, or that ability is switched
                off in Settings → Hermes.
              </Text>
            )}
            {problem !== undefined && <div style={{ fontSize: "var(--font-size-12)", color: "var(--color-danger, #e5484d)" }}>{problem}</div>}
            {job !== undefined && job.error !== "" && <Text size="12">{job.error}</Text>}
          </Stack>
        </div>
      )}
    </Stack>
  );
}
