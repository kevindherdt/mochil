import { afterEach, describe, expect, it, vi } from "vitest";
import { HttpError } from "../../src/lib/http";
import { writeMetadataFiles } from "../../src/features/editor/api/editor";

/** The server's answer to a write, in the shape internal/web/editor_metadata_files.go sends. */
function respond(body: unknown, init: ResponseInit = { status: 200 }): void {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(JSON.stringify(body), { ...init, headers: { "Content-Type": "application/json" } })),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

// Writing metadata.opf is the step that makes the editor's work leave Mochil, so what the panel reports has to match
// what the server actually did — a write that failed must never read as a write that happened.
describe("writeMetadataFiles", () => {
  it("asks for a sidecar and keeps the server's per-book outcomes", async () => {
    respond({
      results: [
        { id: "a1", title: "Dune", status: "written", sidecar: "Dune/Dune 1.opf" },
        { id: "a2", title: "Dune Messiah", status: "exists", message: "left alone" },
        { id: "n1", title: "Neuromancer", status: "failed", message: "the EPUB cannot be read" },
      ],
      summary: { written: 1, exists: 1, failed: 1 },
    });
    const outcome = await writeMetadataFiles("books", ["a1", "a2", "n1"], { embed: false, overwrite: false });
    expect(outcome.summary).toEqual({ written: 1, exists: 1, failed: 1 });
    expect(outcome.results[2]).toMatchObject({ id: "n1", status: "failed", message: "the EPUB cannot be read" });
    const call = (fetch as unknown as { mock: { calls: [string, RequestInit][] } }).mock.calls[0];
    expect(call?.[0]).toBe("/api/metadata/editor/metadata-files");
    expect(JSON.parse(String(call?.[1].body))).toMatchObject({ library: "books", ids: ["a1", "a2", "n1"], sidecar: true });
  });

  it("counts outcomes itself when the server sends no summary", async () => {
    respond({ results: [{ id: "a1", title: "Dune", status: "written" }, { id: "a2", title: "X", status: "exists" }] });
    expect((await writeMetadataFiles("books", ["a1", "a2"], { embed: true, overwrite: true })).summary).toEqual({
      written: 1,
      exists: 1,
      failed: 0,
    });
  });

  it("passes the server's refusal through, so a Calibre library says why instead of failing silently", async () => {
    respond("Calibre keeps its own metadata.opf", { status: 409 });
    const refused = writeMetadataFiles("books", ["a1"], { embed: false, overwrite: false });
    await expect(refused).rejects.toBeInstanceOf(HttpError);
    await expect(refused).rejects.toThrow(/Calibre/);
  });
});
