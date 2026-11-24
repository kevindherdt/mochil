import { describe, expect, it } from "vitest";
import { speechChunks } from "../../src/features/lyana/api/speech";

// The server takes 16 kB of text per request and starts speaking as soon as the first piece arrives, so a long answer is
// cut at a sentence end rather than mid-word. These pin the cutting, which is the part that is easy to get subtly wrong.
describe("speechChunks", () => {
  it("leaves a short message whole", () => {
    expect(speechChunks("Hello there.")).toEqual(["Hello there."]);
  });

  it("returns nothing for blank text, so the speaker does not fire an empty request", () => {
    expect(speechChunks("   \n  ")).toEqual([]);
  });

  it("cuts a long answer at a sentence end, not through a word", () => {
    const sentence = "This is a sentence about a library that holds rather a lot of books. ";
    const chunks = speechChunks(sentence.repeat(40));
    expect(chunks.length).toBeGreaterThan(1);
    for (const chunk of chunks) {
      expect(chunk.length).toBeLessThanOrEqual(1200);
      expect(chunk.endsWith(".")).toBe(true);
    }
    expect(chunks.join(" ").replace(/\s+/g, " ")).toBe(sentence.repeat(40).trim().replace(/\s+/g, " "));
  });

  it("still cuts when there is no sentence end to cut at", () => {
    const chunks = speechChunks("a".repeat(3000));
    expect(chunks.length).toBeGreaterThan(1);
    for (const chunk of chunks) expect(chunk.length).toBeLessThanOrEqual(1200);
  });
});
