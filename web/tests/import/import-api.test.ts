// features/import/api/import.test.ts — the drop folder's contract.
//
// Two things here are easy to get silently wrong and impossible to notice in a screenshot: which filter id maps to which
// `?status=` value (an empty one means "everything", not "no status"), and how a row's status reads to a person when the
// server sends one this app has never seen.
import { describe, expect, it } from "vitest";
import { IMPORT_FILTERS, NAMING_PRESETS, PUBLICATION_KINDS, STATUS_TEXT, importRow } from "../../src/features/import/api/import";

describe("IMPORT_FILTERS", () => {
  it("sends an empty status for Everything, because that is what the server treats as unfiltered", () => {
    expect(IMPORT_FILTERS.find((entry) => entry.id === "all")?.statuses).toBe("");
  });

  it("asks for new books and new volumes together in the default filter", () => {
    expect(IMPORT_FILTERS[0]?.id).toBe("fresh");
    expect(IMPORT_FILTERS[0]?.statuses).toBe("new,new_volume");
  });

  it("has a status for every filter the screen can show", () => {
    for (const filter of IMPORT_FILTERS) expect(filter.statuses).toBeTypeOf("string");
  });
});

describe("a row", () => {
  it("fills in what the server leaves out, so a screen never renders undefined", () => {
    const row = importRow.parse({ source_path: "drop/one.epub", status: "new" });
    expect(row.proposal).toBeUndefined();
    expect(row.copied).toBe(false);
    expect(row.destination).toBe("");
  });

  it("keeps a status this app has never seen, so nothing disappears from the list", () => {
    const row = importRow.parse({ source_path: "drop/two.epub", status: "quarantined" });
    expect(STATUS_TEXT[row.status] ?? row.status).toBe("quarantined");
  });
});

describe("the naming presets", () => {
  it("are the three the server accepts, and nothing invented", () => {
    // internal/ingest/naming.go's ValidNamingPreset: default, calibre, kavita. "folder" was offered for a while and every
    // request carrying it came back 400, which blanked the list instead of renaming anything.
    expect(NAMING_PRESETS.map((entry) => entry.id).sort()).toEqual(["calibre", "default", "kavita"]);
  });
});

describe("the shelves Import offers", () => {
  it("match the kinds the server treats as publications", () => {
    // internal/web's publicationKind: anything else makes the import endpoint answer 404.
    expect([...PUBLICATION_KINDS].sort()).toEqual(["books", "comics", "manga", "mixed"]);
  });
});
