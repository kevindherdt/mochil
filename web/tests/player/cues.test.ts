import { describe, expect, it } from "vitest";
import { cueAt } from "../../src/features/library/api/subtitleImages";

const box = { x: 0, y: 0, w: 0, h: 0 };
const cues = [
  { start: 10, end: 12, file: "c1.png", ...box },
  { start: 20, end: 23, file: "c2.png", ...box },
  { start: 30, end: 30.5, file: "c3.png", ...box },
];

describe("cueAt", () => {
  it("finds the cue covering a position", () => {
    expect(cueAt(cues, 21)?.file).toBe("c2.png");
  });

  it("returns nothing in a gap or outside the track", () => {
    expect(cueAt(cues, 15)).toBeNull();
    expect(cueAt(cues, 0)).toBeNull();
    expect(cueAt(cues, 99)).toBeNull();
  });

  it("treats the bounds as inclusive", () => {
    expect(cueAt(cues, 10)?.file).toBe("c1.png");
    expect(cueAt(cues, 12)?.file).toBe("c1.png");
    expect(cueAt(cues, 30.5)?.file).toBe("c3.png");
  });

  it("handles an empty track", () => {
    expect(cueAt([], 5)).toBeNull();
  });
});
