// How wide each column of the editor's table is, which the person can change by dragging the edge of its heading. The
// widths are kept per column in the browser, so a column keeps its width in every library.

export const MIN_WIDTH = 36;
export const MAX_WIDTH = 900;
const FALLBACK = 150;
const CHECK_COLUMN = 40; // the checkbox column
const COVER_COLUMN = 80; // the cover column, which is only there in the comfortable view

// What each column starts at, by what it usually holds.
export const DEFAULT_WIDTHS = {
  title: 300, authors: 180, series: 200, series_index: 56, tags: 160, publisher: 140, published: 110, language: 80, rating: 100,
  isbn: 140, identifiers: 240, title_sort: 220, author_sort: 200, description: 320, progress: 90, words: 90, pages: 80,
  isbn_found: 140, match: 260, format: 70, collections: 200, hardcover: 130, path: 360, folder: 300, library_folder: 280, status: 240,
};

export const clampWidth = (pixels) => Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, Math.round(pixels)));
export const widthOf = (id, saved = {}) => clampWidth(saved[id] ?? DEFAULT_WIDTHS[id] ?? FALLBACK);

// gridWidth is the width of the whole table: its columns side by side, so a wide table scrolls sideways.
export const gridWidth = (ids, saved, comfortable) => CHECK_COLUMN + (comfortable ? COVER_COLUMN : 0) + ids.reduce((sum, id) => sum + widthOf(id, saved), 0);

// readWidths turns what was stored back into widths, ignoring anything that is not a width.
export function readWidths(text) {
  try {
    const stored = JSON.parse(text || "{}");
    return Object.fromEntries(Object.entries(stored && typeof stored === "object" && !Array.isArray(stored) ? stored : {}).filter(([, width]) => Number.isFinite(width)).map(([id, width]) => [id, clampWidth(width)]));
  } catch (_) { return {}; }
}

// bindResize lets the person drag the right edge of a heading (the element marked data-resize) to change its column,
// and double-click it to put the column back to its starting width. onChange gets the new widths to keep.
export function bindResize({ head, grid, widths, ids, comfortable, onChange }) {
  let drag = null;
  const apply = (id, width) => {
    const cell = head.querySelector(`[data-col-id="${id}"]`);
    if (cell) cell.style.width = `${width}px`;
    grid.style.width = `${gridWidth(ids(), widths, comfortable())}px`;
  };
  head.addEventListener("pointerdown", (event) => {
    const handle = event.target.closest?.("[data-resize]");
    const cell = handle?.closest("[data-col-id]");
    if (!cell) return;
    event.preventDefault();
    event.stopPropagation();
    // Start from the width it has on screen: a table with room to spare stretches its columns a little.
    const shown = Math.round(cell.getBoundingClientRect?.().width || 0);
    drag = { id: cell.dataset.colId, from: event.clientX, width: shown || widthOf(cell.dataset.colId, widths) };
    handle.setPointerCapture?.(event.pointerId);
  });
  head.addEventListener("pointermove", (event) => {
    if (!drag) return;
    widths[drag.id] = clampWidth(drag.width + event.clientX - drag.from);
    apply(drag.id, widths[drag.id]);
  });
  const finish = () => {
    if (!drag) return;
    drag = null;
    onChange(widths);
  };
  head.addEventListener("pointerup", finish);
  head.addEventListener("pointercancel", finish);
  head.addEventListener("dblclick", (event) => {
    const cell = event.target.closest?.("[data-resize]")?.closest("[data-col-id]");
    if (!cell) return;
    delete widths[cell.dataset.colId];
    apply(cell.dataset.colId, widthOf(cell.dataset.colId, widths));
    onChange(widths);
  });
  // A click on the handle must not also sort the column.
  head.addEventListener("click", (event) => { if (event.target.closest?.("[data-resize]")) event.stopPropagation(); }, true);
}
