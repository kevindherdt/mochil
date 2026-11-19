import assert from "node:assert/strict";
import test from "node:test";
import { destinationOf, groupByDestination, tally, readyRows, STATUS_FILTERS, chunk, entriesFor, fieldText, parseEdit, sidecarFor, splitPreview, nextSort, sortRows, statusesFor, rangeBetween, targetOf, validateEdit, view, withoutRepeatedTargets } from "../static/import-model.js";

const row = { source_path: "S/S - Volume 01.epub", suggested_target: "S/S - Vol 01.epub", proposal: { title: "T", authors: ["A"], series: "S", series_index: 1, identifiers: { isbn: "9" } } };

test("a row shows the guess until the person edits it", () => {
  assert.deepEqual(view(row), { title: "T", authors: ["A"], series: "S", series_index: 1 });
  const edited = view(row, { [row.source_path]: { title: "New", series_index: null } });
  assert.deepEqual([edited.title, edited.series_index, edited.series], ["New", null, "S"]);
  assert.deepEqual(view({ source_path: "x", proposal: {} }), { title: "", authors: [], series: "", series_index: null });
});

test("the target is the recomputed one when there is one, even an empty one", () => {
  assert.equal(targetOf(row), "S/S - Vol 01.epub");
  assert.equal(targetOf(row, { [row.source_path]: "Other/Path.epub" }), "Other/Path.epub");
  assert.equal(targetOf(row, { [row.source_path]: "" }), "", "a book that can no longer be named has no target");
});

test("books without a target are set aside, not renamed", () => {
  const unnamed = { source_path: "u.epub", suggested_target: "" };
  const resolve = (r) => targetOf(r);
  assert.deepEqual(entriesFor([row, unnamed], resolve), { entries: [{ source_path: row.source_path, target_path: "S/S - Vol 01.epub" }], unnamed: ["u.epub"] });
});

test("each book goes to its own library, or the page default, and a stale choice falls back", () => {
  const dests = [{ id: "a", name: "A", folder: "a" }, { id: "b", name: "B", folder: "b-books" }];
  const r1 = { source_path: "1" }, r2 = { source_path: "2" }, r3 = { source_path: "3" };
  const choices = { 1: "b", 3: "gone", 2: "" };
  assert.equal(destinationOf(r1, choices, "a", dests), "b");
  assert.equal(destinationOf(r2, choices, "a", dests), "", "an explicit managed-folder choice beats the default");
  assert.equal(destinationOf(r3, choices, "a", dests), "a", "a library that no longer exists falls back to the default");
  assert.equal(destinationOf({ source_path: "4" }, choices, "a", dests), "a");
  assert.equal(destinationOf({ source_path: "4" }, {}, "missing", dests), "");
  const counts = tally([r1, r2, r3, { source_path: "4" }], (r) => destinationOf(r, choices, "a", dests), (id) => (id ? dests.find((d) => d.id === id).name : "the managed folder"));
  assert.deepEqual(counts, [{ name: "B", count: 1 }, { name: "the managed folder", count: 1 }, { name: "A", count: 2 }]);
});

test("chunking keeps order and sizes", () => {
  assert.deepEqual(chunk([1, 2, 3, 4, 5], 2), [[1, 2], [3, 4], [5]]);
  assert.deepEqual(chunk([], 100), []);
});

test("readyRows picks which chosen rows the server confirmed ready, by source_path", () => {
  const chosen = [{ source_path: "a" }, { source_path: "b" }, { source_path: "c" }];
  const ready = [{ source_path: "c", target_path: "C" }, { source_path: "a", target_path: "A" }];
  assert.deepEqual(readyRows(chosen, ready), [{ source_path: "a" }, { source_path: "c" }], "order follows chosen, not ready");
  assert.deepEqual(readyRows(chosen, []), []);
  assert.deepEqual(readyRows([], ready), []);
});

test("readyRows scales linearly, not quadratically, for a large batch", () => {
  const big = Array.from({ length: 4000 }, (_, i) => ({ source_path: `f${i}.epub` }));
  const halfReady = big.slice(0, 2000).map((row) => ({ source_path: row.source_path, target_path: row.source_path }));
  const start = performance.now();
  const picked = readyRows(big, halfReady);
  const elapsed = performance.now() - start;
  assert.equal(picked.length, 2000);
  assert.ok(elapsed < 50, `readyRows took ${elapsed}ms for 4000x2000 -- an O(n*m) scan would be far slower`);
});

test("a preview splits into ready and blocked with plain reasons", () => {
  const split = splitPreview({ entries: [
    { source_path: "a", target_path: "A", ready: true },
    { source_path: "b", target_path: "B", ready: false, issues: [{ code: "target_exists" }, { code: "something_new", message: "Custom" }] },
  ] });
  assert.deepEqual(split.ready, [{ source_path: "a", target_path: "A" }]);
  assert.deepEqual(split.blocked[0].reasons, ["A file is already saved there", "Custom"]);
});

test("the sidecar carries the edits and the identifiers from the book", () => {
  const sidecar = sidecarFor(row, { [row.source_path]: { authors: ["B", "C"], series_index: 2.5 } });
  assert.deepEqual(sidecar, { title: "T", authors: ["B", "C"], series: "S", series_index: 2.5, identifiers: { isbn: "9" } });
  assert.deepEqual(sidecarFor({ source_path: "x", proposal: {} }).identifiers, {});
});

test("edits are validated and parsed like the editor's", () => {
  assert.match(validateEdit("title", " "), /empty/);
  assert.match(validateEdit("series_index", "0"), /positive/);
  assert.equal(validateEdit("series_index", ""), "");
  assert.deepEqual(parseEdit("authors", "A & B &"), ["A", "B"]);
  assert.deepEqual(parseEdit("authors", "AT&T Labs & B"), ["AT&T Labs", "B"], "an & inside a name is kept");
  assert.equal(parseEdit("series_index", ""), null);
  assert.equal(parseEdit("series_index", "1.5"), 1.5);
  assert.equal(fieldText({ authors: ["A", "B"], series_index: 2, title: "T", series: "" }, "authors"), "A & B");
  assert.equal(fieldText({ authors: [], series_index: null, title: "", series: "" }, "series_index"), "");
});

test("the default filter shows what is new and every filter has a server status list", () => {
  assert.equal(statusesFor("fresh"), "new,new_volume");
  assert.equal(statusesFor("unknown"), "new,new_volume");
  assert.equal(statusesFor("all"), "");
  assert.equal(new Set(STATUS_FILTERS.map(([id]) => id)).size, STATUS_FILTERS.length);
});

test("the same name in two different libraries is fine, in one library it is a clash", () => {
  const out = withoutRepeatedTargets([{ source_path: "a", target_path: "S/x.epub", destination: "one" }, { source_path: "b", target_path: "S/x.epub", destination: "two" }, { source_path: "c", target_path: "s/X.epub", destination: "one" }]);
  assert.deepEqual(out.ready.map((e) => e.source_path), ["a", "b"]);
  assert.deepEqual(out.blocked.map((e) => e.source_path), ["c"]);
});

test("entries are grouped by library without the client-only key", () => {
  const groups = groupByDestination([{ source_path: "1", target_path: "a", destination: "x" }, { source_path: "2", target_path: "b" }, { source_path: "3", target_path: "c", destination: "x" }]);
  assert.deepEqual(groups, [["x", [{ source_path: "1", target_path: "a" }, { source_path: "3", target_path: "c" }]], ["", [{ source_path: "2", target_path: "b" }]]]);
});

test("two books headed for the same name are caught even across batches, ignoring case", () => {
  const out = withoutRepeatedTargets([{ source_path: "a", target_path: "S/S - Vol 01.epub" }, { source_path: "b", target_path: "s/s - vol 01.EPUB" }, { source_path: "c", target_path: "S/S - Vol 02.epub" }]);
  assert.deepEqual(out.ready.map((e) => e.source_path), ["a", "c"]);
  assert.equal(out.blocked[0].source_path, "b");
  assert.match(out.blocked[0].reasons[0], /same|another/i);
});

test("shift-click selects everything between two clicks, in either direction", () => {
  const ids = ["a", "b", "c", "d", "e"];
  assert.deepEqual(rangeBetween(ids, "b", "d"), ["b", "c", "d"]);
  assert.deepEqual(rangeBetween(ids, "d", "b"), ["b", "c", "d"]);
  assert.deepEqual(rangeBetween(ids, "", "c"), ["c"]);
  assert.deepEqual(rangeBetween(ids, "zz", "c"), ["c"]);
  assert.deepEqual(rangeBetween(ids, "a", "nope"), []);
});

test("the list sorts by a column, keeps empty values last both ways, and ties keep the folder order", () => {
  const book = (order, title, series, index, status = "new") => ({ order, status, source_path: `p${order}`, suggested_target: `t${order}`, proposal: { title, series, series_index: index, authors: [] } });
  const rows = [book(0, "b", "S", 10), book(1, "a", "", null), book(2, "c", "S", 2), book(3, "d", "T", 2, "in_library")];
  const ids = (list) => list.map((r) => r.order).join("");
  assert.equal(ids(sortRows(rows, "title", "asc")), "1023");
  assert.equal(ids(sortRows(rows, "title", "desc")), "3201");
  assert.equal(ids(sortRows(rows, "series_index", "asc")), "2301", "numbers sort as numbers, the empty one last");
  assert.equal(ids(sortRows(rows, "series_index", "desc")), "0231", "descending keeps the empty one last, ties keep folder order");
  assert.equal(ids(sortRows(rows, "series", "desc")), "3021");
  assert.equal(ids(sortRows(rows, "status", "asc")), "0123", "new before already in the library, ties in folder order");
  assert.equal(ids(sortRows([rows[3], rows[1], rows[0]], "", "asc")), "013", "no sort restores the folder order");
  assert.equal(ids(sortRows(rows, "title", "asc", { p1: { title: "z" } })), "0231", "an edited title sorts as edited");
  assert.equal(ids(sortRows(rows, "target", "asc", {}, { p0: "" })), "1230", "a book with no target goes last");
});

test("clicking a heading cycles ascending, descending, then back to the folder order", () => {
  let sort = { key: "", dir: "asc" };
  sort = nextSort(sort, "title"); assert.deepEqual(sort, { key: "title", dir: "asc" });
  sort = nextSort(sort, "title"); assert.deepEqual(sort, { key: "title", dir: "desc" });
  sort = nextSort(sort, "title"); assert.deepEqual(sort, { key: "", dir: "asc" });
  assert.deepEqual(nextSort({ key: "title", dir: "desc" }, "series"), { key: "series", dir: "asc" });
});
