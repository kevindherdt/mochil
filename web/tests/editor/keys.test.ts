import { describe, expect, it } from "vitest";
import { gridKey, type GridShape } from "../../src/features/editor/keys";

const shape: GridShape = { rows: 3, columns: 4, page: 2 };

describe("gridKey", () => {
  it("moves inside the grid and stops at its edges", () => {
    expect(gridKey("ArrowRight", { row: 0, col: 0 }, shape)).toEqual({ kind: "move", cursor: { row: 0, col: 1 } });
    expect(gridKey("ArrowLeft", { row: 0, col: 0 }, shape)).toEqual({ kind: "move", cursor: { row: 0, col: 0 } });
    expect(gridKey("ArrowDown", { row: 2, col: 0 }, shape)).toEqual({ kind: "move", cursor: { row: 2, col: 0 } });
    expect(gridKey("ArrowUp", { row: 1, col: 2 }, shape)).toEqual({ kind: "move", cursor: { row: 0, col: 2 } });
  });

  it("wraps Tab onto the next row and Shift+Tab onto the previous one", () => {
    expect(gridKey("Tab", { row: 0, col: 3 }, shape)).toEqual({ kind: "move", cursor: { row: 1, col: 0 } });
    expect(gridKey("Tab", { row: 1, col: 0 }, shape, { shift: true })).toEqual({ kind: "move", cursor: { row: 0, col: 3 } });
  });

  it("leaves the table at both ends instead of trapping the keyboard in it", () => {
    expect(gridKey("Tab", { row: 2, col: 3 }, shape)).toEqual({ kind: "leave" });
    expect(gridKey("Tab", { row: 0, col: 0 }, shape, { shift: true })).toEqual({ kind: "leave" });
    // One cell in from either end is still a move: only the true edges are exits.
    expect(gridKey("Tab", { row: 2, col: 2 }, shape)).toEqual({ kind: "move", cursor: { row: 2, col: 3 } });
  });

  it("Home and End go to the row's ends, with Ctrl to the table's corners", () => {
    expect(gridKey("Home", { row: 1, col: 2 }, shape)).toEqual({ kind: "move", cursor: { row: 1, col: 0 } });
    expect(gridKey("End", { row: 1, col: 2 }, shape)).toEqual({ kind: "move", cursor: { row: 1, col: 3 } });
    expect(gridKey("Home", { row: 1, col: 2 }, shape, { ctrl: true })).toEqual({ kind: "move", cursor: { row: 0, col: 0 } });
    expect(gridKey("End", { row: 1, col: 2 }, shape, { meta: true })).toEqual({ kind: "move", cursor: { row: 2, col: 3 } });
  });

  it("moves a page at a time without overshooting", () => {
    expect(gridKey("PageDown", { row: 0, col: 1 }, shape)).toEqual({ kind: "move", cursor: { row: 2, col: 1 } });
    expect(gridKey("PageUp", { row: 2, col: 1 }, shape)).toEqual({ kind: "move", cursor: { row: 0, col: 1 } });
    expect(gridKey("PageDown", { row: 0, col: 1 }, { ...shape, page: 0 })).toEqual({ kind: "move", cursor: { row: 1, col: 1 } });
  });

  it("selects everything on Ctrl+A and owns nothing else", () => {
    expect(gridKey("a", { row: 1, col: 1 }, shape, { ctrl: true })).toEqual({ kind: "select-all" });
    expect(gridKey("a", { row: 1, col: 1 }, shape)).toEqual({ kind: "none" });
    expect(gridKey("Enter", { row: 1, col: 1 }, shape)).toEqual({ kind: "none" });
    expect(gridKey(" ", { row: 1, col: 1 }, shape)).toEqual({ kind: "none" });
  });

  it("does nothing at all when there are no rows to move through", () => {
    expect(gridKey("ArrowDown", { row: 0, col: 0 }, { rows: 0, columns: 4, page: 2 })).toEqual({ kind: "none" });
    expect(gridKey("Tab", { row: 0, col: 0 }, { rows: 0, columns: 4, page: 2 })).toEqual({ kind: "none" });
  });
});
