// The import page's rules, apart from the page: what a row looks like once edited, which books can be copied, and
// what each sidecar says.

export const STATUS_TEXT = {
  new: "New",
  new_volume: "New volume",
  possible: "Possible duplicate",
  in_library: "Already in library",
};

// [id, label, the status list the server filters by]
export const STATUS_FILTERS = [
  ["fresh", "New", "new,new_volume"],
  ["new", "New series", "new"],
  ["new_volume", "New volumes", "new_volume"],
  ["possible", "Possible duplicates", "possible"],
  ["in_library", "Already in library", "in_library"],
  ["all", "Everything", ""],
];
export const statusesFor = (filterId) => (STATUS_FILTERS.find(([id]) => id === filterId) || STATUS_FILTERS[0])[2];

// view is the book as it will be imported: the guess, with the person's edits on top.
export function view(row, edits = {}) {
  const proposal = row.proposal || {};
  const edit = edits[row.source_path] || {};
  return {
    title: "title" in edit ? edit.title : proposal.title || "",
    authors: "authors" in edit ? edit.authors : proposal.authors || [],
    series: "series" in edit ? edit.series : proposal.series || "",
    series_index: "series_index" in edit ? edit.series_index : proposal.series_index ?? null,
  };
}

export const targetOf = (row, targets = {}) => (row.source_path in targets ? targets[row.source_path] : row.suggested_target || "");

export const chunk = (list, size) => {
  const out = [];
  for (let i = 0; i < list.length; i += size) out.push(list.slice(i, i + size));
  return out;
};

// destinationOf is the library a book goes to: the one chosen for that book, otherwise the page's default. "" is the
// managed folder itself. An id that is no longer a destination falls back to the default.
export function destinationOf(row, choices = {}, fallback = "", destinations = []) {
  const known = (id) => id === "" || destinations.some((d) => d.id === id);
  const own = choices[row.source_path];
  return own !== undefined && known(own) ? own : known(fallback) ? fallback : "";
}

// entriesFor pairs each chosen book with where it will go; resolve(row) gives the full path. A book with no target
// cannot be copied, so it is set aside instead of being given an invented name.
export function entriesFor(rows, resolve) {
  const entries = [], unnamed = [];
  for (const row of rows) {
    const target = resolve(row);
    if (target) entries.push({ source_path: row.source_path, target_path: target });
    else unnamed.push(row.source_path);
  }
  return { entries, unnamed };
}

// tally counts books per destination, in the order they first appear, for the review: "12 to Books Two, 3 to ..."
export function tally(rows, destinationFor, nameOf) {
  const counts = new Map();
  for (const row of rows) {
    const name = nameOf(destinationFor(row));
    counts.set(name, (counts.get(name) || 0) + 1);
  }
  return [...counts].map(([name, count]) => ({ name, count }));
}

// readyRows picks which chosen rows the server confirmed are ready to copy, by source_path. A Set lookup
// keeps this linear even for a large batch (hundreds-to-thousands of files from a drop-folder scan),
// instead of re-scanning the whole `ready` array for every row in `chosen`.
export function readyRows(chosen, ready) {
  const readyPaths = new Set(ready.map((entry) => entry.source_path));
  return chosen.filter((row) => readyPaths.has(row.source_path));
}

const ISSUE_TEXT = {
  target_exists: "A file is already saved there",
  duplicate_target: "Another book in this batch would be saved there",
  duplicate_source: "Listed twice",
  source_unavailable: "The file in the drop folder is missing or not a regular file",
  source_too_large: "The file is larger than the allowed size",
  unsafe_path: "The name is not allowed",
  target_unavailable: "The destination could not be checked",
};
export const issueMessage = (issue) => ISSUE_TEXT[issue.code] || issue.message || issue.code;

// splitPreview divides a server preview into books that are ready and books that something blocks.
export function splitPreview(preview) {
  const ready = [], blocked = [];
  for (const entry of preview.entries || []) {
    if (entry.ready) ready.push({ source_path: entry.source_path, target_path: entry.target_path });
    else blocked.push({ source_path: entry.source_path, target_path: entry.target_path, reasons: (entry.issues || []).map(issueMessage) });
  }
  return { ready, blocked };
}

// sidecarFor is what the metadata.opf of an imported book says: the edited fields plus identifiers the book carried.
export function sidecarFor(row, edits = {}) {
  const book = view(row, edits);
  return {
    title: book.title,
    authors: book.authors,
    series: book.series,
    series_index: book.series_index,
    identifiers: (row.proposal && row.proposal.identifiers) || {},
  };
}

export function validateEdit(field, value) {
  const text = String(value ?? "").trim();
  if (field === "title" && !text) return "A title cannot be empty.";
  if (field === "series_index" && text) {
    const number = Number(text);
    if (!Number.isFinite(number) || number <= 0) return "A volume number must be a positive number.";
  }
  return "";
}

export function parseEdit(field, value) {
  const text = String(value ?? "").trim();
  if (field === "authors") return text.split(/\s+&\s*|\s*&\s+/).map((name) => name.trim()).filter(Boolean);
  if (field === "series_index") return text === "" ? null : Number(text);
  return text;
}

export function fieldText(book, field) {
  if (field === "authors") return book.authors.join(" & ");
  if (field === "series_index") return book.series_index == null ? "" : String(book.series_index);
  return book[field] || "";
}

// withoutRepeatedTargets keeps the first book for each saved name and blocks the rest, since the server only compares
// books within one batch and an import may span several.
export function withoutRepeatedTargets(entries) {
  const seen = new Set(), ready = [], blocked = [];
  for (const entry of entries) {
    const key = `${entry.destination || ""}\n${entry.target_path.toLowerCase()}`;
    if (seen.has(key)) blocked.push({ ...entry, reasons: [issueMessage({ code: "duplicate_target" })] });
    else { seen.add(key); ready.push(entry); }
  }
  return { ready, blocked };
}

// rangeBetween lists the ids from one click to another, in the order shown, for shift-click selection. Without a usable
// anchor it is just the clicked id.
export function rangeBetween(ids, anchor, id) {
  const a = ids.indexOf(anchor), b = ids.indexOf(id);
  if (b < 0) return [];
  if (a < 0) return [id];
  return ids.slice(Math.min(a, b), Math.max(a, b) + 1);
}

// Columns the import list can be ordered by. "target" and "source" are paths; the rest read from the edited book.
export const SORT_KEYS = ["status", "title", "authors", "series", "series_index", "target", "source"];
const STATUS_ORDER = { new: 0, new_volume: 1, possible: 2, in_library: 3 };

// sortRows returns the rows ordered by a column. Books with nothing in that column stay last whichever way it is
// sorted, and ties keep the order the folder was read in, so the list never shuffles.
export function sortRows(rows, key, direction, edits = {}, targets = {}) {
  if (!SORT_KEYS.includes(key)) return [...rows].sort((a, b) => a.order - b.order);
  const read = (row) => {
    const book = view(row, edits);
    switch (key) {
      case "status": return STATUS_ORDER[row.status] ?? 9;
      case "title": return book.title;
      case "authors": return book.authors[0] || "";
      case "series": return book.series;
      case "series_index": return book.series_index;
      case "target": return targetOf(row, targets);
      default: return row.source_path;
    }
  };
  const empty = (value) => value === "" || value === null || value === undefined;
  const sign = direction === "desc" ? -1 : 1;
  return [...rows].sort((a, b) => {
    const x = read(a), y = read(b);
    if (empty(x) !== empty(y)) return empty(x) ? 1 : -1;
    let c = 0;
    if (!empty(x)) c = typeof x === "number" && typeof y === "number" ? x - y : String(x).localeCompare(String(y), undefined, { numeric: true, sensitivity: "base" });
    return c ? c * sign : a.order - b.order;
  });
}

export const nextSort = (current, key) => (current.key !== key ? { key, dir: "asc" } : current.dir === "asc" ? { key, dir: "desc" } : { key: "", dir: "asc" });

// groupByDestination splits entries by library, in the order each first appears, and drops the client-only
// "destination" key so what is sent is exactly source_path and target_path.
export function groupByDestination(entries) {
  const groups = new Map();
  for (const entry of entries) {
    const key = entry.destination || "";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push({ source_path: entry.source_path, target_path: entry.target_path });
  }
  return [...groups];
}

// Cells are rebuilt wholesale, so re-rendering while one is being edited removes the input the person
// is typing into: focus is lost without a blur, the editor never finishes, and editing stays switched
// off for the rest of the page. A render asked for during an edit is held until the edit ends.
export function shouldDeferRender(state) {
  return Boolean(state && state.editing);
}
