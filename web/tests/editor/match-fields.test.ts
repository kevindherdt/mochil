import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MATCH_WRITABLE, matchFieldsFor, setMatchFieldsFor, useMatchFields } from "../../src/features/editor/matchFields";

// A match carries a title, a series, a year and a blurb, and "take the match" used to mean all of them. Sometimes the
// blurb is the point and the series name is the thing that must not move. The choice now lives on the server with the
// library, so these tests stub the two endpoints rather than a browser store.

interface Call {
  url: string;
  init: RequestInit;
}

/** A stand-in for the match-fields endpoints that keeps what it was told, so a read after a write answers what the
 *  server would: a library that has saved nothing gets all seven. */
function server(options: { saved?: Record<string, string[]>; failWrites?: boolean } = {}) {
  const calls: Call[] = [];
  const saved: Record<string, string[]> = { ...(options.saved ?? {}) };
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input);
      calls.push({ url, init });
      if (init.method === "PUT") {
        if (options.failWrites === true) return new Response("no", { status: 500 });
        const body = JSON.parse(String(init.body)) as { library: string; fields: string[] };
        saved[body.library] = body.fields;
        return new Response(null, { status: 204 });
      }
      const library = new URL(url, "http://mochil.test").searchParams.get("library") ?? "";
      return new Response(JSON.stringify({ fields: saved[library] ?? [...MATCH_WRITABLE] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );
  return { calls, saved };
}

function mount(libraryId: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  // No JSX here: this file is `.ts`, and the wrapper is one element. Same provider the app mounts around the editor.
  const wrapper = ({ children }: { children: ReactNode }) => createElement(QueryClientProvider, { client }, children);
  return renderHook(() => useMatchFields(libraryId), { wrapper });
}

function puts(calls: Call[]) {
  return calls.filter((call) => call.init.method === "PUT");
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("which fields a match may write", () => {
  it("treats the answer as on its way, not as none, while it is still loading", () => {
    // A request that never lands: the boxes must still start at every field, never at zero, because an empty
    // choice would stage nothing at all and a match would look like it found nothing.
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => {})));
    const { result } = mount("shelf-loading");
    expect(result.current[0]).toEqual([...MATCH_WRITABLE]);
  });

  it("starts with every field when the server has nothing saved", async () => {
    const { calls } = server();
    const { result } = mount("shelf-default");
    await waitFor(() => expect(calls.length).toBeGreaterThan(0));
    expect(calls[0]?.url).toContain("/api/metadata/editor/match-fields?library=shelf-default");
    expect(result.current[0]).toEqual([...MATCH_WRITABLE]);
  });

  it("sends the whole list when a field is chosen and shows it before the server answers", async () => {
    const { calls } = server({ saved: { "shelf-save": ["title"] } });
    const { result } = mount("shelf-save");
    await waitFor(() => expect(result.current[0]).toEqual(["title"]));

    act(() => result.current[1](["title", "description"]));
    // Optimistic: the checkbox follows the click, not the round trip.
    expect(result.current[0]).toEqual(["title", "description"]);

    await waitFor(() => expect(puts(calls).length).toBe(1));
    const put = puts(calls)[0];
    expect(put?.url).toBe("/api/metadata/editor/match-fields");
    expect(JSON.parse(String(put?.init.body))).toEqual({ library: "shelf-save", fields: ["title", "description"] });
  });

  it("keeps the previous choice when the server refuses the write", async () => {
    const { calls } = server({ saved: { "shelf-refused": ["description"] }, failWrites: true });
    const { result } = mount("shelf-refused");
    await waitFor(() => expect(result.current[0]).toEqual(["description"]));

    act(() => result.current[1](["title"]));
    expect(result.current[0]).toEqual(["title"]);
    await waitFor(() => expect(puts(calls).length).toBe(1));
    // The write failed, so the box goes back: a tick the library does not hold would promise something the next
    // match would not do.
    await waitFor(() => expect(result.current[0]).toEqual(["description"]));
  });

  it("asks each library for its own choice", async () => {
    const { calls } = server({ saved: { "shelf-books": ["title"], "shelf-comics": ["description", "published"] } });
    const books = mount("shelf-books");
    const comics = mount("shelf-comics");
    await waitFor(() => expect(books.result.current[0]).toEqual(["title"]));
    await waitFor(() => expect(comics.result.current[0]).toEqual(["published", "description"]));
    const urls = calls.map((call) => call.url);
    expect(urls).toContain("/api/metadata/editor/match-fields?library=shelf-books");
    expect(urls).toContain("/api/metadata/editor/match-fields?library=shelf-comics");
  });

  it("answers with the same array until the choice changes", () => {
    // useSyncExternalStore compares by identity: a fresh copy on every read is an endless re-render, which is what
    // this guard exists for.
    server();
    const first = matchFieldsFor("shelf-stable");
    expect(matchFieldsFor("shelf-stable")).toBe(first);
    setMatchFieldsFor("shelf-stable", ["series"]);
    expect(matchFieldsFor("shelf-stable")).not.toBe(first);
    expect(matchFieldsFor("shelf-stable")).toEqual(["series"]);
  });
});
