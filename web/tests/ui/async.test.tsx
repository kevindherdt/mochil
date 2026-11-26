// tests/ui/async.test.tsx — the one primitive every screen renders through, so its four states are pinned here
// rather than rediscovered per screen.
//
// The case that mattered: TanStack Query keeps the last successful `data` and still reports `isError` when a
// *refetch* fails. <Async> answered every `isError` with the full error panel, so one dropped background request
// replaced a list the person was reading — and the scroll position and edits inside it — with "Something went
// wrong." and a Retry button. Data that loaded once and failed to refresh stays on screen now.
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Async } from "../../src/ui/Async";

afterEach(cleanup);

const books = ["The Bellwether Line", "A Quiet Harbor"];

describe("Async", () => {
  it("renders the children once data is there", () => {
    render(
      <Async query={{ data: books, isLoading: false, isError: false }}>
        {(list) => <p>{list.join(", ")}</p>}
      </Async>,
    );
    expect(screen.getByText("The Bellwether Line, A Quiet Harbor")).toBeDefined();
    expect(screen.queryByText("Something went wrong.")).toBeNull();
  });

  it("shows the error panel when the first load fails, because there is nothing else to show", () => {
    render(
      <Async query={{ data: undefined, isLoading: false, isError: true }}>
        {() => <p>a list that must not appear</p>}
      </Async>,
    );
    expect(screen.getByText("Something went wrong.")).toBeDefined();
    expect(screen.queryByText("a list that must not appear")).toBeNull();
  });

  it("keeps the data and offers a retry when only the refetch failed", () => {
    const refetch = vi.fn();
    render(
      <Async query={{ data: books, isLoading: false, isError: true, refetch }}>
        {(list) => <p>{list.join(", ")}</p>}
      </Async>,
    );
    expect(screen.getByText("The Bellwether Line, A Quiet Harbor")).toBeDefined();
    expect(screen.getByRole("status").textContent).toContain("Couldn't refresh");
    expect(screen.queryByText("Something went wrong.")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("takes the error panel instead when the screen cannot act on stale data", () => {
    render(
      <Async query={{ data: books, isLoading: false, isError: true }} failOnRefetchError>
        {() => <p>a list that must not appear</p>}
      </Async>,
    );
    expect(screen.getByText("Something went wrong.")).toBeDefined();
    expect(screen.queryByText("a list that must not appear")).toBeNull();
  });

  it("shows the empty state for loaded-but-empty data, and keeps the retry strip above it", () => {
    render(
      <Async query={{ data: [], isLoading: false, isError: true }} empty={(list) => list.length === 0} emptyMessage="Nothing here yet.">
        {() => <p>a list that must not appear</p>}
      </Async>,
    );
    expect(screen.getByText("Nothing here yet.")).toBeDefined();
    // The emptiness could itself be out of date, so the strip stays — the empty message alone would be a claim.
    expect(screen.getByRole("status")).toBeDefined();
  });

  it("waits rather than erroring for a query that has not started, and for one still loading", () => {
    const waiting = render(
      <Async query={{ data: undefined, isLoading: false, isError: false }}>
        {() => <p>a list that must not appear</p>}
      </Async>,
    );
    expect(waiting.container.querySelector('[aria-hidden="true"]')).not.toBeNull();
    cleanup();
    const loading = render(
      <Async query={{ data: undefined, isLoading: true, isError: false }}>
        {() => <p>a list that must not appear</p>}
      </Async>,
    );
    expect(loading.container.querySelector('[aria-hidden="true"]')).not.toBeNull();
    expect(screen.queryByText("Something went wrong.")).toBeNull();
  });
});
