import assert from "node:assert/strict";
import test from "node:test";
import { COLUMNS, fieldValue, isValidIsbn, isbnOf, stage, validate, withIsbn, withPending } from "../static/editor-model.js";

const row = { id: "a", identifiers: "goodreads:9, isbn:9780306406157, kobo:x" };

test("the ISBN column shows the isbn entry of the identifiers", () => {
  assert.ok(COLUMNS.find((column) => column.id === "isbn" && column.field === "isbn"));
  assert.equal(isbnOf(row), "9780306406157");
  assert.equal(fieldValue(row, "isbn"), "9780306406157");
  assert.equal(isbnOf({ identifiers: "goodreads:9" }), "");
  assert.equal(isbnOf({}), "");
});

test("only a correct 13- or 10-digit ISBN is accepted, with or without hyphens", () => {
  assert.equal(isValidIsbn("978-0-306-40615-7"), true);
  assert.equal(isValidIsbn("0306406152"), true);
  assert.equal(isValidIsbn("080442957X"), true);
  assert.equal(isValidIsbn("9780306406158"), false, "wrong check digit");
  assert.equal(isValidIsbn("12345"), false);
  assert.equal(validate("isbn", ""), "", "an empty ISBN removes it");
  assert.notEqual(validate("isbn", "9780306406158"), "");
});

test("editing the ISBN changes only that entry of the identifiers, keeping its place", () => {
  const next = stage({}, row, "isbn", "978-0-7432-7356-5");
  assert.deepEqual(next, { a: { identifiers: "goodreads:9, isbn:9780743273565, kobo:x" } });
  assert.equal(fieldValue(withPending(row, next), "isbn"), "9780743273565");
  assert.deepEqual(stage({}, row, "isbn", "9780306406157"), {}, "the same ISBN is no edit");
  assert.deepEqual(stage({}, row, "isbn", ""), { a: { identifiers: "goodreads:9, kobo:x" } });
  assert.deepEqual(stage({}, { id: "b", identifiers: "" }, "isbn", "9780306406157"), { b: { identifiers: "isbn:9780306406157" } });
});

test("a second edit builds on the first instead of undoing it", () => {
  const first = stage({}, row, "identifiers", "goodreads:9, isbn:9780306406157, kobo:y");
  assert.deepEqual(stage(first, row, "isbn", "9780743273565"), { a: { identifiers: "goodreads:9, isbn:9780743273565, kobo:y" } });
});

test("withIsbn appends when there was none", () => {
  assert.equal(withIsbn("hardcover:dune", "9780306406157"), "hardcover:dune, isbn:9780306406157");
});

test("a found ISBN is a read-only column, and taking it is an ordinary staged edit of the ISBN", async () => {
  const { merge, FILTERS, FOUND_BY_CHECK, issueText, FIELDS } = await import("../static/editor-model.js");
  const found = COLUMNS.find((column) => column.id === "isbn_found");
  assert.ok(found && found.field === undefined, "read-only");
  assert.ok(!FIELDS.includes("isbn_found"), "the server does not take it as a field");
  for (const id of ["isbn_found", "isbn_unsure"]) {
    assert.ok(FILTERS.some(([filter]) => filter === id), `${id} has a filter`);
    assert.ok(FOUND_BY_CHECK.has(id), `${id} stays hidden until the search found some`);
    assert.notEqual(issueText(id), id, `${id} has words`);
  }
  const row = { id: "a", identifiers: "hardcover:dune", isbn_found: "9780306406157" };
  assert.deepEqual(merge({}, { a: { isbn: row.isbn_found } }, [row]), { a: { identifiers: "hardcover:dune, isbn:9780306406157" } });
});

test("taking a certain Hardcover match fills only empty fields and adds the link, and never a doubtful match", async () => {
  const { fillFromMatch, FILTERS, FOUND_BY_CHECK } = await import("../static/editor-model.js");
  const match = { title: "Dune", authors: ["Frank Herbert"], year: 1965, overview: "Spice.", provider: "Hardcover", provider_id: "5", score: 1 };
  const row = { title: "Dune", authors: "Frank Herbert", published: "", description: "", identifiers: "isbn:9780306406157", match, match_clear: true };
  assert.deepEqual(fillFromMatch(row), { published: "1965", description: "Spice.", identifiers: "isbn:9780306406157, hardcover:5" });
  const filled = { ...row, description: "My own blurb", published: "1966-01-01", title: "Dune (my edition)", authors: "F. Herbert" };
  assert.deepEqual(fillFromMatch(filled), { identifiers: "isbn:9780306406157, hardcover:5" }, "what the book already says is never replaced");
  assert.deepEqual(fillFromMatch({ ...row, match_clear: false }), {}, "a doubtful match is not taken in bulk");
  assert.deepEqual(fillFromMatch({ ...row, match: undefined }), {});
  for (const id of ["match_found", "match_unsure", "match_none"]) {
    assert.ok(FILTERS.some(([filter]) => filter === id) && FOUND_BY_CHECK.has(id), id);
  }
  assert.ok(COLUMNS.some((column) => column.id === "match" && column.field === undefined), "a read-only Hardcover match column");
});

test("Folder shows the folder a book sits in, and the file columns sort by path", async () => {
  const { folderOf } = await import("../static/editor-model.js");
  assert.equal(folderOf({ path: "Frank Herbert/Dune (12)/Dune.epub" }), "Frank Herbert/Dune (12)");
  assert.equal(folderOf({ path: "loose.epub" }), "");
  assert.equal(folderOf({}), "");
  for (const id of ["path", "folder"]) assert.equal(COLUMNS.find((column) => column.id === id)?.sort, "path");
  const library = COLUMNS.find((column) => column.id === "library_folder");
  assert.ok(library && library.field === undefined, "read-only");
});
