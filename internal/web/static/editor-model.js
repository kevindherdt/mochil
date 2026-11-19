// The editor's rules, apart from the page: what an edit is, and how bulk actions turn a selection into edits.
// An edit is { [bookId]: { [field]: value } }. An empty value means "back to what the files say".

// Every column the grid can show. "field" is set for the ones that can be edited; "sort" is the server's name for the
// order. Columns marked "on" are shown until the person chooses otherwise, the rest are in the Columns menu.
const BASE_COLUMNS = [
  { id: "title", label: "Title", field: "title", sort: "title", on: true },
  { id: "authors", label: "Author", field: "authors", sort: "author", on: true },
  { id: "series", label: "Series", field: "series", sort: "series", on: true },
  { id: "series_index", label: "#", field: "series_index", sort: "number", on: true, num: true },
  { id: "tags", label: "Tags", field: "tags", sort: "tags" },
  { id: "publisher", label: "Publisher", field: "publisher", sort: "publisher" },
  { id: "published", label: "Published", field: "published", sort: "published" },
  { id: "language", label: "Language", field: "language", sort: "language" },
  { id: "rating", label: "Rating", field: "rating", sort: "rating", num: true },
  { id: "isbn", label: "ISBN", field: "isbn", sort: "isbn" },
  { id: "identifiers", label: "Identifiers", field: "identifiers", sort: "identifiers" },
  { id: "title_sort", label: "Title sort", field: "title_sort", sort: "title_sort" },
  { id: "author_sort", label: "Author sort", field: "author_sort", sort: "author_sort" },
  { id: "description", label: "Description", field: "description", sort: "description" },
  { id: "format", label: "Format", sort: "format", on: true },
  { id: "collections", label: "Collections", sort: "collections" },
  { id: "hardcover", label: "Hardcover", sort: "hardcover" },
  { id: "path", label: "File", sort: "path" },
  { id: "folder", label: "Folder", sort: "path" },
  { id: "library_folder", label: "Library folder", sort: "library_folder" },
  { id: "status", label: "Status", sort: "status", on: true },
];

// Progress is read-only and comes from the signed-in person's reading. It sits before Format and Status.
const PROGRESS_COLUMN = { id: "progress", label: "Progress", sort: "progress", num: true };

// Words and Pages come from counting the books (Tools > Count words and pages). Read-only, and off until chosen.
const WORDS_COLUMN = { id: "words", label: "Words", sort: "words", num: true };
const PAGES_COLUMN = { id: "pages", label: "Pages", sort: "pages", num: true };
// The ISBN read out of the book itself (Tools > Find ISBNs): read-only, and taken into the ISBN column only on request.
const FOUND_ISBN_COLUMN = { id: "isbn_found", label: "Found ISBN", sort: "isbn_found" };
// The Hardcover book found for this one (Tools > Find Hardcover matches): read-only, taken only on request.
const MATCH_COLUMN = { id: "match", label: "Hardcover match", sort: "match" };
// folderOf is the folder a book sits in, inside its library ("Author/Title (12)"); empty for a book at the top.
export const folderOf = (row) => String(row.path || "").split("/").slice(0, -1).join("/");
export const countText = (n) => (n > 0 ? n.toLocaleString("en") : "");

// COLUMNS and FIELDS are the base columns plus the person's own; setCustomColumns refreshes them in place, so the
// page and the rules always read the same list.
export const COLUMNS = [];
export const FIELDS = [];
export function setCustomColumns(definitions = []) {
  const custom = definitions.map((d) => ({ id: d.id, label: d.label, field: d.id, sort: d.id, custom: true, type: d.type, num: d.type === "number" }));
  const base = BASE_COLUMNS.slice();
  const at = base.findIndex((column) => column.id === "format");
  COLUMNS.length = 0;
  COLUMNS.push(...base.slice(0, at), ...custom, PROGRESS_COLUMN, WORDS_COLUMN, PAGES_COLUMN, FOUND_ISBN_COLUMN, MATCH_COLUMN, ...base.slice(at));
  FIELDS.length = 0;
  FIELDS.push(...COLUMNS.filter((column) => column.field && column.field !== "isbn").map((column) => column.field)); // ISBN is part of the identifiers on the server
}
setCustomColumns();
export const defaultColumns = () => COLUMNS.filter((column) => column.on).map((column) => column.id);

// visibleColumns keeps the order of COLUMNS and ignores ids it does not know, so an old saved choice cannot break the grid.
export const visibleColumns = (ids) => COLUMNS.filter((column) => ids.includes(column.id));

// The ISBN column is the "isbn:" entry of the identifiers; editing it changes only that entry and leaves the rest.
const identifierParts = (text) => String(text || "").split(",").map((part) => part.trim()).filter(Boolean);
export const isbnOf = (row) => (identifierParts(row.identifiers).find((part) => /^isbn:/i.test(part)) || "").replace(/^isbn:\s*/i, "");
export function withIsbn(identifiers, isbn) {
  const parts = identifierParts(identifiers), at = parts.findIndex((part) => /^isbn:/i.test(part));
  if (at >= 0) { if (isbn) parts[at] = `isbn:${isbn}`; else parts.splice(at, 1); } // stays where it was, so an unchanged ISBN is no edit
  else if (isbn) parts.push(`isbn:${isbn}`);
  return parts.join(", ");
}
const isbnDigits = (text) => String(text || "").replace(/[\s-]/g, "").toUpperCase();

// isValidIsbn checks the length and the check digit, so a mistyped digit is caught instead of saved.
export function isValidIsbn(text) {
  const digits = isbnDigits(text);
  if (/^\d{13}$/.test(digits)) return [...digits].reduce((sum, d, i) => sum + Number(d) * (i % 2 ? 3 : 1), 0) % 10 === 0;
  if (/^\d{9}[\dX]$/.test(digits)) return [...digits].reduce((sum, d, i) => sum + (d === "X" ? 10 : Number(d)) * (10 - i), 0) % 11 === 0;
  return false;
}

export function fieldValue(row, field) {
  if (field === "isbn") return isbnOf(row);
  if (field.startsWith("#")) return (row.custom || {})[field] || "";
  if (field === "authors") return (row.authors || []).join(" & ");
  if (field === "tags") return (row.tags || []).join(", ");
  if (field === "series_index") return row.series_index == null ? "" : String(row.series_index);
  if (field === "rating") return row.rating ? String(row.rating) : "";
  return row[field] || "";
}

// stars shows a rating as ★★★★½; empty when there is none.
export function stars(rating) {
  const whole = Math.floor(rating || 0);
  return rating ? "★".repeat(whole) + (rating - whole >= 0.5 ? "½" : "") : "";
}

// shown is what a cell displays, which may be shorter or prettier than what editing it starts from.
export function shown(row, field) {
  if (field.startsWith("#") && (COLUMNS.find((c) => c.id === field) || {}).type === "yesno") return { yes: "✓ yes", no: "no" }[fieldValue(row, field)] || "";
  if (field === "rating") return stars(row.rating);
  const value = fieldValue(row, field);
  if (field === "published") return value.replace(/T.*$/, "");
  return field === "description" && value.length > 140 ? `${value.slice(0, 140)}…` : value;
}

export function validate(field, value) {
  const text = String(value ?? "").trim();
  if (field.startsWith("#")) {
    const type = (COLUMNS.find((column) => column.id === field) || {}).type;
    if (type === "number" && text && !Number.isFinite(Number(text))) return "This column takes a number.";
    if (type === "yesno" && text && !/^(yes|y|true|1|no|n|false|0)$/i.test(text)) return "This column takes yes or no.";
  }
  if (field === "title" && !text) return "A title cannot be empty.";
  if (field === "isbn" && text && !isValidIsbn(text)) return "An ISBN has 10 or 13 digits and a correct check digit.";
  if (field === "series_index" && text) {
    const number = Number(text);
    if (!Number.isFinite(number) || number <= 0) return "A volume number must be a positive number.";
  }
  if (field === "rating" && text) {
    const number = Number(text);
    if (!Number.isFinite(number) || number < 0 || number > 5) return "A rating is between 0 and 5 stars.";
  }
  if (field === "identifiers" && text) {
    for (const part of text.split(",")) {
      const [scheme, ...rest] = part.trim().split(":");
      if (!scheme.trim() || !rest.join(":").trim() || /[\s<>&"'/]/.test(scheme.trim())) return "Identifiers look like isbn:978…, hardcover:dune (separate several with commas).";
    }
  }
  return "";
}

// stage records one edit, or forgets it when the value is back to what the row already says.
export function stage(pending, row, field, value) {
  if (field === "isbn") {
    const current = (pending[row.id] || {}).identifiers ?? row.identifiers;
    return stage(pending, row, "identifiers", withIsbn(current, isbnDigits(value)));
  }
  const next = { ...pending };
  const fields = { ...(next[row.id] || {}) };
  const text = String(value ?? "").trim();
  if (text === fieldValue(row, field)) delete fields[field];
  else fields[field] = text;
  if (Object.keys(fields).length) next[row.id] = fields;
  else delete next[row.id];
  return next;
}

export function merge(pending, changes, rows) {
  const byId = new Map(rows.map((row) => [row.id, row]));
  let next = pending;
  for (const [id, fields] of Object.entries(changes)) {
    const row = byId.get(id);
    if (!row) continue;
    for (const [field, value] of Object.entries(fields)) next = stage(next, row, field, value);
  }
  return next;
}

// withPending is the row as it would look once the pending edits are saved.
export function withPending(row, pending) {
  const fields = pending[row.id];
  if (!fields) return row;
  const out = { ...row };
  for (const [field, value] of Object.entries(fields)) {
    if (field === "authors") out.authors = value ? value.split(" & ").map((name) => name.trim()).filter(Boolean) : row.authors;
    else if (field === "series_index") out.series_index = value === "" ? null : Number(value);
    else if (field === "tags") out.tags = value.split(",").map((tag) => tag.trim()).filter(Boolean);
    else if (field === "rating") out.rating = value === "" ? 0 : Number(value);
    else if (field.startsWith("#")) { out.custom = { ...(row.custom || {}) }; if (value) out.custom[field] = value; else delete out.custom[field]; }
    else out[field] = value;
  }
  return out;
}

export const pendingCount = (pending) => Object.values(pending).reduce((total, fields) => total + Object.keys(fields).length, 0);

export function setAll(rows, field, value) {
  return Object.fromEntries(rows.map((row) => [row.id, { [field]: String(value).trim() }]));
}

// renumber gives the rows, in the order given, the numbers start, start+step, ...
export function renumber(rows, start = 1, step = 1) {
  const clean = (number) => String(Math.round(number * 1000) / 1000);
  return Object.fromEntries(rows.map((row, index) => [row.id, { series_index: clean(start + index * step) }]));
}

// replaceText swaps text inside a field. With prefixOnly, only text at the start is removed or swapped, which is what
// stripping "Series - " from titles needs.
export function replaceText(rows, field, find, replacement = "", { prefixOnly = false, matchCase = false } = {}) {
  if (!find) return {};
  const changes = {};
  const flags = matchCase ? "g" : "gi";
  const escaped = find.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const pattern = new RegExp(prefixOnly ? `^${escaped}` : escaped, prefixOnly ? (matchCase ? "" : "i") : flags);
  for (const row of rows) {
    const before = fieldValue(row, field);
    const after = before.replace(pattern, () => replacement).replace(/\s{2,}/g, " ").trim();
    if (after !== before) changes[row.id] = { [field]: after };
  }
  return changes;
}

export function request(library, pending, { preview = false, reset = [] } = {}) {
  return { library, changes: pending, reset, preview };
}

// A short sentence for a status badge; the order is the order a person should fix things in.
const ISSUE_TEXT = {
  duplicate: "Duplicate number",
  no_author: "No author",
  no_number: "In a series but has no number",
  side_story: "Side story",
  no_description: "No description",
  no_isbn: "No ISBN",
  no_cover: "No cover",
  unreadable: "File will not open",
  isbn_found: "ISBN found in the book",
  isbn_unsure: "ISBN: several, not sure which",
  match_found: "Hardcover match found",
  match_unsure: "Hardcover match unsure",
  hardcover_doubt: "Hardcover match to check",
  hardcover_nomatch: "Not on Hardcover",
  hardcover_ignored: "Left out of Hardcover sync",
  match_none: "Not on Hardcover",
};
export const issueText = (issue) => ISSUE_TEXT[issue] || issue;
// Filters that stay out of sight until the quality check has found something for them.
export const FOUND_BY_CHECK = new Set(["no_cover", "unreadable", "isbn_found", "isbn_unsure", "match_found", "match_unsure", "match_none", "hardcover_doubt", "hardcover_nomatch", "hardcover_ignored"]);
export const FILTERS = [
  ["all", "All"],
  ["issues", "Needs review"],
  ["no_author", "No author"],
  ["no_number", "No number"],
  ["duplicate", "Duplicates"],
  ["side_story", "Side stories"],
  ["no_description", "No description"],
  ["no_isbn", "No ISBN"],
  ["no_cover", "No cover"],
  ["unreadable", "Will not open"],
  ["isbn_found", "ISBN found"],
  ["isbn_unsure", "ISBN unsure"],
  ["match_found", "Match found"],
  ["match_unsure", "Match unsure"],
  ["hardcover_doubt", "Hardcover: check"],
  ["hardcover_nomatch", "Not on Hardcover"],
  ["hardcover_ignored", "Hardcover: left out"],
  ["match_none", "No match"],
  ["edited", "Edited"],
];

// The filter bar used to be one chip per FILTERS entry, 14+ wide and crammed with zero-count
// chips nobody could use yet. All and Needs review stay pinned in the open bar (the two most
// reached for); everything else lives behind a "Filters" menu, grouped by what it is about, and
// a filter with nothing to show (and that is not the active one) is left out of the menu entirely.
export const FILTER_PINNED = ["all", "issues"];
export const FILTER_GROUPS = [
  { label: "Data quality", filters: ["no_author", "no_number", "duplicate", "side_story", "no_description", "no_cover", "unreadable"] },
  { label: "ISBN", filters: ["no_isbn", "isbn_found", "isbn_unsure"] },
  { label: "Match", filters: ["match_found", "match_unsure", "hardcover_doubt", "hardcover_nomatch", "hardcover_ignored", "match_none"] },
];
// Anything pinned or grouped is not repeated here; today that leaves only "Edited".
export const FILTER_OTHER = FILTERS.map(([id]) => id).filter((id) => !FILTER_PINNED.includes(id) && !FILTER_GROUPS.some((group) => group.filters.includes(id)));

// filterCount gives the number shown on a filter's chip: the library total for "all", the sum of
// the three classic issue kinds for "issues", and the server's own count for everything else.
export function filterCount(id, state) {
  if (id === "all") return state.libraryTotal;
  if (id === "issues") return ["duplicate", "no_author", "no_number"].reduce((n, k) => n + (state.counts[k] || 0), 0);
  return state.counts[id] || 0;
}

const filterLabels = () => Object.fromEntries(FILTERS);

function filterButtonHTML(id, state) {
  const count = filterCount(id, state);
  if (!FILTER_PINNED.includes(id) && !count && state.filter !== id) return ""; // nothing to show, and not already chosen
  return `<button type="button" data-filter="${id}" aria-pressed="${state.filter === id}">${filterLabels()[id]}<small>${count}</small></button>`;
}

// filterBarHTML renders the whole bar: the pinned chips, then a "Filters" dropdown with the rest
// grouped under a heading each, carrying a badge while the active filter lives inside it.
export function filterBarHTML(state) {
  const pinned = FILTER_PINNED.map((id) => filterButtonHTML(id, state)).join("");
  const groups = FILTER_GROUPS.map((group) => {
    const buttons = group.filters.map((id) => filterButtonHTML(id, state)).join("");
    return buttons ? `<p class="ed-filters-group-label">${group.label}</p>${buttons}` : "";
  }).join("");
  const other = FILTER_OTHER.map((id) => filterButtonHTML(id, state)).join("");
  const activeInMenu = [...FILTER_GROUPS.flatMap((group) => group.filters), ...FILTER_OTHER].includes(state.filter);
  const summary = `Filters${activeInMenu ? '<small class="ed-filters-badge">1</small>' : ""}`;
  return `${pinned}<details class="ed-columns ed-filters-menu" id="ed-filters-menu"${activeInMenu ? " open" : ""}><summary>${summary}</summary><div class="ed-columns-list ed-filters-groups">${groups}${other}</div></details>`;
}

// ---- matching a book to a candidate from a metadata source -------------------------------------------------------

const withIdentifier = (current, scheme, id) => [...String(current || "").split(",").map((part) => part.trim()).filter((part) => part && !part.toLowerCase().startsWith(`${scheme}:`)), `${scheme}:${id}`].join(", ");

// matchFields lists what a candidate could change on a book: only fields where it says something different. The person
// ticks which to take; nothing is taken for them except a link, which is never wrong to add.
export function matchFields(row, candidate) {
  const fields = [];
  const add = (field, label, from, to) => { if (to && to !== from) fields.push({ field, label, from: from || "", to }); };
  add("title", "Title", row.title, candidate.title);
  add("authors", "Author", row.authors, (candidate.authors || []).join(" & "));
  if (!(candidate.year && String(row.published || "").startsWith(String(candidate.year)))) add("published", "Published", row.published, candidate.year ? String(candidate.year) : "");
  add("description", "Description", row.description, candidate.overview);
  if (candidate.provider === "Hardcover" || candidate.provider === "hardcover") {
    add("identifiers", "Hardcover link", row.identifiers, candidate.provider_id ? withIdentifier(row.identifiers, "hardcover", candidate.provider_id) : "");
  }
  return fields;
}

// fillFromMatch is what taking a certain match does to a book in bulk: it fills the fields the book has nothing in
// (a description, a year) and adds the Hardcover link. It never replaces anything the book already says.
export function fillFromMatch(row) {
  if (!row.match || !row.match_clear) return {};
  return Object.fromEntries(matchFields(row, row.match).filter((f) => f.field === "identifiers" || !f.from).map((f) => [f.field, f.to]));
}

export const matchChanges = (fields, chosen) => Object.fromEntries(fields.filter((f) => chosen.includes(f.field)).map((f) => [f.field, f.to]));

export const scoreText = (score) => `${Math.round(Math.max(0, Math.min(1, score)) * 100)}%`;

// progressText is what the Progress cell says: a percentage, "Finished", or nothing for an unread book.
export const progressText = (row) => (row.progress >= 1 ? "Finished" : row.progress > 0 ? `${Math.max(1, Math.round(row.progress * 100))}%` : "");
