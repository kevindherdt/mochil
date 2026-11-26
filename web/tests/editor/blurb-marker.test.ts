import { describe, expect, it } from "vitest";
import { matchFill, type EditorBook } from "../../src/features/editor/api/editor";

// Hardcover proposals sometimes begin with the volume they follow: the edition for volume 7 arrived as
// "(Volume 6)Accelerator has killed thousands of Sisters…". The blurb is the right book's, but a row that says
// "Volume 07" and a description that opens "(Volume 6)" reads as a mismatch.
function book(overrides: Partial<EditorBook> = {}): EditorBook {
  return {
    id: "b1",
    title: "A Certain Magical Index (Light Novel), Vol. 7",
    authors: ["Kazuma Kamachi"],
    series: "A Certain Magical Index (Light Novel)",
    seriesIndex: 7,
    format: "epub",
    tags: [],
    publisher: "",
    published: "",
    language: "",
    rating: 0,
    description: "",
    identifiers: "",
    titleSort: "",
    authorSort: "",
    path: "a.epub",
    edited: false,
    custom: {},
    collections: [],
    issues: [],
    isbnFound: "",
    isbnNote: "",
    match: undefined,
    matchClear: false,
    quality: 0,
    progress: 0,
    ...overrides,
  } as EditorBook;
}

function withMatch(overview: string): EditorBook {
  return book({
    matchClear: true,
    match: {
      title: "A Certain Magical Index (Light Novel), Vol. 7",
      authors: ["Kazuma Kamachi"],
      year: 2005,
      series: "A Certain Magical Index (Light Novel)",
      seriesIndex: 7,
      overview,
      providerId: "870241",
      provider: "hardcover",
      score: 1,
    },
  } as Partial<EditorBook>);
}

describe("the blurb a match fills in", () => {
  it("drops a recap marker that names another volume", () => {
    const staged = matchFill(
      withMatch("(Volume 6)Accelerator has killed thousands of Sisters--clones of Mikoto Misaka."),
      ["description"],
    );
    expect(staged.description).toBe("Accelerator has killed thousands of Sisters--clones of Mikoto Misaka.");
  });

  it("keeps a marker that names this book's own volume", () => {
    const staged = matchFill(withMatch("(Volume 7) The story continues here."), ["description"]);
    expect(staged.description).toBe("(Volume 7) The story continues here.");
  });

  it("leaves an ordinary blurb alone", () => {
    const staged = matchFill(withMatch("Touma Kamijou is the unluckiest boy in Academy City."), ["description"]);
    expect(staged.description).toBe("Touma Kamijou is the unluckiest boy in Academy City.");
  });

  it("never empties a blurb that is only the marker", () => {
    const staged = matchFill(withMatch("(Volume 6)"), ["description"]);
    expect(staged.description).toBe("(Volume 6)");
  });
});
