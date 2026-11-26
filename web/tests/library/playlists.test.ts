// tests/library/playlists.test.ts — the playlist client against the server's real contract.
//
// The API is revision-guarded, so the interesting failures are not "did it parse" but "did it quote the revision it
// read, and did a 412 arrive as a conflict instead of a generic error". Those are the two things a screen cannot see
// for itself, and the two ways this could silently start overwriting another device's edit.
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  appendToPlaylist,
  createPlaylist,
  deletePlaylist,
  fetchPlaylist,
  fetchPlaylistSummaries,
  PlaylistConflictError,
  PlaylistNameError,
  PlaylistNameTakenError,
  renamePlaylist,
  replacePlaylist,
} from "../../src/features/library/api/playlists";

function json(body: unknown, headers: Record<string, string> = {}, status = 200): Response {
  const merged = { "Content-Type": "application/json", ...headers };
  // 204 must not carry a body — the playlists API answers its writes that way, with the new revision in a header.
  return new Response(status === 204 ? null : JSON.stringify(body), { status, headers: merged });
}

function stub(response: Response | (() => Response)) {
  const spy = vi.fn().mockImplementation(() => Promise.resolve(typeof response === "function" ? response() : response));
  vi.stubGlobal("fetch", spy);
  return spy;
}

function call(spy: ReturnType<typeof stub>, index = 0) {
  const [path, init] = spy.mock.calls[index] as [string, RequestInit];
  return { path, method: init.method, headers: init.headers as Record<string, string>, body: init.body === undefined ? undefined : JSON.parse(String(init.body)) };
}

const onePlaylist = {
  name: "Night drive",
  items: [
    { id: "t1", library_id: "music", title: "First", artist: "A", album: "X", artwork_url: "/a1.jpg", stream_url: "/s1", duration_seconds: 200 },
    { id: "t2", library_id: "music", title: "Second", artist: "B", album: "X" },
  ],
};

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("reading a playlist", () => {
  it("keeps the server's order and takes the ETag as the revision to write against", async () => {
    stub(json(onePlaylist, { ETag: "rev-7" }));
    const playlist = await fetchPlaylist("Night drive");
    expect(playlist.name).toBe("Night drive");
    expect(playlist.revision).toBe("rev-7");
    expect(playlist.tracks.map((track) => track.itemId)).toEqual(["t1", "t2"]);
  });

  it("falls back to the album when a track has no artist, so a row is never blank", async () => {
    stub(json({ name: "p", items: [{ id: "t2", library_id: "music", title: "Second", album: "X" }] }));
    const playlist = await fetchPlaylist("p");
    expect(playlist.tracks[0]?.artist).toBe("");
    expect(playlist.tracks[0]?.album).toBe("X");
  });

  it("encodes the name in the path — a playlist may be called \"100% dance\"", async () => {
    const spy = stub(json({ name: "100% dance", items: [] }));
    await fetchPlaylist("100% dance");
    expect(call(spy).path).toBe("/api/audio-playlists/100%25%20dance");
  });

  it("reads summaries from the versioned envelope and takes the first of the four covers", async () => {
    stub(json({ version: 1, playlists: [{ name: "Night", item_count: 12, artwork_urls: ["/c1.jpg", "/c2.jpg"] }] }));
    const summaries = await fetchPlaylistSummaries();
    expect(summaries).toEqual([{ name: "Night", count: 12, coverUrl: "/c1.jpg" }]);
  });
});

describe("writing a playlist", () => {
  it("reorders with one PUT of the whole list, quoting the revision it read", async () => {
    const spy = stub(json({}, { ETag: "rev-8" }, 204));
    const revision = await replacePlaylist(
      "Night drive",
      [
        { libraryId: "music", itemId: "t2" },
        { libraryId: "music", itemId: "t1" },
      ],
      "rev-7",
    );
    const sent = call(spy);
    expect(sent.method).toBe("PUT");
    expect(sent.headers["If-Match"]).toBe("rev-7");
    expect(sent.body).toEqual([
      { library_id: "music", item_id: "t2" },
      { library_id: "music", item_id: "t1" },
    ]);
    // The answered revision is what the *next* move must quote — otherwise two moves in a row conflict with themselves.
    expect(revision).toBe("rev-8");
  });

  it("sends no If-Match when it has no revision, rather than inventing one", async () => {
    const spy = stub(json({}, {}, 204));
    await replacePlaylist("fresh", [{ libraryId: "music", itemId: "t1" }], null);
    expect(call(spy).headers["If-Match"]).toBeUndefined();
  });

  it("renames with PATCH and the revision, and reports a taken name as a name error", async () => {
    const ok = stub(json({}, {}, 204));
    await renamePlaylist("Night drive", "Night ride", "rev-7");
    const sent = call(ok);
    expect(sent.method).toBe("PATCH");
    expect(sent.headers["If-Match"]).toBe("rev-7");
    expect(sent.body).toEqual({ name: "Night ride" });

    stub(new Response("playlist already exists", { status: 409 }));
    await expect(renamePlaylist("a", "b", "rev-7")).rejects.toBeInstanceOf(PlaylistNameError);
  });

  it("distinguishes a taken name from an unusable one, so \"create and add\" can carry on past a race", async () => {
    // Both are PlaylistNameError subclasses (a dialog reports either the same way), but only "taken" may be ignored
    // by the add-to-playlist flow. The split used to be a search for the word "exists" in the server's message.
    stub(new Response("playlist already exists", { status: 409 }));
    await expect(createPlaylist("Night drive")).rejects.toBeInstanceOf(PlaylistNameTakenError);

    stub(new Response("invalid playlist name", { status: 400 }));
    const unusable = await createPlaylist("").catch((error: unknown) => error);
    expect(unusable).toBeInstanceOf(PlaylistNameError);
    expect(unusable).not.toBeInstanceOf(PlaylistNameTakenError);
  });

  it("reports a 412 on delete as a conflict, not as \"not found\"", async () => {
    stub(new Response("playlist changed", { status: 412 }));
    await expect(deletePlaylist("Night drive", "rev-stale")).rejects.toBeInstanceOf(PlaylistConflictError);
  });

  it("appends one track by library and item id", async () => {
    const spy = stub(json({}, {}, 204));
    await appendToPlaylist("Night drive", { libraryId: "music", itemId: "t9" });
    const sent = call(spy);
    expect(sent.method).toBe("POST");
    expect(sent.body).toEqual({ library_id: "music", item_id: "t9" });
  });

  it("lets anything else through untouched, so a real server error keeps its own message", async () => {
    stub(new Response("boom", { status: 500 }));
    await expect(createPlaylist("x")).rejects.toMatchObject({ status: 500 });
  });
});
