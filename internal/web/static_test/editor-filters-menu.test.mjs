import assert from "node:assert/strict";
import test from "node:test";
import { filterBarHTML, FILTER_PINNED, FILTER_GROUPS, FILTER_OTHER, FILTERS } from "../static/editor-model.js";

const baseState = (overrides = {}) => ({
  filter: "all",
  libraryTotal: 326,
  counts: {
    issues: 0, no_author: 0, no_number: 0, duplicate: 0, side_story: 0,
    no_description: 39, no_isbn: 88, isbn_found: 11, isbn_unsure: 26,
    match_found: 272, match_unsure: 54, hardcover_doubt: 1, edited: 0,
  },
  ...overrides,
});

test("demonstrates the problem: a flat chip per filter crowds the bar even at zero count", () => {
  // This is what editor.js used to build before the reorganization: one chip per FILTERS
  // entry, with only FOUND_BY_CHECK ids skipped when their count is zero. Several
  // chips (Needs review, No author, No number, Duplicates, Side stories, Edited) are not
  // in FOUND_BY_CHECK, so they always rendered even at zero, crowding the bar.
  const FOUND_BY_CHECK = new Set(["no_cover", "unreadable", "isbn_found", "isbn_unsure", "match_found", "match_unsure", "match_none", "hardcover_doubt", "hardcover_nomatch", "hardcover_ignored"]);
  const state = baseState();
  const oldMarkup = FILTERS.map(([id, label]) => {
    const count = id === "all" ? state.libraryTotal : id === "issues" ? ["duplicate", "no_author", "no_number"].reduce((n, k) => n + (state.counts[k] || 0), 0) : state.counts[id] || 0;
    if (FOUND_BY_CHECK.has(id) && !count && state.filter !== id) return "";
    return `<button type="button" data-filter="${id}" aria-pressed="${state.filter === id}">${label}<small>${count}</small></button>`;
  }).join("");
  const chipCount = (oldMarkup.match(/data-filter=/g) || []).length;
  assert.equal(chipCount, 14, "the old flat layout crammed 14 chips into one row, most at zero count");
})

test("the filter bar pins All and Needs review, and groups the rest behind a Filters menu", () => {
  const html = filterBarHTML(baseState());
  assert.ok(html.includes('data-filter="all"'));
  assert.ok(html.includes('data-filter="issues"'));
  assert.match(html, /<details[^>]*class="[^"]*ed-filters-menu[^"]*"/);
  assert.match(html, /<summary>Filters/);
  // the pinned chips are outside the <details>
  const detailsIndex = html.indexOf("<details");
  assert.ok(html.indexOf('data-filter="all"') < detailsIndex);
  assert.ok(html.indexOf('data-filter="issues"') < detailsIndex);
})

test("zero-count filters are left out of the menu unless they are the active filter", () => {
  const html = filterBarHTML(baseState());
  for (const id of ["no_author", "no_number", "duplicate", "side_story", "edited"]) {
    assert.ok(!html.includes(`data-filter="${id}"`), `${id} is hidden while its count is zero`);
  }
  const activeZero = filterBarHTML(baseState({ filter: "no_author" }));
  assert.ok(activeZero.includes('data-filter="no_author"'), "the active filter stays visible even at zero, so it can be cleared");
})

test("non-zero filters are grouped under Data quality, ISBN and Match headings", () => {
  const html = filterBarHTML(baseState());
  assert.match(html, /Data quality[\s\S]*data-filter="no_description"/)
  assert.match(html, /ISBN[\s\S]*data-filter="no_isbn"[\s\S]*data-filter="isbn_found"[\s\S]*data-filter="isbn_unsure"/)
  assert.match(html, /Match[\s\S]*data-filter="match_found"[\s\S]*data-filter="match_unsure"[\s\S]*data-filter="hardcover_doubt"/)
})

test("the Filters summary shows a badge while a grouped filter is active", () => {
  const inactive = filterBarHTML(baseState())
  assert.ok(!/ed-filters-badge/.test(inactive), "no badge while All or Needs review is the active filter")
  const active = filterBarHTML(baseState({ filter: "no_isbn" }))
  assert.match(active, /<summary>Filters<small class="ed-filters-badge">1<\/small><\/summary>/)
})

test("the filter groups and leftover list cover every filter id exactly once", () => {
  const grouped = new Set(FILTER_GROUPS.flatMap((g) => g.filters))
  const accounted = new Set([...FILTER_PINNED, ...grouped, ...FILTER_OTHER])
  for (const [id] of FILTERS) assert.ok(accounted.has(id), `${id} is placed somewhere in the new layout`)
  assert.equal(accounted.size, FILTERS.length)
})
