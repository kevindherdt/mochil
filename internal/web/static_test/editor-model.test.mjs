import assert from "node:assert/strict";
import test from "node:test";
import { progressText, setCustomColumns, matchChanges, matchFields, scoreText, COLUMNS, FIELDS, defaultColumns, fieldValue, merge, shown, stars, visibleColumns, pendingCount, renumber, replaceText, request, setAll, stage, validate, withPending } from "../static/editor-model.js";

const dune = { id: "a", title: "Dune - Vol 1", authors: ["Frank Herbert"], series: "Dune", series_index: 1 };
const messiah = { id: "b", title: "Dune - Vol 2", authors: ["Frank Herbert"], series: "Dune", series_index: null };

test("staging an edit that matches the book forgets it", () => {
  let pending = stage({}, dune, "title", "Dune");
  assert.deepEqual(pending, { a: { title: "Dune" } });
  pending = stage(pending, dune, "title", "Dune - Vol 1");
  assert.deepEqual(pending, {}, "editing back to the original leaves nothing to save");
  assert.deepEqual(stage({}, dune, "series_index", " 1 "), {}, "1 is already the number");
});

test("the row shows its pending edits, and authors are joined with &", () => {
  const row = withPending(dune, { a: { title: "X", authors: "A & B", series_index: "2.5", series: "S" } });
  assert.deepEqual([row.title, row.authors, row.series_index, row.series], ["X", ["A", "B"], 2.5, "S"]);
  assert.equal(withPending(dune, { a: { series_index: "" } }).series_index, null);
  assert.equal(fieldValue(dune, "authors"), "Frank Herbert");
  assert.equal(pendingCount({ a: { title: "x", series: "y" }, b: { title: "z" } }), 3);
});

test("validation matches what the server accepts", () => {
  assert.match(validate("title", "  "), /empty/);
  assert.match(validate("series_index", "two"), /positive/);
  assert.match(validate("series_index", "0"), /positive/);
  assert.equal(validate("series_index", ""), "", "clearing a number is allowed");
  assert.equal(validate("series_index", "1.5"), "");
  assert.equal(validate("series", ""), "");
});

test("renumber follows the given order, with a start and a step", () => {
  assert.deepEqual(renumber([dune, messiah]), { a: { series_index: "1" }, b: { series_index: "2" } });
  assert.deepEqual(renumber([messiah, dune], 3, 0.5), { b: { series_index: "3" }, a: { series_index: "3.5" } });
});

test("stripping a prefix or replacing text only touches rows that change", () => {
  assert.deepEqual(replaceText([dune, messiah], "title", "dune - ", "", { prefixOnly: true }), { a: { title: "Vol 1" }, b: { title: "Vol 2" } });
  assert.deepEqual(replaceText([dune], "title", "Vol", "Volume"), { a: { title: "Dune - Volume 1" } });
  assert.deepEqual(replaceText([dune], "title", "Vol", "", { matchCase: true }), { a: { title: "Dune - 1" } });
  assert.deepEqual(replaceText([dune], "title", "nothing here"), {}, "no match, no edit");
  assert.deepEqual(replaceText([dune], "title", ""), {}, "an empty search changes nothing");
  assert.deepEqual(replaceText([{ id: "z", title: "a.b" }], "title", ".", "-"), { z: { title: "a-b" } }, "the search is literal text, not a pattern");
  assert.deepEqual(replaceText([{ id: "z", title: "$& cost" }], "title", "cost", "$&"), { z: { title: "$& $&" } }, "the replacement is literal too");
});

test("bulk edits merge into pending and drop edits that change nothing", () => {
  const pending = merge({}, { ...setAll([dune, messiah], "series", "Dune"), ...setAll([dune], "title", "New") }, [dune, messiah]);
  assert.deepEqual(pending, { a: { title: "New" } }, "series already is Dune");
  assert.deepEqual(merge({}, { ghost: { title: "x" } }, [dune]), {}, "an unknown book is ignored");
});

test("the request carries the library, the edits and whether it is only a preview", () => {
  assert.deepEqual(request("books", { a: { title: "x" } }, { preview: true, reset: ["b"] }), { library: "books", changes: { a: { title: "x" } }, reset: ["b"], preview: true });
});

test("the Calibre-style fields can be edited, shown and validated", () => {
  const book = { id: "c", title: "T", authors: [], tags: ["Fantasy", "Classic"], rating: 4.5, identifiers: "isbn:1", description: "x".repeat(200) };
  assert.equal(fieldValue(book, "tags"), "Fantasy, Classic");
  assert.equal(fieldValue(book, "rating"), "4.5");
  assert.equal(stars(4.5), "★★★★½");
  assert.equal(stars(0), "");
  assert.equal(shown(book, "rating"), "★★★★½");
  assert.equal(shown(book, "description").length, 141, "long descriptions are cut in the grid");
  assert.equal(fieldValue(book, "description").length, 200, "but editing starts from the whole text");
  const row = withPending(book, { c: { tags: "A, B ,", rating: "", publisher: "Ace" } });
  assert.deepEqual([row.tags, row.rating, row.publisher], [["A", "B"], 0, "Ace"]);
  assert.match(validate("rating", "6"), /0 and 5/);
  assert.match(validate("rating", "x"), /0 and 5/);
  assert.equal(validate("rating", "2.5"), "");
  assert.match(validate("identifiers", "9781975327071"), /isbn:978/);
  assert.match(validate("identifiers", "isbn:1, nope"), /isbn:978/);
  assert.match(validate("identifiers", "a<b:1"), /isbn:978/);
  assert.equal(validate("identifiers", "isbn:978:x, hardcover:dune"), "");
  assert.equal(validate("tags", ""), "");
});

test("the column list is consistent with what the server accepts", () => {
  assert.deepEqual(FIELDS, ["title", "authors", "series", "series_index", "tags", "publisher", "published", "language", "rating", "identifiers", "title_sort", "author_sort", "description"]);
  assert.deepEqual(defaultColumns(), ["title", "authors", "series", "series_index", "format", "status"]);
  assert.deepEqual(visibleColumns(["status", "ghost", "title"]).map((column) => column.id), ["title", "status"], "COLUMNS order wins and unknown ids are ignored");
  assert.equal(new Set(COLUMNS.map((column) => column.id)).size, COLUMNS.length);
});

test("a publication date shows as a date, not a timestamp", () => {
  assert.equal(shown({ published: "2021-05-04T00:00:00+00:00" }, "published"), "2021-05-04");
  assert.equal(shown({ published: "1965" }, "published"), "1965");
  assert.equal(fieldValue({ published: "2021-05-04T00:00:00+00:00" }, "published"), "2021-05-04T00:00:00+00:00", "editing keeps the stored value");
});

test("a match offers only the fields where it differs, and a link without losing other identifiers", () => {
  const row = { title: "Dune", authors: "Frank Herbert", published: "", description: "", identifiers: "isbn:1, hardcover:old" };
  const candidate = { title: "Dune", authors: ["Frank Herbert"], year: 1965, overview: "Spice.", provider: "Hardcover", provider_id: "42" };
  const fields = matchFields(row, candidate);
  assert.deepEqual(fields.map((f) => f.field), ["published", "description", "identifiers"]);
  assert.equal(fields[2].to, "isbn:1, hardcover:42");
  assert.deepEqual(matchChanges(fields, ["published", "identifiers"]), { published: "1965", identifiers: "isbn:1, hardcover:42" });
  assert.deepEqual(matchChanges(fields, []), {});
  assert.deepEqual(matchFields({ ...row, title: "Dune!" }, { title: "Dune" }).map((f) => f.field), ["title"]);
  assert.equal(scoreText(0.876), "88%");
  assert.equal(scoreText(2), "100%");
});

test("a publication date in the same year is not offered as a change", () => {
  const fields = matchFields({ title: "A", authors: "B", published: "1949-01-01T00:00:00+00:00", description: "x", identifiers: "" }, { title: "A", authors: ["B"], year: 1949, overview: "x" });
  assert.deepEqual(fields, []);
});

test("custom columns join the grid, hold their own values and are checked by type", () => {
  setCustomColumns([{ id: "#score", label: "Score", type: "number" }, { id: "#done", label: "Reviewed", type: "yesno" }]);
  try {
    assert.ok(COLUMNS.some((c) => c.id === "#score" && c.field === "#score" && c.sort === "#score" && c.num));
    const ids = COLUMNS.map((c) => c.id);
    assert.ok(ids.indexOf("#done") < ids.indexOf("progress") && ids.indexOf("progress") < ids.indexOf("format"));
    assert.ok(FIELDS.includes("#score") && !FIELDS.includes("progress"));
    const row = { id: "b", custom: { "#score": "7", "#done": "yes" } };
    assert.equal(fieldValue(row, "#score"), "7");
    assert.equal(fieldValue({ id: "c" }, "#score"), "");
    assert.equal(shown(row, "#done"), "✓ yes");
    assert.match(validate("#score", "high"), /number/);
    assert.equal(validate("#score", "7.5"), "");
    assert.match(validate("#done", "maybe"), /yes or no/);
    const pending = stage({}, row, "#score", "9");
    assert.deepEqual(pending, { b: { "#score": "9" } });
    assert.equal(withPending(row, pending).custom["#score"], "9");
    assert.deepEqual(withPending(row, { b: { "#score": "" } }).custom, { "#done": "yes" });
    assert.deepEqual(stage(pending, row, "#score", "7"), {}, "going back to the saved value cancels the edit");
  } finally { setCustomColumns([]); }
  assert.ok(!COLUMNS.some((c) => c.custom));
});

test("progress reads as a percentage or Finished", () => {
  assert.equal(progressText({ progress: 0 }), "");
  assert.equal(progressText({ progress: 0.004 }), "1%");
  assert.equal(progressText({ progress: 0.425 }), "43%");
  assert.equal(progressText({ progress: 1 }), "Finished");
});
