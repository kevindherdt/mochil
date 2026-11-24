import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { ReviewPanel } from "../../src/features/editor/ReviewPanel";
import type { EditorBook, EditorChange } from "../../src/features/editor/api/editor";

/** Only the fields the panel reads. The rest of EditorBook is irrelevant to a preview. */
function book(overrides: Partial<EditorBook>): EditorBook {
  return { id: "a1", title: "EIGHTY-SIX Volume 02", ...overrides } as EditorBook;
}

// Renders share a document unless each one is unmounted: without this, "Apply 2 changes" exists three times over and
// every query that is not the first fails as ambiguous.
afterEach(cleanup);

const noop = () => {};
const props = (changes: EditorChange[], books: EditorBook[]) => ({
  changes,
  books,
  busy: false,
  onApply: noop,
  onCancel: noop,
  onApplyBook: noop,
  onDiscardBook: noop,
  onDiscardAll: noop,
});

const changes: EditorChange[] = [
  { book: "a1", title: "EIGHTY-SIX Volume 02", field: "identifiers", from: "isbn:9781975303144", to: "isbn:9781975303144, hardcover:974285" },
  { book: "a1", title: "EIGHTY-SIX Volume 02", field: "description", from: "", to: "A collective sigh of relief is exhaled as Lena." },
];

describe("ReviewPanel", () => {
  it("groups changes by book and counts them per book", () => {
    render(<ReviewPanel {...props(changes, [book({})])} />);
    expect(screen.getByText(/2 changes in 1 book/)).toBeTruthy();
    expect(screen.getByText("EIGHTY-SIX Volume 02")).toBeTruthy();
    expect(screen.getByText("2 fields")).toBeTruthy();
  });

  it("names the Hardcover match, because hardcover:974285 says nothing about which edition it is", () => {
    const matched = book({
      match: { title: "86—Eighty Six, Vol. 2", score: 0.92 } as EditorBook["match"],
      matchClear: true,
    });
    render(<ReviewPanel {...props(changes, [matched])} />);
    expect(screen.getByText(/86—Eighty Six, Vol\. 2/)).toBeTruthy();
    expect(screen.getByText(/92%/)).toBeTruthy();
  });

  it("marks only the identifier that this change adds", () => {
    render(<ReviewPanel {...props(changes, [book({})])} />);
    const added = screen.getByText("hardcover:974285");
    const kept = screen.getByText("isbn:9781975303144");
    expect(added.getAttribute("title")).toBe("Added by this change");
    expect(kept.getAttribute("title")).toBe("Already there");
  });

  it("says the description was empty rather than printing the whole paragraph", () => {
    render(<ReviewPanel {...props(changes, [book({})])} />);
    expect(screen.getByText("was empty")).toBeTruthy();
  });

  it("offers the rest of a long description instead of flooding the panel", () => {
    const long: EditorChange[] = [
      { book: "a1", title: "EIGHTY-SIX Volume 02", field: "description", from: "", to: "x".repeat(400) },
    ];
    render(<ReviewPanel {...props(long, [book({})])} />);
    expect(screen.getByText(/Show all 400 characters/)).toBeTruthy();
  });

  it("says how many changes Apply will make", () => {
    render(<ReviewPanel {...props(changes, [book({})])} />);
    expect(screen.getByRole("button", { name: "Apply 2 changes" })).toBeTruthy();
  });
});

describe("ReviewPanel — values that mean 'not set'", () => {
  it("does not print Go's zero time as a year", () => {
    const zeroTime: EditorChange[] = [
      { book: "a1", title: "The Hobbit", field: "published", from: "0101-01-01T00:00:00+00:00", to: "1937" },
    ];
    render(<ReviewPanel {...props(zeroTime, [book({ title: "The Hobbit" })])} />);
    expect(screen.getByText("(not set)")).toBeTruthy();
    expect(screen.queryByText(/0101-01-01/)).toBeNull();
    expect(screen.getByText("1937")).toBeTruthy();
  });

  it("keeps the actions above the list, where they cannot scroll away", () => {
    render(<ReviewPanel {...props(changes, [book({})])} />);
    expect(screen.getAllByRole("button", { name: "Apply 2 changes" })).toHaveLength(1);
  });
});

describe("ReviewPanel — per-book actions", () => {
  it("offers Apply and Discard for each book, so a batch can be taken one book at a time", () => {
    const applyBook = vi.fn();
    const discardBook = vi.fn();
    const three: EditorChange[] = [
      { book: "a1", title: "One", field: "publisher", from: "", to: "Ace" },
      { book: "b2", title: "Two", field: "publisher", from: "", to: "Roc" },
    ];
    render(
      <ReviewPanel
        {...props(three, [book({ id: "a1", title: "One" }), book({ id: "b2", title: "Two" })])}
        onApplyBook={applyBook}
        onDiscardBook={discardBook}
      />,
    );
    const buttons = screen.getAllByRole("button", { name: "Apply this book" });
    expect(buttons).toHaveLength(2);
    buttons[1]?.click();
    expect(applyBook).toHaveBeenCalledWith("b2");
    screen.getAllByRole("button", { name: "Discard" })[0]?.click();
    expect(discardBook).toHaveBeenCalledWith("a1");
  });

  it("offers Discard all, because a batch of 38 is often none of them", () => {
    const discardAll = vi.fn();
    const three: EditorChange[] = [
      { book: "a1", title: "One", field: "publisher", from: "", to: "Ace" },
      { book: "b2", title: "Two", field: "publisher", from: "", to: "Roc" },
    ];
    render(<ReviewPanel {...props(three, [book({ id: "a1", title: "One" }), book({ id: "b2", title: "Two" })])} onDiscardAll={discardAll} />);
    screen.getByRole("button", { name: "Discard all" }).click();
    expect(discardAll).toHaveBeenCalledTimes(1);
  });
});
