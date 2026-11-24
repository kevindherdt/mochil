import { describe, expect, it } from "vitest";
import { isbnTone, issueTone, scoreTone, toneWorst } from "../../src/features/editor/tone";

// The colours are the point: a table read at a glance has to say "look here" without being read. These pin the bands,
// because a threshold that drifts quietly turns a library of red rows into one nobody looks at.
describe("scoreTone", () => {
  it("is green from 90%, amber from 50%, red below", () => {
    expect(scoreTone(1)).toBe("good");
    expect(scoreTone(0.9)).toBe("good");
    expect(scoreTone(0.89)).toBe("fair");
    expect(scoreTone(0.5)).toBe("fair");
    expect(scoreTone(0.49)).toBe("poor");
    expect(scoreTone(0)).toBe("poor");
  });

  it("says nothing about a book with no match", () => {
    expect(scoreTone(undefined)).toBe("quiet");
    expect(scoreTone(Number.NaN)).toBe("quiet");
  });
});

describe("issueTone", () => {
  it("marks a disagreement as the thing to fix", () => {
    expect(issueTone("isbn_differs")).toBe("poor");
    expect(issueTone("duplicate")).toBe("poor");
  });

  it("marks an empty or unsettled field as worth a look, not an error", () => {
    expect(issueTone("no_description")).toBe("fair");
    expect(issueTone("match_unsure")).toBe("fair");
  });

  it("leaves a fact about the book uncoloured: a badge on every row is not a warning", () => {
    expect(issueTone("side_story")).toBe("quiet");
    expect(issueTone("something new the server invented")).toBe("quiet");
  });
});

describe("isbnTone", () => {
  it("separates agreement from a mismatch", () => {
    expect(isbnTone("same")).toBe("good");
    expect(isbnTone("differs")).toBe("poor");
    expect(isbnTone("missing")).toBe("fair");
    expect(isbnTone("none")).toBe("quiet");
  });
});

describe("toneWorst", () => {
  it("reports the worst of a set, so a hidden red badge colours its count", () => {
    expect(toneWorst(["quiet", "good", "poor"])).toBe("poor");
    expect(toneWorst(["quiet", "good"])).toBe("good");
    expect(toneWorst([])).toBe("quiet");
  });
});
