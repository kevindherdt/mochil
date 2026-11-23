// features/library/kinds.test.ts — the card-family split is the one rule that decides which component a shelf is
// drawn with, so it is pinned rather than inferred from a screenshot: a mis-mapped kind silently draws music with
// a 2:3 cover (or a movie with square art).
import { describe, expect, it } from "vitest";
import { cardFamily, isBookFamily, isShelvable, kindMeta, unbuiltReason } from "../../src/features/library/kinds";

describe("cardFamily", () => {
  it("maps every shelf kind this server can report", () => {
    // The real kinds from the dev box's config.json, not a guessed list.
    expect(cardFamily("books")).toBe("book");
    expect(cardFamily("audiobooks")).toBe("book");
    expect(cardFamily("storyteller")).toBe("none"); // Storyteller is a *library* whose kind is "books"
    expect(cardFamily("manga")).toBe("book");
    expect(cardFamily("documents")).toBe("book");
    expect(cardFamily("movies")).toBe("video");
    expect(cardFamily("tv")).toBe("video");
    expect(cardFamily("anime")).toBe("video");
    expect(cardFamily("music")).toBe("music");
    expect(cardFamily("podcasts")).toBe("music");
    expect(cardFamily("vault")).toBe("none");
    expect(cardFamily("wiki")).toBe("none");
  });

  it("is case-insensitive, because a kind arrives from config, not from code", () => {
    expect(cardFamily("MOVIES")).toBe("video");
    expect(cardFamily("Music")).toBe("music");
  });

  it("treats unknown kinds as unbrowsable rather than guessing a card", () => {
    expect(cardFamily("holograms")).toBe("none");
    expect(isShelvable("holograms")).toBe(false);
  });

  it("keeps isBookFamily about the book card only", () => {
    expect(isBookFamily("audiobooks")).toBe(true);
    expect(isBookFamily("movies")).toBe(false);
    expect(isBookFamily("music")).toBe(false);
  });
});

describe("kindMeta / unbuiltReason", () => {
  it("gives each family a distinct glyph", () => {
    expect(kindMeta("books").icon).not.toBe(kindMeta("movies").icon);
    expect(kindMeta("movies").icon).toBe(kindMeta("tv").icon);
    expect(kindMeta("music").icon).not.toBe(kindMeta("books").icon);
  });

  it("only the notes vault is still described as unbuilt", () => {
    expect(unbuiltReason("vault")).toContain("notes vault");
    expect(unbuiltReason("music")).not.toContain("next");
    expect(unbuiltReason("movies")).not.toContain("next");
  });
});
