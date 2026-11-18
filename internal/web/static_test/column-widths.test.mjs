import assert from "node:assert/strict";
import test from "node:test";
import { DEFAULT_WIDTHS, MAX_WIDTH, MIN_WIDTH, bindResize, clampWidth, gridWidth, readWidths, widthOf } from "../static/column-widths.js";
import { COLUMNS } from "../static/editor-model.js";

test("a width is kept between a readable minimum and a sane maximum", () => {
  assert.equal(clampWidth(5), MIN_WIDTH);
  assert.equal(clampWidth(5000), MAX_WIDTH);
  assert.equal(clampWidth(200.4), 200);
});

test("a column starts at its default, an unknown one at a fallback, and a saved width wins", () => {
  assert.equal(widthOf("title", {}), DEFAULT_WIDTHS.title);
  assert.equal(widthOf("#my_column", {}), 150);
  assert.equal(widthOf("title", { title: 500 }), 500);
  assert.equal(widthOf("title", { title: 3 }), MIN_WIDTH);
  assert.ok(MIN_WIDTH <= 40 && widthOf("series_index", { series_index: 40 }) === 40, "a narrow column such as # can be made narrow");
});

test("every built-in column has a starting width", () => {
  for (const column of COLUMNS.filter((c) => !c.custom)) assert.ok(DEFAULT_WIDTHS[column.id], `${column.id} has a default width`);
});

test("the table is as wide as its columns, with room for the checkbox and, when comfortable, the cover", () => {
  assert.equal(gridWidth(["title", "authors"], {}, false), 40 + 300 + 180);
  assert.equal(gridWidth(["title", "authors"], { title: 100 }, true), 40 + 80 + 100 + 180);
});

test("stored widths that are not widths are ignored", () => {
  assert.deepEqual(readWidths('{"title":420,"authors":"wide","x":null,"y":9999}'), { title: 420, y: MAX_WIDTH });
  assert.deepEqual(readWidths("not json"), {});
  assert.deepEqual(readWidths(null), {});
  assert.deepEqual(readWidths("[1,2]"), {});
});

// A heading row with one column "title": enough of a DOM for the drag.
function fakeHead() {
  const listeners = {};
  const cell = { dataset: { colId: "title" }, style: {} };
  const handle = { closest: (selector) => (selector === "[data-col-id]" ? cell : handle), setPointerCapture() {} };
  const head = {
    addEventListener: (type, fn) => { (listeners[type] ||= []).push(fn); },
    querySelector: () => cell,
  };
  const fire = (type, event) => (listeners[type] || []).forEach((fn) => fn({ preventDefault() {}, stopPropagation() {}, pointerId: 1, ...event }));
  const onHandle = { closest: (selector) => (selector === "[data-resize]" ? handle : null) };
  return { head, cell, fire, onHandle };
}

test("dragging a heading's edge resizes the column live, updates the table, and keeps the width when let go", () => {
  const { head, cell, fire, onHandle } = fakeHead();
  const grid = { style: {} };
  const widths = {}, kept = [];
  bindResize({ head, grid, widths, ids: () => ["title"], comfortable: () => false, onChange: (w) => kept.push({ ...w }) });
  fire("pointerdown", { target: onHandle, clientX: 500 });
  fire("pointermove", { clientX: 560 });
  assert.equal(cell.style.width, `${DEFAULT_WIDTHS.title + 60}px`);
  assert.equal(grid.style.width, `${40 + DEFAULT_WIDTHS.title + 60}px`);
  assert.deepEqual(kept, [], "nothing is stored until the drag ends");
  fire("pointermove", { clientX: 0 });
  assert.equal(cell.style.width, `${MIN_WIDTH}px`, "dragging too far stops at the minimum");
  fire("pointerup", {});
  assert.deepEqual(kept, [{ title: MIN_WIDTH }]);
  fire("pointermove", { clientX: 900 });
  assert.equal(kept.length, 1, "moving after letting go changes nothing");
});

test("double-clicking the edge puts the column back to its starting width", () => {
  const { head, cell, fire, onHandle } = fakeHead();
  const widths = { title: 700 }, kept = [];
  bindResize({ head, grid: { style: {} }, widths, ids: () => ["title"], comfortable: () => false, onChange: (w) => kept.push({ ...w }) });
  fire("dblclick", { target: onHandle });
  assert.equal(cell.style.width, `${DEFAULT_WIDTHS.title}px`);
  assert.deepEqual(kept, [{}]);
});

test("pressing anywhere but the handle does nothing", () => {
  const { head, fire } = fakeHead();
  const widths = {}, kept = [];
  bindResize({ head, grid: { style: {} }, widths, ids: () => [], comfortable: () => false, onChange: (w) => kept.push(w) });
  fire("pointerdown", { target: { closest: () => null }, clientX: 10 });
  fire("pointermove", { clientX: 99 });
  fire("pointerup", {});
  assert.deepEqual(kept, []);
});

test("the standard columns together fit a normal laptop window", () => {
  assert.ok(gridWidth(["title", "authors", "series", "series_index", "format", "status"], {}, false) <= 1100);
});

test("every column can be sorted by clicking its heading", () => {
  for (const column of COLUMNS) assert.ok(column.sort, `${column.id} has no sort`);
});

import { FILTERS, FOUND_BY_CHECK, issueText } from "../static/editor-model.js";
import { rowHTML } from "../static/editor-rows.js";

test("the Hardcover sync column shows the state in words, with its explanation as the tooltip", () => {
  const state = { density: "dense", pending: {}, focus: { row: -1, col: -1 }, selected: new Set(), resets: new Set(), library: "books", libraryFolder: "" };
  const column = COLUMNS.find((c) => c.id === "hardcover");
  const html = (hardcover) => rowHTML({ id: "a", title: "A", hardcover, hardcover_note: hardcover ? 'Why "so"' : "" }, {}, 0, { columns: [column], editable: [], state });
  assert.match(html("doubt"), /title="Why &quot;so&quot;">Check match</);
  assert.match(html("synced"), />Synced</);
  assert.match(html(""), /<td><\/td><td class="ed-fill">/);
});

test("the Hardcover findings have labels and filters that only show once something is found", () => {
  for (const id of ["hardcover_doubt", "hardcover_nomatch", "hardcover_ignored"]) {
    assert.ok(FOUND_BY_CHECK.has(id), `${id} is hidden until found`);
    assert.ok(FILTERS.some(([filter]) => filter === id), `${id} has a filter`);
    assert.notEqual(issueText(id), id, `${id} has a readable label`);
  }
});
