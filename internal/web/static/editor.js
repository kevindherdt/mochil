import { COLUMNS, countText, progressText, setCustomColumns, matchChanges, matchFields, scoreText, filterBarHTML, defaultColumns, fieldValue, issueText, merge, pendingCount, renumber, replaceText, request, setAll, shown as cellText, stage, validate, visibleColumns as baseVisibleColumns, withPending, isbnOf, fillFromMatch, folderOf } from "./editor-model.js?v=17";

import { blockedText, confirmText, doneText, transferRequest } from "./editor-transfer.js?v=1";
import { attachSuggestions } from "./field-suggestions.js?v=3";
import { mountTools, resumeJobs } from "./editor-tools.js?v=13";
import { keepInView } from "./menu-in-view.js?v=2";
import { fitHeight } from "./fit-height.js?v=2";
import { bindResize, gridWidth, readWidths, widthOf } from "./column-widths.js?v=8";
import { rowHTML, spacerHTML } from "./editor-rows.js?v=9";
import { applyOrder, mountColumnDrag } from "./editor-column-order.js?v=2";
import { collectionsDialog, fillCollectionFilter } from "./editor-collections.js?v=9";
import { manageCollections } from "./editor-collections-manage.js?v=2";
import { hardcoverSyncDialog } from "./editor-hardcover.js?v=3";
import { scrollTopFor, stillCovers, windowFor } from "./virtual-rows.js?v=2";
import { moduleEnabled } from "./module-enabled.js?v=1";

const $ = (selector) => document.querySelector(selector);
const PUBLICATION_KINDS = new Set(["books", "comics", "manga", "mixed"]);
const PAGE = 1000; // rows asked for at a time; all of them are kept, only the visible ones are drawn
const escapeHTML = (value) => String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const remember = (key, value) => { try { localStorage.setItem(key, value); } catch (_) {} };
const recall = (key) => { try { return localStorage.getItem(key); } catch (_) { return null; } };

// The same palette and light/dark choice as the rest of the app.
const root = document.documentElement;
const mode = recall("mochil-theme-mode") || recall("mochil-theme") || "system";
root.dataset.palette = recall("mochil-palette") || "mochil";
root.dataset.theme = mode === "system" ? (matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark") : mode;

const state = {
  libraries: [], library: "", libraryFolder: "", collection: "", collections: [], hardcoverSync: false, filter: "all", q: "", rows: [], total: 0, counts: {}, libraryTotal: 0,
  sort: "", dir: "asc", pending: {}, resets: new Set(), selected: new Set(), anchor: null, focus: { row: 0, col: 0 }, editing: false, undo: "",
  density: recall("mochil-editor-density") === "comfortable" ? "comfortable" : "dense",
  widths: readWidths(recall("mochil-editor-widths")),
  columns: savedColumns(), order: savedOrder(),
  transferTo: [], undoRescan: [],
};

function savedColumns() {
  try {
    const ids = JSON.parse(recall("mochil-editor-columns") || "null");
    if (Array.isArray(ids) && baseVisibleColumns(ids).some((column) => column.field)) return ids;
  } catch (_) {}
  return defaultColumns();
}
function savedOrder() {
  try { const ids = JSON.parse(recall("mochil-editor-column-order") || "[]"); return Array.isArray(ids) ? ids : []; } catch (_) { return []; }
}
function visibleColumns(ids) { return applyOrder(baseVisibleColumns(ids), state.order); }
// The editable columns that are showing, in order: the keyboard moves between these.
const editableColumns = () => visibleColumns(state.columns).filter((column) => column.field);

async function api(path, options) {
  const response = await fetch(path, options && { ...options, headers: { "Content-Type": "application/json" }, body: JSON.stringify(options.body) });
  if (!response.ok) throw new Error((await response.text()).trim() || `HTTP ${response.status}`);
  return response.json();
}

const shown = (row) => withPending(row, state.pending);
// kind "problem" paints the line as something to act on (a tool that cannot start), not as passing progress.
const status = (text, kind) => { const line = $("#ed-status"); line.textContent = text; line.classList.toggle("problem", kind === "problem"); };

async function loadRows(append = false) {
  // Every page is fetched in order, then applied. Without a ticket, a slow load started before the
  // person switched library or changed the search lands last and quietly paints the wrong rows.
  const generation = (state.loadGeneration = (state.loadGeneration || 0) + 1);
  const target = Infinity;
  const rows = [];
  let reply;
  try {
    while (rows.length < target) {
      const query = new URLSearchParams({ library: state.library, filter: state.filter, q: state.q, sort: state.sort, dir: state.dir, collection: state.collection, offset: String(rows.length), limit: String(PAGE) });
      reply = await api(`/api/metadata/editor/books?${query}`);
      rows.push(...reply.items);
      if (!reply.items.length || rows.length >= reply.total) break;
    }
  } catch (error) {
    status(`Could not load books: ${error.message}`);
    return;
  }
  if (generation !== state.loadGeneration) return; // a newer load owns the screen now
  setCustomColumns(reply?.columns || []);
  state.rows = rows;
  state.total = reply?.total ?? 0;
  state.counts = reply?.counts ?? {};
  state.libraryFolder = reply?.library_folder ?? "";
  state.collections = reply?.collections ?? [];
  state.hardcoverSync = Boolean(reply?.hardcover_sync);
  if (state.collection && !state.collections.some((c) => c.id === state.collection)) state.collection = ""; // it was deleted
  state.libraryTotal = reply?.library_total ?? 0;
  state.transferTo = reply?.transfer_to ?? [];
  const ids = new Set(rows.map((row) => row.id));
  state.selected = new Set([...state.selected].filter((id) => ids.has(id)));
  render();
}

const fitTable = fitHeight($(".ed-scroll"));
const RESIZE = '<span class="ed-resize" data-resize title="Drag to resize; double-click to reset" aria-hidden="true"></span>';
bindResize({ head: $("#ed-head"), grid: $("#ed-grid"), widths: state.widths, ids: () => visibleColumns(state.columns).map((column) => column.id), comfortable: () => state.density === "comfortable", onChange: (widths) => remember("mochil-editor-widths", JSON.stringify(widths)) });

// Only the rows in view are drawn; spacers above and below stand in for the rest (see virtual-rows.js).
let drawn = null;
let rowHeight = 0;
function drawRows(force = false) {
  const box = $(".ed-scroll");
  const head = $("#ed-head").offsetHeight;
  const view = { scrollTop: box.scrollTop, viewport: Math.max(0, box.clientHeight - head), rowHeight: rowHeight || (state.density === "comfortable" ? 85 : 33), count: state.rows.length };
  if (!force && drawn && stillCovers(drawn, view)) return;
  const columns = visibleColumns(state.columns);
  const editable = editableColumns();
  const part = windowFor(view);
  const wide = columns.length + (state.density === "comfortable" ? 3 : 2); // checkbox, cover (comfortable only), the columns, the blank end
  $("#ed-body").innerHTML = spacerHTML(part.before, wide) +
    state.rows.slice(part.start, part.end).map((row, offset) => rowHTML(row, shown(row), part.start + offset, { columns, editable, state })).join("") +
    spacerHTML(part.after, wide);
  drawn = part;
  const first = $("#ed-body tr[data-id]");
  if (first && first.offsetHeight && first.offsetHeight !== rowHeight) { rowHeight = first.offsetHeight; drawn = null; drawRows(true); }
}
$(".ed-scroll").addEventListener("scroll", () => { if (!state.editing) drawRows(); }, { passive: true });

function render() {
  document.body.classList.toggle("ed-comfortable", state.density === "comfortable");
  document.querySelectorAll("[data-density]").forEach((button) => button.setAttribute("aria-pressed", String(button.dataset.density === state.density)));
  $("#ed-filters").innerHTML = filterBarHTML(state);
  const columns = visibleColumns(state.columns);
  const editable = editableColumns();
  $("#ed-head").innerHTML = '<th class="ed-check"><input type="checkbox" id="ed-all" aria-label="Select all loaded books"></th><th class="ed-cover">Cover</th>' +
    columns.map((column) => column.sort
      ? `<th class="${column.num ? "ed-num" : ""}" data-col-id="${column.id}" style="width:${widthOf(column.id, state.widths)}px" aria-sort="${state.sort === column.sort ? (state.dir === "desc" ? "descending" : "ascending") : "none"}"><button type="button" class="ed-sort" data-sort="${column.sort}">${escapeHTML(column.label)}<span class="ed-arrow" aria-hidden="true"></span></button>${RESIZE}</th>`
      : `<th data-col-id="${column.id}" style="width:${widthOf(column.id, state.widths)}px">${escapeHTML(column.label)}${RESIZE}</th>`).join("") + '<th class="ed-fill" aria-hidden="true"></th>';
  $("#ed-grid").style.width = `${gridWidth(columns.map((column) => column.id), state.widths, state.density === "comfortable")}px`;
  $("#ed-all").checked = state.selected.size > 0 && state.selected.size === state.rows.length;
  $("#ed-all").onchange = (event) => { state.selected = event.target.checked ? new Set(state.rows.map((row) => row.id)) : new Set(); render(); };
  $("#ed-columns-list").innerHTML = applyOrder(COLUMNS, state.order).map((column) => `<label draggable="true" data-column-row="${escapeHTML(column.id)}"><span class="ed-grip" aria-hidden="true">⋮⋮</span><input type="checkbox" data-column="${escapeHTML(column.id)}" ${state.columns.includes(column.id) ? "checked" : ""}> ${escapeHTML(column.label)}${column.custom ? `<button type="button" class="ed-quiet ed-remove-column" data-remove-column="${escapeHTML(column.id)}" aria-label="Remove column ${escapeHTML(column.label)}" title="Remove this column">×</button>` : ""}</label>`).join("") + '<button type="button" class="ed-add-column" data-add-column>＋ Add a column…</button>';
  drawn = null;
  drawRows();
  fillCollectionFilter($("#ed-collection"), state.collections, state.collection);
  status(state.total ? `Showing ${state.rows.length} of ${state.total} books` : "No books match.");
  const count = state.selected.size;
  $("#ed-selection").hidden = count === 0;
  $("#ed-selected-count").textContent = `${count} selected`;
  $("#ed-move").hidden = $("#ed-copy").hidden = !state.transferTo.length;
  $("#ed-match").disabled = $("#ed-open").disabled = count !== 1;
  $("#ed-hardcover").hidden = !state.hardcoverSync;
  const edits = pendingCount(state.pending) + state.resets.size;
  $("#ed-save").hidden = edits === 0;
  $("#ed-save-text").textContent = `${edits} unsaved ${edits === 1 ? "change" : "changes"}`;
  fitTable(); // what is above the table may have changed size
}

// ---- editing a cell -------------------------------------------------------------------------------------------
function cellAt(row, col) { return document.querySelector(`td[data-row="${row}"][data-col="${col}"]`); }

function moveFocus(dRow, dCol) {
  const row = Math.min(state.rows.length - 1, Math.max(0, state.focus.row + dRow));
  const col = Math.min(editableColumns().length - 1, Math.max(0, state.focus.col + dCol));
  state.focus = { row, col };
  document.querySelectorAll('td[data-focus="true"]').forEach((cell) => { cell.dataset.focus = "false"; });
  const box = $(".ed-scroll");
  const header = $("#ed-head").offsetHeight;
  const top = scrollTopFor(row, { scrollTop: box.scrollTop, viewport: box.clientHeight, rowHeight, header });
  if (top !== box.scrollTop) { box.scrollTop = top; drawRows(); }
  const cell = cellAt(row, col);
  if (cell) { cell.dataset.focus = "true"; cell.scrollIntoView({ block: "nearest", inline: "nearest" }); }
}

// Fields that suggest what the library already holds, and what separates several values in one cell.
const SUGGESTED = { authors: "&", tags: ",", series: "", publisher: "", language: "" };

function startEdit(initial) {
  const cell = cellAt(state.focus.row, state.focus.col);
  const row = state.rows[state.focus.row];
  if (!cell || !row || state.editing) return;
  const field = editableColumns()[state.focus.col]?.field;
  if (!field) return;
  const input = document.createElement("input");
  input.className = "ed-cell";
  input.type = "text";
  input.value = initial ?? fieldValue(shown(row), field);
  input.setAttribute("aria-label", `Edit ${field.replace("_", " ")}`);
  if (SUGGESTED[field]) {
    const library = state.library;
    attachSuggestions({
      input, floating: true, separator: SUGGESTED[field],
      fetchItems: async (text) => (await (await fetch(`/api/metadata/editor/values?library=${encodeURIComponent(library)}&field=${field}&q=${encodeURIComponent(text)}`, { credentials: "same-origin" })).json()),
    });
  }
  if (field === "series_index") input.inputMode = "decimal";
  cell.textContent = "";
  cell.append(input);
  state.editing = true;
  input.focus();
  if (initial === undefined) input.select();
  let finished = false;
  const finish = (commit, dRow = 0, dCol = 0) => {
    if (finished) return true;
    if (commit) {
      const problem = validate(field, input.value);
      if (problem) { status(problem); input.focus(); return false; }
      state.pending = stage(state.pending, row, field, input.value);
    }
    finished = true;
    state.editing = false;
    render();
    moveFocus(dRow, dCol);
    return true;
  };
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") { event.preventDefault(); finish(true, 1, 0); }
    else if (event.key === "Tab") { event.preventDefault(); finish(true, 0, event.shiftKey ? -1 : 1); }
    else if (event.key === "Escape") { event.preventDefault(); finish(false); }
  });
  input.addEventListener("blur", () => finish(true));
}

document.addEventListener("keydown", (event) => {
  if (state.editing || document.querySelector("dialog[open]") || event.target.closest?.("input,select,textarea,button")) return;
  const move = { ArrowDown: [1, 0], ArrowUp: [-1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1] }[event.key];
  if (move) { event.preventDefault(); moveFocus(...move); }
  else if (event.key === "Enter" || event.key === "F2") { event.preventDefault(); startEdit(); }
  else if (event.key === " " && state.rows[state.focus.row]) { event.preventDefault(); toggle(state.rows[state.focus.row].id, event.shiftKey); }
  else if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) { event.preventDefault(); startEdit(event.key); }
});

$("#ed-body").addEventListener("click", (event) => {
  const box = event.target.closest("[data-select]");
  if (box) { toggle(box.dataset.select, event.shiftKey, box.checked); return; }
  const cell = event.target.closest("td[data-field]");
  if (!cell || state.editing) return;
  state.focus = { row: Number(cell.dataset.row), col: Number(cell.dataset.col) };
  moveFocus(0, 0);
});
$("#ed-body").addEventListener("dblclick", (event) => { if (event.target.closest("td[data-field]")) startEdit(); });

function toggle(id, range, on = true) {
  const ids = state.rows.map((row) => row.id);
  if (range && state.anchor && ids.includes(state.anchor)) {
    const [from, to] = [ids.indexOf(state.anchor), ids.indexOf(id)].sort((a, b) => a - b);
    for (const between of ids.slice(from, to + 1)) { if (on) state.selected.add(between); else state.selected.delete(between); }
  } else if (state.selected.has(id)) state.selected.delete(id);
  else state.selected.add(id);
  state.anchor = id;
  render();
}


// ---- dialogs --------------------------------------------------------------------------------------------------
function ask(title, bodyHTML, okLabel, onOk) {
  return new Promise((resolve) => {
    const dialog = $("#ed-dialog");
    $("#ed-dialog-title").textContent = title;
    $("#ed-dialog-body").innerHTML = bodyHTML;
    $("#ed-dialog-error").textContent = "";
    $("#ed-dialog-ok").textContent = okLabel;
    $("#ed-dialog-ok").onclick = async (event) => {
      event.preventDefault();
      const problem = await onOk(new FormData($("#ed-dialog-form")));
      if (problem) { $("#ed-dialog-error").textContent = problem; return; }
      dialog.close("ok");
    };
    dialog.addEventListener("close", () => resolve(dialog.returnValue === "ok"), { once: true });
    dialog.returnValue = "";
    dialog.showModal();
    dialog.querySelector("input,select")?.focus();
  });
}

const selectedRows = () => state.rows.filter((row) => state.selected.has(row.id));
const apply = (changes) => { state.pending = merge(state.pending, changes, state.rows); render(); };

const actions = {
  hardcover: () => hardcoverSyncDialog({ state, api, ask, status, reload: () => loadRows() }),
  collections: () => collectionsDialog({ state, api, ask, status, reload: () => loadRows() }),
  open: () => window.open(`/reader/${encodeURIComponent(state.library)}/${encodeURIComponent([...state.selected][0])}`, "_blank", "noopener"),
  useMatches: () => {
    const changes = Object.fromEntries(selectedRows().map((row) => [row.id, fillFromMatch(row)]).filter(([, fields]) => Object.keys(fields).length));
    if (!Object.keys(changes).length) { toast("None of the selected books has a certain Hardcover match to take. Run Tools > Find Hardcover matches first."); return; }
    apply(changes);
    status(`${Object.keys(changes).length} books ready to review: only empty fields are filled and nothing is replaced. Nothing is saved until you review and save.`);
  },
  useIsbn: () => {
    const picked = selectedRows().filter((row) => row.isbn_found && !isbnOf(row));
    if (!picked.length) { toast("None of the selected books has a found ISBN to use. Run Tools > Find ISBNs first."); return; }
    apply(Object.fromEntries(picked.map((row) => [row.id, { isbn: row.isbn_found }])));
    status(`${picked.length} ISBN${picked.length === 1 ? "" : "s"} ready to review. Nothing is saved until you review and save.`);
  },
  series: () => ask("Set series", '<label>Series for the selected books<input type="text" name="value" autocomplete="off"></label><p>Leave it empty to take the series back to what the files say.</p>', "Set series", (form) => { apply(setAll(selectedRows(), "series", form.get("value") || "")); return ""; }),
  authors: () => ask("Set author", '<label>Author (use " & " between several)<input type="text" name="value" autocomplete="off"></label>', "Set author", (form) => {
    if (!String(form.get("value") || "").trim()) return "Type an author, or use Reset to files to go back to the book's own.";
    apply(setAll(selectedRows(), "authors", form.get("value"))); return "";
  }),
  renumber: () => ask("Renumber in the order shown", '<label>First number<input type="number" name="start" value="1" step="any" min="0"></label><label>Step<input type="number" name="step" value="1" step="any" min="0"></label><p>The selected books get consecutive numbers in the order they are listed (series, then number).</p>', "Renumber", (form) => {
    const start = Number(form.get("start")), step = Number(form.get("step"));
    if (!(start > 0) || !(step > 0)) return "Use a first number and step above zero.";
    apply(renumber(selectedRows(), start, step)); return "";
  }),
  replace: () => ask("Find & replace", '<label>In<select name="field"><option value="title">Title</option><option value="series">Series</option><option value="authors">Author</option><option value="publisher">Publisher</option><option value="tags">Tags</option><option value="description">Description</option></select></label><label>Find<input type="text" name="find" autocomplete="off"></label><label>Replace with (empty removes it)<input type="text" name="replacement" autocomplete="off"></label><label><input type="checkbox" name="prefix"> Only at the start (strips a prefix like “Series - ”)</label><label><input type="checkbox" name="case"> Match case</label>', "Replace", (form) => {
    if (!form.get("find")) return "Type the text to find.";
    const changes = replaceText(selectedRows(), form.get("field"), form.get("find"), form.get("replacement") || "", { prefixOnly: Boolean(form.get("prefix")), matchCase: Boolean(form.get("case")) });
    const blocked = Object.entries(changes).find(([, fields]) => validate("title", fields.title ?? "x"));
    if (blocked) return "That would leave a title empty.";
    if (!Object.keys(changes).length) return "Nothing in the selection contains that text.";
    apply(changes); return "";
  }),
  field: () => ask("Set a field", `<label>Field<select name="field">${COLUMNS.filter((c) => c.field && c.field !== "title" && c.field !== "isbn").map((c) => `<option value="${c.field}">${escapeHTML(c.label)}</option>`).join("")}</select></label><label>New value<input type="text" name="value" autocomplete="off"></label><p>Tags and identifiers take a comma-separated list. A rating is 0 to 5 stars. Leave the value empty to take the field back to what the files say.</p>`, "Set field", (form) => {
    const field = form.get("field"), value = String(form.get("value") || "").trim();
    const problem = validate(field, value);
    if (problem) return problem;
    apply(setAll(selectedRows(), field, value)); return "";
  }),
  match: () => findMatches(),
  reset: () => { for (const id of state.selected) { state.resets.add(id); delete state.pending[id]; } render(); },
  clear: () => { state.selected = new Set(); render(); },
};
$("#ed-selection").addEventListener("click", (event) => { const name = event.target.closest("[data-action]")?.dataset.action; if (name) actions[name](); });

// ---- saving ---------------------------------------------------------------------------------------------------
const label = (field) => ({ title: "Title", authors: "Author", series: "Series", series_index: "Number", all: "Everything" })[field] || field;

async function review() {
  const body = request(state.library, state.pending, { preview: true, reset: [...state.resets] });
  let preview;
  try { preview = await api("/api/metadata/editor/edit", { method: "POST", body }); } catch (error) { status(error.message); return; }
  const rows = preview.changes.map((c) => `<tr><td>${escapeHTML(c.title)}</td><td>${label(c.field)}</td><td><del>${escapeHTML(c.from || "—")}</del></td><td><ins>${escapeHTML(c.to || "from the files")}</ins></td></tr>`).join("");
  const saved = await ask(`Save ${preview.changes.length} ${preview.changes.length === 1 ? "change" : "changes"}?`,
    `<div class="ed-changes"><table><thead><tr><th>Book</th><th>Field</th><th>Was</th><th>Becomes</th></tr></thead><tbody>${rows}</tbody></table></div><p>Mochil keeps these edits. Your book files are not changed, and you can undo right after saving.</p>`, "Save", async () => {
      try {
        const done = await api("/api/metadata/editor/edit", { method: "POST", body: { ...body, preview: false } });
        state.undo = done.undo;
        return "";
      } catch (error) { return error.message; }
    });
  if (!saved) return;
  const count = preview.changes.length;
  state.pending = {}; state.resets = new Set();
  await loadRows();
  toast(`Saved ${count} ${count === 1 ? "change" : "changes"}.`);
}

function toast(text) {
  $("#ed-toast-text").textContent = text;
  $("#ed-toast").hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => { $("#ed-toast").hidden = true; }, 15000);
}
$("#ed-undo").addEventListener("click", async () => {
  try {
    await api("/api/metadata/editor/undo", { method: "POST", body: { token: state.undo } });
    $("#ed-toast").hidden = true;
    // Undoing a move or copy changed files, so both libraries look at their folders again.
    for (const id of state.undoRescan || []) { try { await api("/api/library-scan", { method: "POST", body: { library_id: id } }); } catch (_) {} }
    state.undoRescan = [];
    await loadRows();
    status("Undone.");
  } catch (error) { $("#ed-toast-text").textContent = error.message; }
});
$("#ed-review").addEventListener("click", review);
$("#ed-discard").addEventListener("click", () => { state.pending = {}; state.resets = new Set(); render(); });
window.addEventListener("beforeunload", (event) => { if (pendingCount(state.pending) + state.resets.size) event.preventDefault(); });

// ---- controls -------------------------------------------------------------------------------------------------
$("#ed-filters").addEventListener("click", (event) => {
  const filter = event.target.closest("[data-filter]")?.dataset.filter;
  if (filter) { state.filter = filter; state.rows = []; loadRows(); }
});
let searchTimer;
$("#ed-search").addEventListener("input", (event) => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => { state.q = event.target.value.trim(); state.rows = []; loadRows(); }, 250);
});
// Sorting asks the server, since only part of a large library is loaded. Heading clicks go ascending, descending, then
// back to the default series order.
document.querySelector("thead").addEventListener("click", (event) => {
  const column = event.target.closest("[data-sort]")?.dataset.sort;
  if (!column) return;
  if (state.sort !== column) { state.sort = column; state.dir = "asc"; }
  else if (state.dir === "asc") state.dir = "desc";
  else { state.sort = ""; state.dir = "asc"; }
  state.rows = [];
  state.focus = { row: 0, col: state.focus.col };
  loadRows();
});
$("#ed-manage-collections").addEventListener("click", () => manageCollections({ state, ask, status, reload: () => loadRows() }));
$("#ed-collection").addEventListener("change", (event) => { state.collection = event.target.value; state.rows = []; loadRows(); });
$("#ed-columns-list").addEventListener("change", (event) => {
  const id = event.target.dataset.column;
  if (!id) return;
  const next = event.target.checked ? [...state.columns, id] : state.columns.filter((column) => column !== id);
  if (!visibleColumns(next).some((column) => column.field)) { event.target.checked = true; status("Keep at least one editable column."); return; }
  state.columns = next;
  state.focus = { row: state.focus.row, col: Math.min(state.focus.col, editableColumns().length - 1) };
  remember("mochil-editor-columns", JSON.stringify(next));
  render();
});
mountColumnDrag({
  list: $("#ed-columns-list"),
  ids: () => applyOrder(COLUMNS, state.order).map((column) => column.id),
  onMove: (order, focusId) => {
    state.order = order;
    remember("mochil-editor-column-order", JSON.stringify(order));
    render();
    if (focusId) $(`#ed-columns-list [data-column-row="${focusId}"] input`)?.focus();
  },
});
$("#ed-body").addEventListener("click", (event) => {
  const id = event.target.closest("[data-fetch]")?.dataset.fetch;
  if (!id) return;
  state.selected = new Set([id]);
  render();
  findMatches();
});
document.addEventListener("click", (event) => { if (!event.target.closest("#ed-columns")) $("#ed-columns").open = false; });
$("#ed-library").addEventListener("change", (event) => {
  if (pendingCount(state.pending) + state.resets.size && !confirm("You have unsaved changes in this library. Leave them behind?")) { event.target.value = state.library; return; }
  state.library = event.target.value; state.collection = ""; state.pending = {}; state.resets = new Set(); state.selected = new Set(); state.rows = []; state.filter = "all";
  remember("mochil-editor-library", state.library);
  loadRows();
});
document.querySelectorAll("[data-density]").forEach((button) => button.addEventListener("click", () => {
  state.density = button.dataset.density;
  remember("mochil-editor-density", state.density);
  render();
}));

async function start() {
  try {
    state.libraries = (await api("/api/libraries")).filter((library) => PUBLICATION_KINDS.has(library.kind));
  } catch (error) { status(`Could not load libraries: ${error.message}`); return; }
  if (!state.libraries.length) { status("There are no book libraries to edit."); return; }
  const wanted = new URLSearchParams(location.search).get("library") || recall("mochil-editor-library");
  state.library = state.libraries.some((library) => library.id === wanted) ? wanted : state.libraries[0].id;
  $("#ed-library").innerHTML = state.libraries.map((library) => `<option value="${escapeHTML(library.id)}">${escapeHTML(library.name)}</option>`).join("");
  $("#ed-library").value = state.library;
  await loadRows();
}
const stopButton = $("#ed-stop");
const statusLine = $(".ed-statusline");
const stop = { show: (onClick) => { stopButton.hidden = false; stopButton.onclick = onClick; statusLine.classList.add("busy"); }, hide: () => { stopButton.hidden = true; statusLine.classList.remove("busy"); } };
const toolContext = () => ({ state, api, status, reload: () => loadRows(), stop });
start().then(() => resumeJobs(toolContext()));
keepInView($("#ed-tools"), $("#ed-tools-list"));
keepInView($(".ed-columns:not(.ed-tools)"), $("#ed-columns-list"));
mountTools({ list: $("#ed-tools-list"), details: $("#ed-tools"), context: toolContext, hermesEnabled: moduleEnabled("hermes") });


// ---- find matches ---------------------------------------------------------------------------------------------
// Looks the one selected book up on Hardcover. Nothing is applied for the person: they pick the candidate and tick the
// fields to take, which then wait as ordinary pending edits to review and save (or discard).
async function findMatches() {
  const [row] = selectedRows();
  if (!row) return;
  status(`Looking up “${row.title}”…`);
  let reply;
  try {
    reply = await api(`/api/metadata/editor/match?library=${encodeURIComponent(state.library)}&book=${encodeURIComponent(row.id)}`);
  } catch (error) { status(error.message); return; }
  status("");
  if (!reply.candidates.length) { await ask("No matches", `<p>Hardcover has nothing for “${escapeHTML(row.title)}”. Try correcting the title first.</p>`, "OK", () => ""); return; }
  const list = reply.candidates.map((c, i) => `<label class="ed-candidate"><input type="radio" name="candidate" value="${i}" ${i === 0 ? "checked" : ""}><span><strong>${escapeHTML(c.title)}</strong><br>${escapeHTML((c.authors || []).join(", "))}${c.year ? ` · ${escapeHTML(String(c.year))}` : ""}</span><em>${scoreText(c.score)}</em></label>`).join("");
  const unsure = reply.candidates.length === 1 ? "Only one match, and not a certain one. Check it is this book before taking anything." : "Several are close. Check which one is yours before taking anything.";
  const hint = reply.clear ? "" : `<p class="ed-error">${unsure}</p>`;
  const picked = await ask(`Matches for “${row.title}”`, `${hint}<div class="ed-candidates">${list}</div><p class="ed-hint">Next you choose which fields to take.</p>`, "Choose fields", (form) => (form.get("candidate") === null ? "Pick a match." : ""));
  if (!picked) return;
  const candidate = reply.candidates[Number($("#ed-dialog-form").elements.candidate?.value ?? 0)] || reply.candidates[0];
  const fields = matchFields(row, candidate);
  if (!fields.length) { toast("That match has nothing new for this book."); return; }
  const rows = fields.map((f) => `<label class="ed-field-pick"><input type="checkbox" name="take" value="${f.field}" ${f.field === "identifiers" ? "checked" : ""}><span><strong>${escapeHTML(f.label)}</strong><br><s>${escapeHTML(String(f.from).slice(0, 120)) || "empty"}</s> → ${escapeHTML(String(f.to).slice(0, 160))}</span></label>`).join("");
  await ask("Take which fields?", `<div class="ed-candidates">${rows}</div>`, "Add to my changes", (form) => {
    const chosen = form.getAll("take");
    if (!chosen.length) return "Tick at least one field.";
    apply({ [row.id]: matchChanges(fields, chosen) });
    return "";
  });
}


// ---- the person's own columns ---------------------------------------------------------------------------------
$("#ed-columns-list").addEventListener("click", async (event) => {
  if (event.target.closest("[data-add-column]")) {
    $("#ed-columns").open = false;
    let created = null;
    await ask("Add a column", '<label>Name<input type="text" name="label" maxlength="40" autocomplete="off"></label><label>What it holds<select name="type"><option value="text">Text</option><option value="number">Number</option><option value="yesno">Yes / no</option><option value="tags">A list (comma-separated)</option></select></label><p>It is kept by Mochil, per book, and never written into your files. Edit it like any other column.</p>', "Add column", async (form) => {
      try { created = await api("/api/metadata/editor/columns", { method: "POST", body: { label: form.get("label"), type: form.get("type") } }); return ""; } catch (error) { return error.message; }
    });
    if (created) {
      state.columns = [...state.columns, created.id];
      remember("mochil-editor-columns", JSON.stringify(state.columns));
      await loadRows();
    }
    return;
  }
  const remove = event.target.closest("[data-remove-column]")?.dataset.removeColumn;
  if (remove) {
    event.preventDefault();
    const column = COLUMNS.find((c) => c.id === remove);
    if (!confirm(`Remove the “${column?.label}” column and what is in it for every book?`)) return;
    try { await api(`/api/metadata/editor/columns?id=${encodeURIComponent(remove)}`, { method: "DELETE" }); } catch (error) { status(error.message); return; }
    state.columns = state.columns.filter((id) => id !== remove);
    for (const fields of Object.values(state.pending)) delete fields[remove];
    state.pending = Object.fromEntries(Object.entries(state.pending).filter(([, fields]) => Object.keys(fields).length));
    remember("mochil-editor-columns", JSON.stringify(state.columns));
    if (state.sort === remove) state.sort = "";
    await loadRows();
  }
});


// ---- move or copy to another library ----------------------------------------------------------------------------
// Two steps: choose where (the server checks every book and says what would stop it), then confirm. A move or copy
// that goes through can be undone from the toast.
async function transfer(mode) {
  if (pendingCount(state.pending) + state.resets.size) { status("Save or discard your changes first: a book that moves takes its saved changes with it."); return; }
  const books = selectedRows();
  if (!books.length) return;
  if (!state.transferTo.length) { status("There is no other ordinary folder library to send books to."); return; }
  const ids = books.map((book) => book.id);
  const options = state.transferTo.map((library) => `<option value="${escapeHTML(library.id)}">${escapeHTML(library.name)}</option>`).join("");
  let chosen = null;
  const picked = await ask(`${mode === "move" ? "Move" : "Copy"} ${books.length} ${books.length === 1 ? "book" : "books"}`, `<label>To library<select name="to">${options}</select></label><p class="ed-hint">Each book keeps its folder and file name in the other library.</p>`, "Check", async (form) => {
    const to = String(form.get("to") || "");
    try {
      const preview = await api("/api/metadata/editor/transfer", { method: "POST", body: transferRequest(state.library, to, ids, mode, true) });
      if (preview.blocked) return blockedText(preview);
      chosen = { to, name: preview.to };
      return "";
    } catch (error) { return error.message; }
  });
  if (!picked || !chosen) return;
  let result = null;
  const confirmed = await ask("Are you sure?", `<p>${escapeHTML(confirmText(mode, books.length, chosen.name))}</p>`, mode === "move" ? "Move" : "Copy", async () => {
    try { result = await api("/api/metadata/editor/transfer", { method: "POST", body: transferRequest(state.library, chosen.to, ids, mode, false) }); return ""; } catch (error) { return error.message; }
  });
  if (!confirmed || !result) return;
  state.undo = result.token;
  state.undoRescan = result.rescan || [];
  state.selected = new Set();
  status("Updating the libraries…");
  for (const id of result.rescan || []) { try { await api("/api/library-scan", { method: "POST", body: { library_id: id } }); } catch (_) {} }
  await loadRows();
  status("");
  toast(doneText(mode, result.done, result.to));
}
Object.assign(actions, { move: () => transfer("move"), copy: () => transfer("copy") });
