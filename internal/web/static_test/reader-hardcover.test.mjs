import assert from "node:assert/strict";
import test from "node:test";
import { createReaderHardcover, hardcoverLine, previewLine } from "../static/reader-hardcover.js";

test("the line says what the book is matched to, and how", () => {
  assert.deepEqual(
    hardcoverLine({ enabled: true, match: { hardcover_title: "Dune", how: "matched", url: "https://hardcover.app/books/dune" } }),
    { text: "Hardcover: Dune (matched by name)", href: "https://hardcover.app/books/dune" },
  );
  assert.equal(hardcoverLine({ enabled: true, match: { hardcover_title: "Rebuild World: Volume 1 Part 2", how: "linked" } }).text, "Hardcover: Rebuild World: Volume 1 Part 2 (linked by you)");
  assert.match(hardcoverLine({ enabled: true, match: { hardcover_id: "42", how: "matched" } }).text, /book 42/);
});

test("a doubtful match asks the reader to check it", () => {
  assert.match(hardcoverLine({ enabled: true, match: { hardcover_title: "Dune", how: "matched", doubt: "Several Hardcover books have this title and author." } }).text, /check this match/);
  assert.doesNotMatch(hardcoverLine({ enabled: true, match: { hardcover_title: "Dune", how: "matched" } }).text, /check/);
});

test("an unmatched or unsent book says so, and nothing shows when the module is off", () => {
  assert.match(hardcoverLine({ enabled: true, match: null, skipped: true }).text, /not matched.*Hardcover shelf/);
  assert.match(hardcoverLine({ enabled: true, match: null, skipped: false }).text, /nothing sent yet/);
  assert.equal(hardcoverLine({ enabled: false, match: null }), null);
  assert.equal(hardcoverLine(null), null);
});

function fixture(response) {
  const line = { hidden: true, textContent: "", children: [], append(...parts) { this.children.push(...parts); } };
  const document = { querySelector: (selector) => (selector === "#hardcover-line" ? line : null), createElement: () => ({}) };
  const requested = [];
  const fetchImpl = async (url) => {
    requested.push(url);
    if (response instanceof Error) throw response;
    return response;
  };
  return { line, requested, reader: createReaderHardcover({ document, library: "light-novels", id: "b 1", fetchImpl }) };
}

test("the reader asks about this book and shows the match with a link", async () => {
  const { line, requested, reader } = fixture({ ok: true, json: async () => ({ enabled: true, match: { hardcover_title: "Dune", how: "matched", url: "https://hardcover.app/books/dune" } }) });
  await reader.refresh();
  assert.equal(requested[0], "/api/metadata/hardcover/match?library=light-novels&book=b%201");
  assert.equal(line.hidden, false);
  assert.equal(line.textContent, "Hardcover: Dune (matched by name)");
  assert.equal(line.children[1].href, "https://hardcover.app/books/dune");
  assert.equal(line.children[1].rel, "noopener noreferrer");
});

test("a reader who may not see it, or a network failure, leaves the line hidden", async () => {
  for (const response of [{ ok: false, status: 403 }, new Error("offline")]) {
    const { line, reader } = fixture(response);
    await reader.refresh();
    assert.equal(line.hidden, true);
  }
});

test("a preview says what the book would match to, and never claims it was sent", () => {
  assert.deepEqual(previewLine({ match: { title: "Dune", authors: ["Frank Herbert"], source_url: "https://hardcover.app/books/dune" } }), { text: "Would match: Dune by Frank Herbert", href: "https://hardcover.app/books/dune" });
  assert.match(previewLine({ match: { title: "Dune", authors: [] }, ambiguous: true }).text, /several Hardcover books fit/);
  assert.match(previewLine({ match: { title: "Dune", authors: [] }, linked: true }).text, /linked by you/);
  const none = previewLine({ match: null, candidates: [{ title: "A" }, { title: "B" }, { title: "C" }, { title: "D" }] }).text;
  assert.match(none, /No clear match on Hardcover\. Closest: A; B; C\./);
  assert.doesNotMatch(none, /D/);
  assert.match(previewLine(null).text, /could not be reached/);
});

test("Check match appears only while nothing is sent, and asks Hardcover without sending", async () => {
  const buttons = [];
  const line = { hidden: true, textContent: "", append(...parts) { parts.forEach((part) => part?.onclick !== undefined && buttons.push(part)); } };
  const document = { querySelector: () => line, createElement: () => ({}) };
  const requested = [];
  const fetchImpl = async (url) => {
    requested.push(url);
    if (url.includes("/match?")) return { ok: true, json: async () => ({ enabled: true, match: null, skipped: false }) };
    return { ok: true, json: async () => ({ match: { title: "Dune", authors: ["Frank Herbert"], source_url: "https://hardcover.app/books/dune" }, candidates: [] }) };
  };
  const reader = createReaderHardcover({ document, library: "books", id: "a1", fetchImpl });
  await reader.refresh();
  assert.equal(buttons.length, 1, "one Check match button");
  assert.equal(buttons[0].textContent, "Check match");
  await buttons[0].onclick();
  assert.equal(requested.at(-1), "/api/metadata/hardcover/preview?library=books&book=a1");
  assert.equal(line.textContent, "Would match: Dune by Frank Herbert");

  const matched = [];
  const line2 = { hidden: true, textContent: "", append(...parts) { parts.forEach((part) => part?.onclick !== undefined && matched.push(part)); } };
  const sent = createReaderHardcover({
    document: { querySelector: () => line2, createElement: () => ({}) }, library: "books", id: "a1",
    fetchImpl: async () => ({ ok: true, json: async () => ({ enabled: true, match: { hardcover_title: "Dune", how: "matched" } }) }),
  });
  await sent.refresh();
  assert.equal(matched.length, 0, "no button once the book has a match");
});
