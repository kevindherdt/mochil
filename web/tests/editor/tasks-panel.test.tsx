import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TasksPanel } from "../../src/features/editor/TasksPanel";

/** The server's answer to GET /api/metadata/editor/tasks, in the shape the contract sends. */
interface FakeTask {
  tool: string;
  title: string;
  state: "idle" | "running" | "finished";
  found: number;
  dismissed: number;
  ran_at: string;
}

/**
 * A stand-in for the tasks endpoints that keeps its own counts: a discard raises them and a restore clears them, so the
 * test asserts what the panel *shows* after the round trip rather than what it hoped for.
 */
function server(initial: Record<string, FakeTask[]>): { calls: { url: string; init: RequestInit }[] } {
  const calls: { url: string; init: RequestInit }[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init: RequestInit = {}) => {
      calls.push({ url, init });
      const library = new URL(url, "http://mochil.test").searchParams.get("library") ?? "";
      const tasks = initial[library] ?? [];
      const tool = /\/tasks\/([^/]+)\/dismiss/.exec(url)?.[1] ?? "";
      const task = tasks.find((entry) => entry.tool === decodeURIComponent(tool));
      if (init.method === "POST" && task !== undefined) {
        const body = JSON.parse(String(init.body)) as { books: string[] };
        task.dismissed = body.books.length === 0 ? task.found : task.dismissed + body.books.length;
        return new Response(null, { status: 204 });
      }
      if (init.method === "DELETE" && task !== undefined) {
        task.dismissed = 0;
        return new Response(null, { status: 204 });
      }
      return new Response(JSON.stringify({ tasks }), { status: 200, headers: { "Content-Type": "application/json" } });
    }),
  );
  return { calls };
}

/** Three tools, the state the design's own example row describes. */
function threeTasks(): FakeTask[] {
  return [
    { tool: "isbn", title: "Find ISBNs", state: "finished", found: 41, dismissed: 8, ran_at: "2026-11-25T10:41:00Z" },
    { tool: "match", title: "Hardcover matches", state: "finished", found: 12, dismissed: 0, ran_at: "2026-11-25T10:38:00Z" },
    { tool: "tidy", title: "Clean up titles", state: "idle", found: 0, dismissed: 0, ran_at: "" },
  ];
}

const noop = () => {};

function mount(libraryId: string, props: Partial<Parameters<typeof TasksPanel>[0]> = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onReview = vi.fn();
  const onRerun = vi.fn(async () => {});
  const view = render(
    <QueryClientProvider client={client}>
      <TasksPanel
        libraryId={libraryId}
        buildChanges={() => ({})}
        onReview={onReview}
        onRerun={onRerun}
        {...props}
      />
    </QueryClientProvider>,
  );
  return { ...view, onReview, onRerun };
}

function posts(calls: { url: string; init: RequestInit }[]) {
  return calls.filter((call) => call.init.method === "POST");
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("TasksPanel", () => {
  it("draws one row per task the server lists, with its own counts", async () => {
    server({ "lib-rows": threeTasks() });
    mount("lib-rows");

    expect(await screen.findAllByRole("listitem")).toHaveLength(3);
    expect(screen.getByText("Find ISBNs")).toBeTruthy();
    expect(screen.getByText("41 found · 8 dismissed")).toBeTruthy();
    expect(screen.getByText("Hardcover matches")).toBeTruthy();
    expect(screen.getByText("12 found · 0 dismissed")).toBeTruthy();
    // The cleanup rules have no stored count at all — the server answers 0 and works the answer out on demand — so the
    // row says that rather than pretending to know.
    expect(screen.getByText("checked when you review")).toBeTruthy();
  });

  // The server has no stored count for the cleanup rules: its preview is worked out per book set on demand, so the row
  // arrives as "0 found". Gating Review on that count made the tool unreachable — which it was, until this test.
  it("lets a tool with no stored count be reviewed anyway", async () => {
    server({
      "lib-tidy": [
        { tool: "isbn", title: "Find ISBNs", state: "idle", found: 0, dismissed: 0, ran_at: "" },
        { tool: "match", title: "Hardcover matches", state: "idle", found: 0, dismissed: 0, ran_at: "" },
        { tool: "tidy", title: "Clean up titles", state: "idle", found: 0, dismissed: 0, ran_at: "" },
      ],
    });
    mount("lib-tidy");

    await screen.findByText("Clean up titles");
    const review = screen.getByRole("button", { name: "Review Clean up titles" });
    expect(review.hasAttribute("disabled")).toBe(false);
    expect(screen.getByText("checked when you review")).toBeTruthy();
    // Nothing found yet, so there is nothing to discard: the action would promise what it cannot do.
    expect(screen.queryByRole("button", { name: "Discard the rest of Clean up titles" })).toBeNull();
    // Whereas a tool that really found nothing stays off.
    expect(screen.getByRole("button", { name: "Review Find ISBNs" }).hasAttribute("disabled")).toBe(true);
  });

  it("says which task each action acts on, so four identical buttons are not four identical names", async () => {
    server({ "lib-names": threeTasks() });
    mount("lib-names");

    await screen.findByText("Find ISBNs");
    expect(screen.getByRole("button", { name: "Review Find ISBNs" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Discard the rest of Find ISBNs" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Run Find ISBNs again" })).toBeTruthy();
    // The cleanup rules are a read, not a job, so their row has no re-run.
    expect(screen.queryByRole("button", { name: "Run Clean up titles again" })).toBeNull();
  });

  it("posts a discard with the library and an empty book list — the rest of the run, not every id of it", async () => {
    const fake = server({ "lib-discard": threeTasks() });
    mount("lib-discard");

    await screen.findByText("Find ISBNs");
    screen.getByRole("button", { name: "Discard the rest of Find ISBNs" }).click();

    await waitFor(() => expect(posts(fake.calls)).toHaveLength(1));
    const call = posts(fake.calls)[0];
    expect(call?.url).toBe("/api/metadata/editor/tasks/isbn/dismiss");
    expect(JSON.parse(String(call?.init.body))).toEqual({ library: "lib-discard", books: [] });
    // And the row takes the server's new count rather than keeping the one it had.
    expect(await screen.findByText("41 found · 41 dismissed")).toBeTruthy();
  });

  it("restores one task without touching the others", async () => {
    const fake = server({ "lib-restore": threeTasks() });
    mount("lib-restore");

    await screen.findByText("Find ISBNs");
    screen.getByRole("button", { name: "Bring back Find ISBNs" }).click();

    await waitFor(() => expect(fake.calls.some((call) => call.init.method === "DELETE")).toBe(true));
    const call = fake.calls.find((entry) => entry.init.method === "DELETE");
    expect(call?.url).toBe("/api/metadata/editor/tasks/isbn/dismiss?library=lib-restore");
    // The count comes back and the Restore button goes away with it; the other rows keep their own answers.
    expect(await screen.findByText("41 found · 0 dismissed")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Bring back Find ISBNs" })).toBeNull();
    expect(screen.getByRole("button", { name: "Review Find ISBNs" })).toBeTruthy();
  });

  it("turns Review off when a run has nothing left to review", async () => {
    server({
      "lib-empty": [
        { tool: "isbn", title: "Find ISBNs", state: "finished", found: 41, dismissed: 41, ran_at: "2026-11-25T10:41:00Z" },
      ],
    });
    mount("lib-empty");

    await screen.findByText("Find ISBNs");
    expect((screen.getByRole("button", { name: "Review Find ISBNs" }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: "Discard the rest of Find ISBNs" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("keeps what was dismissed when the library is switched away and back", async () => {
    // The bug this guards: a dismissal used to be forgotten on a library switch, so the same 41 found ISBNs were
    // offered again every time the shelf was left and returned to.
    const fake = server({
      "lib-a": threeTasks(),
      "lib-b": [{ tool: "isbn", title: "Find ISBNs", state: "finished", found: 7, dismissed: 0, ran_at: "" }],
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const panel = (libraryId: string) => (
      <QueryClientProvider client={client}>
        <TasksPanel libraryId={libraryId} buildChanges={() => ({})} onReview={noop} onRerun={async () => {}} />
      </QueryClientProvider>
    );
    const view = render(panel("lib-a"));

    await screen.findByText("Find ISBNs");
    screen.getByRole("button", { name: "Discard the rest of Find ISBNs" }).click();
    await waitFor(() => expect(posts(fake.calls)).toHaveLength(1));
    expect(await screen.findByText("41 found · 41 dismissed")).toBeTruthy();

    // The other shelf has its own answer, and none of this library's.
    view.rerender(panel("lib-b"));
    expect(await screen.findByText("7 found · 0 dismissed")).toBeTruthy();

    // Back again: the decision made on lib-a is still there, and Review is still off.
    view.rerender(panel("lib-a"));
    expect(await screen.findByText("41 found · 41 dismissed")).toBeTruthy();
    expect((screen.getByRole("button", { name: "Review Find ISBNs" }) as HTMLButtonElement).disabled).toBe(true);
  });
});
