// tests/library/playlist-detail.test.tsx — the two behaviours the server's contract makes easy to get wrong.
//
// 1. A move must send ONE put of the whole list in the new order — not a delete plus an add, which would show up as
//    two revisions and could leave the playlist half-changed if the second failed.
// 2. The revision the server answers with must replace the one the screen read; otherwise the second move of a
//    session sends a stale ETag and the server's 412 makes the app look like it is fighting another device.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PlaylistDetail } from "../../src/features/library/PlaylistsView";

const playlist = {
  name: "Night drive",
  items: [
    { id: "t1", library_id: "music", title: "First", artist: "A", artwork_url: "/a1.jpg", stream_url: "/s1" },
    { id: "t2", library_id: "music", title: "Second", artist: "B", artwork_url: "/a2.jpg", stream_url: "/s2" },
  ],
};

function renderDetail() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PlaylistDetail name="Night drive" onBack={() => {}} onPlay={() => {}} />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  // No setup file registers RTL's auto-cleanup here, so without this the second test finds the first test's DOM and
  // every query matches twice.
  cleanup();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("PlaylistDetail", () => {
  it("moves a track with one whole-list PUT in the new order, quoting the revision it read", async () => {
    let revision = 0;
    const fetchSpy = vi.fn().mockImplementation((_path: string, init?: RequestInit) => {
      if ((init?.method ?? "GET") === "GET") {
        return Promise.resolve(new Response(JSON.stringify(playlist), { headers: { "Content-Type": "application/json", ETag: `rev-${revision}` } }));
      }
      revision += 1;
      return Promise.resolve(new Response(null, { status: 204, headers: { ETag: `rev-${revision}` } }));
    });
    vi.stubGlobal("fetch", fetchSpy);

    renderDetail();
    const down = await screen.findByLabelText("Move First down");
    fireEvent.click(down);

    await waitFor(() => expect(fetchSpy.mock.calls.filter(([, init]) => (init as RequestInit | undefined)?.method === "PUT")).toHaveLength(1));
    const [, init] = fetchSpy.mock.calls.find(([, init]) => (init as RequestInit | undefined)?.method === "PUT") as [string, RequestInit];
    expect((init.headers as Record<string, string>)["If-Match"]).toBe("rev-0");
    expect(JSON.parse(String(init.body))).toEqual([
      { library_id: "music", item_id: "t2" },
      { library_id: "music", item_id: "t1" },
    ]);
  });

  it("quotes the revision the server answered with for the next write, not the one it read", async () => {
    const seen: string[] = [];
    let revision = 0;
    const fetchSpy = vi.fn().mockImplementation((_path: string, init?: RequestInit) => {
      if ((init?.method ?? "GET") === "GET") {
        // The refetch after a write answers with the revision the write produced — exactly what the real server does.
        return Promise.resolve(new Response(JSON.stringify(playlist), { headers: { "Content-Type": "application/json", ETag: `rev-${revision}` } }));
      }
      const quoted = (init?.headers as Record<string, string>)["If-Match"] ?? "";
      seen.push(quoted);
      revision += 1;
      return Promise.resolve(new Response(null, { status: 204, headers: { ETag: `rev-${revision}` } }));
    });
    vi.stubGlobal("fetch", fetchSpy);

    renderDetail();
    fireEvent.click(await screen.findByLabelText("Move First down"));
    await waitFor(() => expect(seen).toHaveLength(1));
    // The list is back in its original order on every GET (this fake always answers the same list), so the second move
    // is the same click again — what matters is the revision it quotes.
    fireEvent.click(await screen.findByLabelText("Move First down"));
    await waitFor(() => expect(seen).toHaveLength(2));
    expect(seen).toEqual(["rev-0", "rev-1"]);
  });

  it("shows a conflict in words instead of silently retrying", async () => {
    const fetchSpy = vi.fn().mockImplementation((_path: string, init?: RequestInit) => {
      if ((init?.method ?? "GET") === "GET") {
        return Promise.resolve(new Response(JSON.stringify(playlist), { headers: { "Content-Type": "application/json", ETag: "rev-0" } }));
      }
      return Promise.resolve(new Response("playlist changed", { status: 412 }));
    });
    vi.stubGlobal("fetch", fetchSpy);

    renderDetail();
    fireEvent.click(await screen.findByLabelText("Remove First"));
    expect(await screen.findByText(/changed on another device/i)).toBeTruthy();
  });
});
