// tests/notes/tree.test.ts — the vault tree's sorting and shape, which is the whole reason NoteTree exists
// (Kevin: the flat list of always-open folder sections was not scannable). Sorting is where a notes list quietly
// goes wrong, so it gets a test rather than a hope: numeric-aware date folders, case-insensitive names, and
// folders before notes at every level.
import { describe, expect, it } from "vitest";
import { buildTree, countNotes, noteName } from "../../src/features/notes/tree";
import type { VaultPage } from "../../src/features/notes/api/vault";

const page = (path: string): VaultPage => ({
  id: path,
  library_id: "wiki",
  path,
  title: noteName({ path } as VaultPage),
  format: "md",
  size: 10,
  modified: "2026-11-01T00:00:00Z",
  revision: "r",
  text_url: `/api/documents/wiki/${path}/text`,
});

describe("buildTree", () => {
  it("nests folders and puts notes in their own folder, not the root", () => {
    const tree = buildTree(["Agent", "Agent/default"], [page("Agent/default/map.md"), page("HOME.md")]);
    expect(tree.notes.map((note) => note.path)).toEqual(["HOME.md"]);
    expect(tree.folders.map((folder) => folder.name)).toEqual(["Agent"]);
    expect(tree.folders[0]?.folders.map((folder) => folder.name)).toEqual(["default"]);
    expect(tree.folders[0]?.folders[0]?.notes.map((note) => note.path)).toEqual(["Agent/default/map.md"]);
  });

  it("creates missing parent folders instead of dropping a note", () => {
    // The tree listing always sends every folder, but a note must never vanish if one is missing.
    const tree = buildTree([], [page("Deep/Nested/note.md")]);
    expect(tree.folders[0]?.name).toBe("Deep");
    expect(tree.folders[0]?.folders[0]?.name).toBe("Nested");
    expect(tree.folders[0]?.folders[0]?.notes).toHaveLength(1);
  });

  it("sorts folders and notes numerically and case-insensitively", () => {
    const tree = buildTree(
      ["Projects", "personal"],
      [page("Projects/2026-11-10.md"), page("Projects/2026-11-7.md"), page("Projects/alpha.md"), page("Projects/Beta.md")],
    );
    expect(tree.folders.map((folder) => folder.name)).toEqual(["personal", "Projects"]);
    expect(tree.folders[1]?.notes.map((note) => note.path.split("/")[1])).toEqual([
      "2026-11-7.md",
      "2026-11-10.md",
      "alpha.md",
      "Beta.md",
    ]);
  });

  it("counts notes in sub-folders, which is what a collapsed row shows", () => {
    const tree = buildTree(
      ["Agent", "Agent/default"],
      [page("Agent/a.md"), page("Agent/default/b.md"), page("Agent/default/c.md"), page("root.md")],
    );
    expect(countNotes(tree)).toBe(4);
    expect(countNotes(tree.folders[0] as never)).toBe(3);
  });
});
