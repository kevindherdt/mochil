import { describe, expect, it } from "vitest";
import { matchReplace, matchReplaces, type EditorBook, type EditorField } from "../../src/features/editor/api/editor";

const ALL: readonly EditorField[] = [
  "title",
  "authors",
  "series",
  "series_index",
  "publisher",
  "published",
  "language",
  "tags",
  "identifiers",
  "description",
  "title_sort",
  "author_sort",
];

/** A clear Hardcover match, as the editor row carries it — the panel only ever sees this shape. */
function matched(overrides: Partial<EditorBook>, match: Partial<NonNullable<EditorBook["match"]>>): EditorBook {
  return {
    id: "a1",
    title: "",
    authors: [],
    series: "",
    // The link a match would write, already present: these tests are about the fields beside it.
    identifiers: "isbn:9780316497469, hardcover:89809",
    description: "",
    published: "",
    matchClear: true,
    ...overrides,
    match: {
      title: "",
      authors: [],
      year: 0,
      series: "",
      seriesIndex: undefined,
      overview: "",
      provider: "Hardcover",
      providerId: "89809",
      score: 0.95,
      ...match,
    },
  } as EditorBook;
}

// "Replace from the match" exists for the values filling never touches: a title that is not empty but is wrong, and a
// series spelled one way here and another on Hardcover. These pin what it rewrites and — just as important — what it
// refuses to, because a mode that quietly rewrites a whole library has to be boring and exact.
describe("matchReplace", () => {
  it("fixes the title that has an edition marker stuck on it", () => {
    const book = matched({ title: "The Last Wish (Polish)" }, { title: "The Last Wish" });
    expect(matchReplace(book, ALL)).toEqual({ title: "The Last Wish" });
  });

  it("renames a series the book gets wrong, and takes the position with it", () => {
    const book = matched({ series: "Wiedźmin", seriesIndex: undefined }, { series: "The Witcher", seriesIndex: 0.5 });
    expect(matchReplace(book, ALL)).toEqual({ series: "The Witcher", series_index: "0.5" });
  });

  it("leaves punctuation, case and accents alone: those are not what a person is fixing", () => {
    const book = matched(
      { title: "Dune: Messiah", authors: ["F. Herbert"], series: "Dune" },
      { title: "Dune Messiah", authors: ["F Herbert"], series: "DUNE" },
    );
    expect(matchReplace(book, ALL)).toEqual({});
  });

  it("takes the match's author order, which is a real difference even when the names themselves agree", () => {
    const book = matched({ authors: ["Herbert, Frank"] }, { authors: ["Frank Herbert"] });
    expect(matchReplace(book, ALL)).toEqual({ authors: "Frank Herbert" });
  });

  it("never blanks a field the match does not know", () => {
    const book = matched({ title: "Wrong", series: "Dune" }, { title: "Right", series: "" });
    expect(matchReplace(book, ALL)).toEqual({ title: "Right" });
  });

  it("does nothing for a match that is not clear", () => {
    const book = { ...matched({ title: "Wrong" }, { title: "Right" }), matchClear: false };
    expect(matchReplace(book, ALL)).toEqual({});
  });

  it("does nothing where the library keeps its own names, which is every Calibre library", () => {
    // Calibre owns title, authors, series, position and identifiers. What is left — year, description — is writable
    // there, but Replace is not offered on a Calibre library at all, so this is the belt to the UI's braces.
    const book = matched({ title: "The Last Wish (Polish)" }, { title: "The Last Wish", series: "The Witcher" });
    const calibre = ALL.filter((field) => !["title", "authors", "series", "series_index", "identifiers"].includes(field));
    expect(matchReplace(book, calibre)).toEqual({});
  });

  it("will not write a zero position — Hardcover sends 0 for 'not placed'", () => {
    const book = matched({ series: "Dune" }, { series: "Dune Rising", seriesIndex: 0 });
    expect(matchReplace(book, ALL)).toEqual({ series: "Dune Rising" });
  });

  it("keeps filling a book's empty title, which is the overlap with fill mode", () => {
    const book = matched({ title: "" }, { title: "The Last Wish" });
    expect(matchReplace(book, ALL)).toEqual({ title: "The Last Wish" });
  });

  it("replaces the year and the blurb too: a match that says 1993 is saying the 1990 is wrong", () => {
    const book = matched({ description: "Old blurb", published: "1990-01-01T00:00:00+00:00" }, { overview: "New blurb", year: 1993 });
    expect(matchReplace(book, ALL)).toEqual({ published: "1993", description: "New blurb" });
  });

  it("compares the blurb exactly, because punctuation inside prose is not a cosmetic difference", () => {
    const blurb = "Spice & sand.";
    const book = matched({ description: blurb }, { overview: blurb });
    expect(matchReplace(book, ALL)).toEqual({});
    const punctuated = matched({ description: "Spice and sand" }, { overview: blurb });
    expect(matchReplace(punctuated, ALL)).toEqual({ description: blurb });
  });

  it("writes the hardcover link, and leaves the ISBN alone", () => {
    const book = matched({ identifiers: "isbn:9780316497469, amazon:0316497468" }, { providerId: "89809" });
    expect(matchReplace(book, ALL)).toEqual({ identifiers: "isbn:9780316497469, amazon:0316497468, hardcover:89809" });
  });

  it("replaces a hardcover link that points somewhere else, without duplicating the scheme", () => {
    const book = matched({ identifiers: "isbn:9780316497469, hardcover:111" }, { providerId: "89809" });
    expect(matchReplace(book, ALL)).toEqual({ identifiers: "isbn:9780316497469, hardcover:89809" });
  });
});

describe("matchReplaces", () => {
  it("stages only the books whose names the match disagrees with", () => {
    const wrong = matched({ id: "a1", title: "The Last Wish (Polish)" }, { title: "The Last Wish" });
    const right = matched({ id: "b2", title: "Sword of Destiny" }, { title: "Sword of Destiny" });
    expect(matchReplaces([wrong, right], ALL)).toEqual({ a1: { title: "The Last Wish" } });
  });

  it("skips books with no match at all", () => {
    expect(matchReplaces([{ id: "a1", title: "x" } as EditorBook], ALL)).toEqual({});
  });
});
