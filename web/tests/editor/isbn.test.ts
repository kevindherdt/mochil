import { describe, expect, it } from "vitest";
import { isbnOf, isbnState, type EditorBook } from "../../src/features/editor/api/editor";

/** The smallest book shape the comparison reads — everything else in EditorBook is irrelevant here. */
function book(identifiers: string, isbnFound: string): EditorBook {
  return { identifiers, isbnFound } as EditorBook;
}

// "Found ISBN" is the number printed inside the file; the comparison against the number already in the metadata is the
// whole reason a book that already has an ISBN can be read at all. These pin the four cases, because getting "same"
// wrong turns every hyphenated ISBN into a false difference.
describe("isbnState", () => {
  it("says nothing when the book was never read", () => {
    expect(isbnState(book("isbn:9780441172719", ""))).toBe("none");
  });

  it("offers a number when the book has none — the only case the bulk fill acts on", () => {
    expect(isbnState(book("", "9780306406157"))).toBe("missing");
  });

  it("agrees across hyphens, spacing and case", () => {
    expect(isbnState(book("isbn:978-0-306-40615-7", "9780306406157"))).toBe("same");
    expect(isbnState(book("isbn:9780306406157", "9780306406157"))).toBe("same");
  });

  it("notices a different number, which is how a wrong or duplicated ISBN looks", () => {
    expect(isbnState(book("isbn:9781111111111", "9780306406157"))).toBe("differs");
  });

  it("ignores other identifier schemes when comparing", () => {
    expect(isbnOf(book("amazon:1945054484, isbn:9781945054488", ""))).toBe("9781945054488");
  });
});
