// features/editor/EditorTools.tsx — the bulk metadata tools: find first, then deal with what was found.
//
// Per-book lookups are useless on a library of thousands, so this follows the legacy editor's two stages:
//
//   1. FIND — a background job on the server, scoped to the selection or the whole library. It keeps running if the page
//      closes, it can be stopped, and starting it again continues where a stop left off. Four of them: find ISBNs, find
//      Hardcover matches, and the two AI asks that only decide the cases the plain lookups were unsure about.
//   2. THE TASKS PANEL — one row per tool run, with its own counts, its own Review, its own dismissal set and its own
//      re-run. It replaced one "Stage N" button fed by every tool at once, where the button meant something different
//      depending on which tool ran last and the only way to a result was a chain (cancel the ISBNs, then the matches
//      appear, then discard those). Reviewing a row stages into the same pending-edits map a typed cell goes into, so
//      the review → save → undo path is unchanged; matches still stage one of two ways, chosen beside the panel: FILL
//      rewrites only the fields the book has nothing in, REPLACE rewrites what the match disagrees about.
//
// Nothing here writes metadata by itself. A tool that wrote directly would be a tool nobody dares run on 4 800 books.
import { useEffect, useId, useMemo, useState } from "react";
import { Search } from "lucide-react";
import { toneInk } from "./tone";
import { Button, FilterChips, Inline, Menu, Stack, Text, Toolbar } from "../../ui";
import { MATCH_WRITABLE, useMatchFields } from "./matchFields";
import { TasksPanel } from "./TasksPanel";
import {
  FIELD_LABELS,
  TASK_JOBS,
  isbnFills,
  jobState,
  matchFills,
  matchReplaces,
  startJob,
  stopJob,
  tidyBooks,
  writeMetadataFiles,
  type MetadataFileOutcome,
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
  /** Spoken name and the progress line. */
  label: string;
  /** The chip in the panel: the same tool in two or three words, because the panel is a row of buttons now. */
  short: string;
  /** What it does, on hover — the panel stopped carrying a paragraph under every tool. */
  hint: string;
  needs?: "isbn" | "match";
}

const TOOLS: ToolDefinition[] = [
  {
    id: "matches",
    label: "Find Hardcover matches",
    short: "Find matches",
    hint: "Looks each book up by ISBN, else by title. Slow — about a minute per sixty books — and safe to stop.",
  },
  {
    id: "isbn",
    label: "Find ISBNs",
    short: "Find ISBNs",
    hint: "Reads each book for the ISBN printed in it. Nothing changes until you use what it finds.",
  },
  {
    id: "match-ask",
    label: "Ask Hermes about matches",
    short: "Hermes: matches",
    hint: "For books the match search was unsure about. She may only pick one of Hardcover's own candidates, and only when confident.",
    needs: "match",
  },
  {
    id: "isbn-ask",
    label: "Ask Hermes about ISBNs",
    short: "Hermes: ISBNs",
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
  onReviewTask,
  refetch,
  open,
  pending,
  calibre,
}: {
  libraryId: string;
  books: EditorBook[];
  selected: string[];
  editable: readonly EditorField[];
  hermes: HermesAbilities;
  /** Stage one task's changes and open the review on them. The screen owns staging and the diff, so the panel only
   *  says which tool and what it would change. */
  onReviewTask: (tool: string, title: string, changes: EditorChanges) => void;
  /** How many edits are staged but not saved. Writing files writes what is saved, so this has to be zero first. */
  pending: number;
  /** Calibre owns this library's folders and its own metadata.opf, so writing files there is not offered at all. */
  calibre: boolean;
  /** Owned by the screen: the trigger lives in the toolbar, the panel expands under it. */
  open: boolean;
  /** react-query's refetch — a stable identity, so the polling effect does not restart on every render. */
  refetch: () => Promise<unknown>;
}) {
  const [running, setRunning] = useState<ToolId | undefined>(undefined);
  const [job, setJob] = useState<JobState | undefined>(undefined);
  const [problem, setProblem] = useState<string | undefined>(undefined);
  // What the last job said when it finished. A job can end in under a second, and the progress line only exists while a
  // job is running, so without this the server's own message ("2 already have an ISBN") would never be read.
  const [summary, setSummary] = useState<string | undefined>(undefined);
  // The task rows open under the trigger rather than in a dialog: they are a list to keep an eye on while the table
  // behind them is being worked, not a thing to be dismissed before you can see the library again.
  const [tasksOpen, setTasksOpen] = useState(false);
  const tasksId = useId();
  // Writing metadata.opf beside the books: what makes the work leave Mochil and land in Calibre's import.
  const [embed, setEmbed] = useState(false);
  const [overwriteSidecar, setOverwriteSidecar] = useState(false);
  const [writing, setWriting] = useState(false);
  const [writeOutcome, setWriteOutcome] = useState<MetadataFileOutcome | undefined>(undefined);
  const [writeProblem, setWriteProblem] = useState<string | undefined>(undefined);
  const [writeOpen, setWriteOpen] = useState(false);

  const available = TOOLS.filter((tool) => tool.needs === undefined || (hermes.helper && hermes[tool.needs]));
  const scope = selected.length > 0 ? selected : undefined;
  const scopeBooks = scope === undefined ? books : books.filter((book) => scope.includes(book.id));

  // Two ways to take a match, and the difference is the whole point: filling leaves the books that are already named
  // something alone, replacing is how "The Last Wish (Polish)" becomes "The Last Wish". Both stage into the same
  // pending map, so both get the same per-book diff, apply and undo.
  const [matchMode, setMatchMode] = useState<"fill" | "replace">("fill");
  // What a match may write here, chosen per library. Everything below that takes the match — the two counts and the
  // staging itself — narrows to it, so a series name that must not move is never even proposed.
  const [matchFields, setMatchFields] = useMatchFields(libraryId);
  const matchEditable = useMemo(
    () => matchFields.filter((field) => editable.includes(field)),
    [editable, matchFields],
  );
  const fillChanges = useMemo(() => matchFills(scopeBooks, matchEditable), [scopeBooks, matchEditable]);
  const replaceChanges = useMemo(() => matchReplaces(scopeBooks, matchEditable), [scopeBooks, matchEditable]);
  // A Calibre library keeps its own title, authors and series (the server answers 409 for them), so offering to
  // replace them there would be offering a refusal.
  const canReplace = editable.includes("title");
  const fillCount = Object.keys(fillChanges).length;
  const replaceCount = Object.keys(replaceChanges).length;

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

  /** False when the server refused it. `problem` says why either way; the Tasks panel needs the answer because it
   *  started the job from its own row. */
  async function start(tool: ToolId, options: { recheck?: boolean; ids?: string[] } = {}): Promise<boolean> {
    setProblem(undefined);
    setSummary(undefined);
    try {
      await startJob(libraryId, tool, options.ids ?? selected, { recheck: tool === "isbn" && options.recheck === true });
      setRunning(tool);
      setJob(undefined);
      return true;
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "That could not be started.");
      return false;
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

  async function writeFiles() {
    setWriteProblem(undefined);
    setWriteOutcome(undefined);
    setWriting(true);
    try {
      setWriteOutcome(await writeMetadataFiles(libraryId, selected, { embed, overwrite: overwriteSidecar }));
    } catch (error) {
      setWriteProblem(error instanceof Error ? error.message : "That could not be written.");
    } finally {
      setWriting(false);
    }
  }

  /**
   * What one task's Review would stage, with that task's dismissed books left out — the whole point of the panel:
   * what the ISBN run put aside must not be offered by the match run, and vice versa.
   */
  async function changesFor(tool: string, excluded: string[]): Promise<EditorChanges> {
    const skip = new Set(excluded);
    const scoped = skip.size === 0 ? scopeBooks : scopeBooks.filter((book) => !skip.has(book.id));
    if (tool === "tidy") return (await tidyBooks(libraryId, scoped.map((book) => book.id))).changes;
    if (tool === "isbn") return isbnFills(scoped, editable);
    if (tool === "match") return matchMode === "replace" ? matchReplaces(scoped, matchEditable) : matchFills(scoped, matchEditable);
    return {};
  }

  /** A re-run is the whole library: the row is about a run, not about whatever happens to be selected now. */
  async function rerun(tool: string): Promise<void> {
    const jobId = TASK_JOBS[tool];
    if (jobId === undefined) return;
    if (!(await start(jobId, { ids: [] }))) throw new Error("That could not be started.");
  }

  const scopeLabel = selected.length > 0 ? `${selected.length} selected` : "the whole library";

  // Closed with nothing to report: render nothing at all, rather than an empty row where a panel would be. Hooks are
  // all above this point, so the early return is safe.
  if (!open && running === undefined && problem === undefined && summary === undefined) return null;

  return (
    <Stack gap={8}>
      <Inline gap={8} align="center" wrap>
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
        {/* The one door to what the tools found. Each row carries its own counts and actions; nothing here stages
            anything by itself. */}
        <Button variant="secondary" aria-expanded={tasksOpen} aria-controls={tasksId} onClick={() => setTasksOpen(!tasksOpen)}>
          Tasks
        </Button>
        {canReplace && (fillCount > 0 || replaceCount > 0) && (
          <FilterChips
            items={[
              { id: "fill", label: "Fill empty fields" },
              { id: "replace", label: "Replace fields" },
            ]}
            activeId={matchMode}
            onSelect={(id) => setMatchMode(id === "replace" ? "replace" : "fill")}
          />
        )}
        {/* Which fields a match may write. A match carries a title, a series, a year and a blurb, and taking it used to
            mean all of them — sometimes the blurb is the point and the series name is the thing that must not move. */}
        {available.some((tool) => tool.id === "matches") && (
          <Inline gap={8} align="center" wrap>
            <Text size="11" muted>
              Match may write
            </Text>
            {MATCH_WRITABLE.filter((field) => editable.includes(field)).map((field) => (
              <label
                key={field}
                style={{
                  display: "inline-flex",
                  gap: "var(--space-4)",
                  alignItems: "center",
                  fontSize: "var(--font-size-11)",
                }}
              >
                <input
                  type="checkbox"
                  checked={matchFields.includes(field)}
                  onChange={() =>
                    setMatchFields(
                      matchFields.includes(field)
                        ? matchFields.filter((chosen) => chosen !== field)
                        : [...matchFields, field],
                    )
                  }
                />
                {FIELD_LABELS[field]}
              </label>
            ))}
          </Inline>
        )}
      </Inline>

      {tasksOpen && (
        <div id={tasksId}>
          <TasksPanel libraryId={libraryId} buildChanges={changesFor} onReview={onReviewTask} onRerun={rerun} />
        </div>
      )}

      {open && (
        <div style={{ border: "1px solid var(--color-line)", borderRadius: "var(--radius-12)", padding: "var(--space-8) var(--space-12)", background: "var(--color-surface)" }}>
          <Stack gap={8}>
            {/* One compact row: a chip per tool, the explanation on hover. Four cards with two lines of prose each ate
                half the screen the table needed — Kevin: "declutter... so we can use a lot of free space to actually
                show the table". */}
            <Toolbar gap={8}>
              <Text size="11" muted>
                Runs on the server against {scopeLabel}; nothing is written until you save.
              </Text>
              <div style={{ flex: 1 }} />
              {available.map((tool) =>
                // The ISBN tool asks a second question — whether to read the books that already have a number — so it
                // carries that choice on itself. A checkbox that only ever applies to one of four tools was the panel's
                // one piece of permanent clutter.
                tool.id === "isbn" ? (
                  <span key={tool.id} title={tool.hint}>
                    <Menu
                      icon={Search}
                      text={tool.short}
                      label={`${tool.label}: ${tool.hint}`}
                      // Every other tool stops while one runs. This one did not, and starting an ISBN job mid-matches
                      // tore down the matches job's polling: its Stop button vanished and its results never refreshed.
                      disabled={running !== undefined}
                      items={[
                        { id: "new", label: "Books without an ISBN" },
                        { id: "recheck", label: "Also re-read books that have one" },
                      ]}
                      onSelect={(id) => void start("isbn", { recheck: id === "recheck" })}
                    />
                  </span>
                ) : (
                  <Button
                    key={tool.id}
                    variant="secondary"
                    title={tool.hint}
                    disabled={running !== undefined}
                    onClick={() => void start(tool.id)}
                    style={running === tool.id ? { borderColor: "var(--color-accent)", color: "var(--color-accent)" } : undefined}
                  >
                    {tool.short}
                  </Button>
                ),
              )}
              {!calibre && (
                <Button
                  variant="secondary"
                  title="Write a metadata.opf beside each selected book from the saved metadata: Calibre reads it when it adds a folder. Opening this does not write anything."
                  aria-expanded={writeOpen}
                  onClick={() => setWriteOpen(!writeOpen)}
                >
                  Write metadata…
                </Button>
              )}
            </Toolbar>

            <Inline gap={16} align="center" wrap>
              {calibre && (
                <Text size="11" muted>
                  Calibre owns this library's folders and its own metadata.opf, so writing files stays Calibre's job.
                </Text>
              )}
              {available.length < TOOLS.length && (
                <Text size="11" muted>
                  “Ask Hermes” needs a helper model — Settings → Hermes.
                </Text>
              )}
            </Inline>

            {writeOpen && !calibre && (
              <div style={{ border: "1px solid var(--color-line)", borderRadius: "var(--radius-8)", padding: "var(--space-8) var(--space-12)" }}>
                <Stack gap={4}>
                  <Text size="11" muted>
                    Writes metadata.opf beside each selected book from what is saved — Calibre reads it when it adds the
                    folder. Nothing is written into a library Mochil does not own.
                  </Text>
                  <Inline gap={16} align="center" wrap>
                    <label style={{ display: "flex", alignItems: "center", gap: "var(--space-4)", fontSize: "var(--font-size-11)" }}>
                      <input type="checkbox" checked={embed} onChange={(event) => setEmbed(event.target.checked)} />
                      <span title="Also write the fields into the EPUB itself, so the file carries them with no sidecar beside it.">
                        Also into the EPUB
                      </span>
                    </label>
                    <label style={{ display: "flex", alignItems: "center", gap: "var(--space-4)", fontSize: "var(--font-size-11)" }}>
                      <input
                        type="checkbox"
                        checked={overwriteSidecar}
                        onChange={(event) => setOverwriteSidecar(event.target.checked)}
                      />
                      <span title="A metadata.opf that is already beside a book may be Calibre's, so it is left alone unless this is ticked.">
                        Replace an existing one
                      </span>
                    </label>
                  </Inline>
                  <Inline gap={8} align="center" wrap>
                    <Button
                      variant="secondary"
                      disabled={selected.length === 0 || selected.length > 100 || pending > 0 || writing}
                      onClick={() => void writeFiles()}
                    >
                      {writing ? "Writing…" : `Write for ${selected.length} book${selected.length === 1 ? "" : "s"}`}
                    </Button>
                    {pending > 0 && (
                      <Text size="11" muted>
                        Save the pending changes first — this writes what is saved.
                      </Text>
                    )}
                    {pending === 0 && selected.length === 0 && (
                      <Text size="11" muted>
                        Select the books to write.
                      </Text>
                    )}
                    {selected.length > 100 && (
                      <Text size="11" muted>
                        100 books at a time.
                      </Text>
                    )}
                    {writeOutcome !== undefined && (
                      <Text size="11" muted>
                        {writeOutcome.summary.written} written
                        {writeOutcome.summary.exists > 0 ? `, ${writeOutcome.summary.exists} already there` : ""}
                        {writeOutcome.summary.failed > 0 ? `, ${writeOutcome.summary.failed} failed` : ""}
                      </Text>
                    )}
                  </Inline>
                  {writeProblem !== undefined && (
                    <div style={{ fontSize: "var(--font-size-11)", color: toneInk("poor") }}>{writeProblem}</div>
                  )}
                  {writeOutcome?.results
                    .filter((entry) => entry.status === "failed")
                    .slice(0, 3)
                    .map((entry) => (
                      <Text key={entry.id} size="11" muted>
                        {entry.title}: {entry.message}
                      </Text>
                    ))}
                </Stack>
              </div>
            )}

            {problem !== undefined && <div style={{ fontSize: "var(--font-size-11)", color: toneInk("poor") }}>{problem}</div>}
            {job !== undefined && job.error !== "" && <Text size="11">{job.error}</Text>}
          </Stack>
        </div>
      )}
    </Stack>
  );
}
