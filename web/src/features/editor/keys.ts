// keys.ts — what a key means in the editor's grid, as a value rather than as a branch inside a component.
//
// The rules are a spreadsheet's, and the interesting ones are the ones nobody tests by hand: Tab wrapping to the next
// row, Tab *leaving* the table at its edges, Page Up moving a screen and not a screen plus one. A keyboard trap is a
// defect (WCAG 2.1.2), and a 5000-row grid with no way out is the worst kind, so "leave" is an answer this module can
// give rather than something the caller has to remember to allow.

export interface Cursor {
  row: number;
  col: number;
}

export interface GridShape {
  /** How many books are in the table. */
  rows: number;
  /** How many columns are visible. */
  columns: number;
  /** How many rows Page Up and Page Down move. */
  page: number;
}

export type Intent =
  | { kind: "move"; cursor: Cursor }
  /** Tab at the first or last cell: the browser should move focus on, out of the table. */
  | { kind: "leave" }
  | { kind: "select-all" }
  | { kind: "none" };

export function clamp(value: number, low: number, high: number): number {
  return Math.min(Math.max(value, low), high);
}

function hasRows(shape: GridShape): boolean {
  return shape.rows > 0 && shape.columns > 0;
}

/**
 * Where the cursor goes for a key, independent of what is drawn.
 *
 * Returns `none` for any key the grid does not own, so the caller can go on to Enter, Space, type-to-edit and Escape
 * without a second list of exclusions to keep in step.
 */
export function gridKey(
  key: string,
  cursor: Cursor,
  shape: GridShape,
  modifiers: { shift?: boolean; ctrl?: boolean; meta?: boolean } = {},
): Intent {
  if (!hasRows(shape)) return { kind: "none" };
  const lastRow = shape.rows - 1;
  const lastCol = shape.columns - 1;
  const row = clamp(cursor.row, 0, lastRow);
  const col = clamp(cursor.col, 0, lastCol);

  const steps: Record<string, [number, number]> = {
    ArrowDown: [1, 0],
    ArrowUp: [-1, 0],
    ArrowLeft: [0, -1],
    ArrowRight: [0, 1],
  };
  const step = steps[key];
  if (step !== undefined) {
    return { kind: "move", cursor: { row: clamp(row + step[0], 0, lastRow), col: clamp(col + step[1], 0, lastCol) } };
  }

  if (key === "Tab") {
    // The last cell forward and the first cell backward are exits, not a wrap: a table you cannot Tab out of is a trap.
    if (!modifiers.shift && row === lastRow && col === lastCol) return { kind: "leave" };
    if (modifiers.shift && row === 0 && col === 0) return { kind: "leave" };
    let nextCol = col + (modifiers.shift ? -1 : 1);
    let nextRow = row;
    if (nextCol > lastCol) {
      nextCol = 0;
      nextRow = clamp(row + 1, 0, lastRow);
    } else if (nextCol < 0) {
      nextCol = lastCol;
      nextRow = clamp(row - 1, 0, lastRow);
    }
    return { kind: "move", cursor: { row: nextRow, col: nextCol } };
  }

  if (key === "Home" || key === "End") {
    const edge = key === "Home";
    if (modifiers.ctrl || modifiers.meta) return { kind: "move", cursor: { row: edge ? 0 : lastRow, col: edge ? 0 : lastCol } };
    return { kind: "move", cursor: { row, col: edge ? 0 : lastCol } };
  }

  if (key === "PageDown" || key === "PageUp") {
    const delta = key === "PageDown" ? Math.max(1, shape.page) : -Math.max(1, shape.page);
    return { kind: "move", cursor: { row: clamp(row + delta, 0, lastRow), col } };
  }

  if ((key === "a" || key === "A") && (modifiers.ctrl || modifiers.meta)) return { kind: "select-all" };

  return { kind: "none" };
}
