import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { EditorTools } from "../../src/features/editor/EditorTools";
import type { EditorBook, EditorChanges } from "../../src/features/editor/api/editor";

/** Only the fields isbnFills() reads. */
const books = [
  {
    id: "b1",
    title: "Dune",
    authors: ["Frank Herbert"],
    identifiers: "",
    isbnFound: "9780441013593",
    issues: [],
    tags: [],
    collections: [],
    custom: {},
  },
] as unknown as EditorBook[];

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("EditorTools tasks trigger", () => {
  it("opens the task rows and hands a task's own changes up for review", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) => {
        if (url.startsWith("/api/metadata/editor/tasks")) {
          return new Response(
            JSON.stringify({
              tasks: [{ tool: "isbn", title: "Find ISBNs", state: "finished", found: 1, dismissed: 0, ran_at: "" }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        return new Response(JSON.stringify({ status: "idle" }), { status: 200, headers: { "Content-Type": "application/json" } });
      }),
    );
    const onReviewTask = vi.fn<(tool: string, title: string, changes: EditorChanges) => void>();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <EditorTools
          libraryId="lib-smoke"
          books={books}
          selected={[]}
          editable={["identifiers"]}
          hermes={{ helper: false, isbn: false, match: false }}
          onReviewTask={onReviewTask}
          pending={0}
          calibre={false}
          open
          refetch={async () => {}}
        />
      </QueryClientProvider>,
    );

    // Nothing is staged until a row is reviewed: the trigger is a door, not a button that consumes whatever is computed.
    expect(onReviewTask).not.toHaveBeenCalled();
    screen.getByRole("button", { name: "Tasks" }).click();

    // The tool chip in the row above says "Find ISBNs" too, so wait for the row's own Review button.
    await screen.findByRole("button", { name: "Review Find ISBNs" });
    screen.getByRole("button", { name: "Review Find ISBNs" }).click();
    await waitFor(() => expect(onReviewTask).toHaveBeenCalledTimes(1));
    expect(onReviewTask.mock.calls[0]?.[0]).toBe("isbn");
    expect(onReviewTask.mock.calls[0]?.[2]).toEqual({ b1: { identifiers: "isbn:9780441013593" } });
  });
});
