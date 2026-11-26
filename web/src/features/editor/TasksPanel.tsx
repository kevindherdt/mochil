// features/editor/TasksPanel.tsx — one row per tool run: what it found, what is left, and the way into each.
//
// This is the door the editor used to lack. Every tool wrote into one staging map behind one "Stage N" button, so the
// button meant something different depending on which tool ran last, and the only route to a result was a chain —
// cancel the ISBNs, then the matches appear, then discard those. Here each tool's run is a row with its own Review,
// its own dismissal set and its own re-run, so a 251-row title diff and a 41-row ISBN fill can be dealt with in one
// sitting without either hiding the other.
//
// The rows themselves are ui/TaskList: a feature component owns no layout (law L2), so this file owns the data and the
// actions and lets the primitive draw them.
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { TaskList, type TaskRow } from "../../ui";
import { TASK_JOBS, fetchEditorTasks, type EditorChanges } from "./api/editor";
import { dismissBooks, isFullyDismissed, restoreDismissals, useTaskDismissals } from "./taskDismissals";

export function TasksPanel({
  libraryId,
  buildChanges,
  onReview,
  onRerun,
}: {
  libraryId: string;
  /** What this task's Review would stage, with the books already put aside left out. The editor owns the tools, so it
   *  owns the answer; the panel only says which task and which books. */
  buildChanges: (tool: string, excluded: string[]) => EditorChanges | Promise<EditorChanges>;
  onReview: (tool: string, title: string, changes: EditorChanges) => void;
  /** Start the tool again through the same job endpoints the Tools row uses. No polling here: the row reports the
   *  job's state, which is the same state the Tools row reads. */
  onRerun: (tool: string) => Promise<void>;
}) {
  const query = useQuery({
    queryKey: ["editor", "tasks", libraryId],
    queryFn: ({ signal }) => fetchEditorTasks(libraryId, signal),
    staleTime: 10_000,
    // Only while something is actually running: the row's state word is the run's state, and three idle rows do not
    // need asking about every three seconds.
    refetchInterval: (current) => ((current.state.data ?? []).some((task) => task.state === "running") ? 3000 : false),
  });
  // Subscribed, not read once: a book discarded in the review panel must raise this row's count without a reload.
  const dismissals = useTaskDismissals(libraryId);
  // Which row is working, not whether anything is: see TaskList's busyTool.
  const [busyTool, setBusyTool] = useState<string | undefined>(undefined);
  const [problem, setProblem] = useState<string | undefined>(undefined);

  const rows: TaskRow[] = (query.data ?? []).map((task) => {
    const ids = dismissals[task.tool] ?? [];
    // The server's count is the truth; the local ids only fill the gap between pressing Discard and the refetch, and
    // "everything" answers with the run's own size because a fully dismissed run has no ids worth listing.
    const dismissed = isFullyDismissed(libraryId, task.tool) ? task.found : Math.max(task.dismissed, ids.length);
    return {
      tool: task.tool,
      title: task.title,
      state: task.state,
      found: task.found,
      dismissed,
      ranAt: task.ranAt,
      canRerun: TASK_JOBS[task.tool] !== undefined,
      // No job behind it: the row's answer is worked out on demand, so it has no stored count to gate on.
      preview: TASK_JOBS[task.tool] === undefined,
    };
  });

  /** One request, then the counts again from the server: the local set is optimistic, the row is not. */
  async function run(tool: string, action: () => Promise<void>): Promise<void> {
    setBusyTool(tool);
    setProblem(undefined);
    try {
      await action();
      await query.refetch();
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "That could not be done.");
    } finally {
      setBusyTool(undefined);
    }
  }

  function titleOf(tool: string): string {
    return rows.find((row) => row.tool === tool)?.title ?? tool;
  }

  return (
    <TaskList
      rows={rows}
      busyTool={busyTool}
      problem={query.isError ? (query.error instanceof Error ? query.error.message : "The tasks could not be loaded.") : problem}
      onReview={(tool) => {
        const title = titleOf(tool);
        setBusyTool(tool);
        setProblem(undefined);
        void (async () => {
          try {
            const excluded = isFullyDismissed(libraryId, tool) ? [] : (dismissals[tool] ?? []);
            onReview(tool, title, await buildChanges(tool, excluded));
          } catch (error) {
            setProblem(error instanceof Error ? error.message : `${title} could not be prepared.`);
          } finally {
            setBusyTool(undefined);
          }
        })();
      }}
      onDiscardRest={(tool) => {
        // No books: the server reads an empty list as "everything this run found", which is what "the rest" means and
        // keeps a 251-row title run from travelling to the server as 251 ids.
        void run(tool, async () => {
          await dismissBooks(libraryId, tool, []);
        });
      }}
      onRestore={(tool) => {
        void run(tool, async () => {
          await restoreDismissals(libraryId, tool);
        });
      }}
      onRerun={(tool) => {
        void run(tool, async () => {
          await onRerun(tool);
        });
      }}
    />
  );
}
