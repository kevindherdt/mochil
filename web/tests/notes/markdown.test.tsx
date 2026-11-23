// tests/notes/markdown.test.tsx — the note renderer. Two things here are not cosmetic:
//   1. it TERMINATES. The first version shared one module-level /g regex across a recursive inline scan, which
//      hung the tab on a 6 kB note (the outer scan resumed from the inner call's lastIndex). These tests fail
//      loudly — vitest's per-test timeout — if that shape ever comes back.
//   2. it renders Obsidian's own syntax (`![[image.png]]` with a size, `[[Note]]`) instead of leaving it as text,
//      which is the whole reason a plain markdown renderer was not enough for a real vault.
import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { Markdown, type MarkdownContext } from "../../src/features/notes/markdown";
import type { VaultEntry } from "../../src/features/notes/api/vault";

const context = (entries: VaultEntry[] = [], onOpenNote = () => {}): MarkdownContext => ({ noteFolder: "Personal", entries, onOpenNote });

const render = (source: string, ctx: MarkdownContext = context()) => renderToStaticMarkup(<Markdown source={source} context={ctx} />);

describe("Markdown", () => {
  it("renders the basics and terminates on nested emphasis", () => {
    const html = render("# Title\n\nSome **bold with *italic* inside** text.\n\n- one\n- two\n");
    expect(html).toContain("Title");
    expect(html).toContain("<strong>");
    expect(html).toContain("<em>");
    expect(html).toContain("<li");
  });

  it("does not hang on a long note with repeated emphasis", () => {
    // 400 lines of mixed inline syntax: the shape that hung the first cut.
    const source = Array.from({ length: 400 }, (_, index) => `Line ${index} with **bold** and *it* and [[Note]].`).join("\n\n");
    const markup = render(source);
    expect(markup).toContain("Line 399");
  });

  it("drops frontmatter instead of rendering it as prose", () => {
    const html = render("---\ntags: [a, b]\n---\n\nBody text.\n");
    expect(html).toContain("Body text.");
    expect(html).not.toContain("tags:");
  });

  it("resolves an Obsidian image embed, including its size", () => {
    const entries: VaultEntry[] = [
      { id: "1", library_id: "wiki", path: "Personal/photo.png", title: "photo", format: "png", size: 10, modified: "", revision: "r", open_url: "/documents/wiki/1/photo.png" },
    ];
    const html = render("![[photo.png|220]]", context(entries));
    expect(html).toContain('src="/documents/wiki/1/photo.png"');
    expect(html).toContain("width:220px");
  });

  it("resolves a wikilink to a note in the vault and leaves a dangling one visibly dangling", () => {
    const entries: VaultEntry[] = [
      { id: "2", library_id: "wiki", path: "Agent/map.md", title: "map", format: "md", size: 10, modified: "", revision: "r", text_url: "/api/documents/wiki/2/text" },
    ];
    const html = render("[[map]] and [[missing note]]", context(entries));
    expect(html).toContain("<button");
    expect(html).toContain("map");
    expect(html).toContain("No note with that name");
    expect(html).toContain("missing note");
  });

  it("renders task lists as checkboxes", () => {
    const html = render("- [x] done\n- [ ] todo\n");
    expect(html).toContain('type="checkbox"');
    expect(html).toContain("checked");
  });
});
