// tests/import/copy-api.test.ts — the three-step copy, at the seam where a mistake is silent.
//
// The server refuses an execute whose confirmation is not the transaction id, and it refuses a plan of more than a hundred
// entries; both failures look like "the copy button did nothing", so the wire shape is pinned here rather than discovered in
// a browser.
import { afterEach, describe, expect, it, vi } from "vitest";
import { executeCopy, planCopy, previewCopy } from "../../src/features/import/api/import";

const reply = (body: unknown, status = 200) =>
  Promise.resolve(
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("previewCopy", () => {
  it("sends every entry and reports the refusals the server gave", async () => {
    const fetchMock = vi.fn(() =>
      reply({
        ready: false,
        conflicts: 1,
        entries: [
          { source_path: "incoming/a.epub", target_path: "A/a.epub", ready: true, issues: [] },
          {
            source_path: "incoming/b.epub",
            target_path: "A/b.epub",
            ready: false,
            issues: [{ code: "target_exists", message: "that name is taken" }],
          },
        ],
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const preview = await previewCopy([
      { source: "incoming/a.epub", target: "A/a.epub" },
      { source: "incoming/b.epub", target: "A/b.epub" },
    ]);

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toContain("/api/metadata/managed-ingest/preview");
    expect(init.method).toBe("POST");
    expect(JSON.parse(String(init.body))).toEqual({
      entries: [
        { source_path: "incoming/a.epub", target_path: "A/a.epub" },
        { source_path: "incoming/b.epub", target_path: "A/b.epub" },
      ],
    });
    expect(preview.entries.filter((entry) => !entry.ready)).toHaveLength(1);
    expect(preview.entries[1]?.issues[0]?.message).toBe("that name is taken");
  });
});

describe("plan and execute", () => {
  it("executes with the transaction's own id as the confirmation", async () => {
    const calls: { url: string; body: unknown }[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn((url: string, init: RequestInit) => {
        calls.push({ url, body: JSON.parse(String(init.body)) });
        if (url.includes("/plan")) return reply({ id: "tx-42", state: "planned", entries: [] }, 201);
        return reply({ id: "tx-42", state: "copying", entries: [] }, 201);
      }),
    );

    const planned = await planCopy([{ source: "incoming/a.epub", target: "A/a.epub" }]);
    expect(planned.id).toBe("tx-42");

    const running = await executeCopy(planned.id);
    expect(running.state).toBe("copying");
    expect(calls[1]?.url).toContain("/api/metadata/managed-ingest/tx-42/execute");
    // The server rejects a mismatched confirmation with a 400, which would read as "the button did nothing".
    expect(calls[1]?.body).toEqual({ confirmation: "tx-42" });
  });
});

describe("sidecars and embeds", () => {
  it("carry only the four fields the screen knows, and send series_index as a number", async () => {
    const calls: { url: string; body: unknown }[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn((url: string, init: RequestInit) => {
        calls.push({ url, body: JSON.parse(String(init.body)) });
        return reply({ results: [{ target_path: "A/a.epub", sidecar: "A/a.opf", status: "written" }] });
      }),
    );
    const { writeSidecars, embedMetadata } = await import("../../src/features/import/api/import");
    const entries = [
      { target: "A/a.epub", metadata: { title: "Dune", authors: ["Frank Herbert"], series: "Dune", seriesIndex: 2 } },
    ];
    const results = await writeSidecars(entries);
    await embedMetadata(entries);
    expect(results[0]?.status).toBe("written");
    expect(calls[0]?.url).toContain("/api/metadata/managed-ingest/sidecars");
    expect(calls[1]?.url).toContain("/api/metadata/managed-ingest/embed");
    // The server's own field names, and nothing invented: a tag list or publisher the screen never showed must not appear.
    expect(calls[0]?.body).toEqual({
      entries: [
        {
          target_path: "A/a.epub",
          metadata: { title: "Dune", authors: ["Frank Herbert"], series: "Dune", series_index: 2 },
        },
      ],
    });
  });
});
