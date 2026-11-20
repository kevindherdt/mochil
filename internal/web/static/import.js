import { destinationOf, groupByDestination, tally, readyRows, STATUS_FILTERS, STATUS_TEXT, chunk, entriesFor, fieldText, parseEdit, sidecarFor, splitPreview, statusesFor, nextSort, rangeBetween, sortRows, targetOf, validateEdit, view, withoutRepeatedTargets, shouldDeferRender } from "./import-model.js?v=7";
import { createFolderPicker, attachFolderBrowse } from "./folder-picker.js?v=4";
import { requestJSON } from "./http-json.js?v=2";

const $ = (selector) => document.querySelector(selector);
const PUBLICATION_KINDS = new Set(["books", "comics", "manga", "mixed"]);
const FIELDS = ["title", "authors", "series", "series_index"];
const escapeHTML = (value) => String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const remember = (key, value) => { try { localStorage.setItem(key, value); } catch (_) {} };
const recall = (key) => { try { return localStorage.getItem(key); } catch (_) { return null; } };

const root = document.documentElement;
const mode = recall("mochil-theme-mode") || recall("mochil-theme") || "system";
root.dataset.palette = recall("mochil-palette") || "mochil";
root.dataset.theme = mode === "system" ? (matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark") : mode;


const state = {
  libraries: [], library: "", naming: recall("mochil-import-naming") || "default", into: recall("mochil-import-into") || "", destinations: [], dest: {}, hideCopied: recall("mochil-import-hide-copied") === "1", filter: "fresh",
  rows: [], cursor: "", scanned: 0, enabled: true, loading: false, incoming: "", setupOpen: false,
  sort: { key: "", dir: "asc" }, managed: { directory: "", library: null }, selected: new Set(), anchor: "", edits: {}, targets: {}, copied: new Set(), focus: { row: 0, col: 0 }, editing: false,
};

// Where a book goes: its own choice, otherwise the page's default. The saved name is the layout's name inside that
// library's folder.
const destinationId = (row) => destinationOf(row, state.dest, state.into, state.destinations);
const destinationName = (id) => (id ? (state.destinations.find((d) => d.id === id) || {}).name || id : state.managed.library ? state.managed.library.name : "Not sorted yet");
const fullTarget = (row) => targetOf(row, state.targets);
const destinationQuery = (id) => (id ? `?destination=${encodeURIComponent(id)}` : "");

async function api(path, options) {
  const response = await fetch(path, options && { ...options, headers: { "Content-Type": "application/json" }, body: JSON.stringify(options.body) });
  if (!response.ok) throw new Error((await response.text()).trim() || `HTTP ${response.status}`);
  return response.json();
}
const status = (text) => { $("#im-status").textContent = text; };
const rowAt = (index) => state.rows[index];

async function load(append = false) {
  if (state.loading) return;
  state.loading = true;
  status("Reading the drop folder…");
  try {
    const query = new URLSearchParams({ library: state.library, naming: state.naming, status: statusesFor(state.filter), limit: "100" });
    if (append && state.cursor) query.set("cursor", state.cursor);
    const reply = await api(`/api/metadata/editor/import?${query}`);
    state.enabled = reply.enabled !== false;
    state.incoming = reply.incoming_directory || "";
    const known = new Set(state.rows.map((row) => row.source_path));
    const start = append ? state.rows.length : 0;
    const fresh = (append ? reply.items.filter((row) => !known.has(row.source_path)) : reply.items).map((row, i) => ({ ...row, order: start + i }));
    // The server remembers what was copied, so every device shows the same marks.
    for (const row of fresh) if (row.copied) state.copied.add(row.source_path);
    state.rows = append ? [...state.rows, ...fresh] : fresh;
    state.destinations = reply.destinations || [];
    if (state.into && !state.destinations.some((d) => d.id === state.into)) state.into = "";
    state.managed = { directory: reply.managed_directory || "", library: reply.managed_library || null };
    applySort();
    state.cursor = reply.next_cursor || "";
    state.scanned = (append ? state.scanned : 0) + (reply.scanned || 0);
  } catch (error) {
    status(`Could not read the drop folder: ${error.message}`);
  } finally {
    state.loading = false;
  }
  render();
}

const applySort = () => { state.rows = sortRows(state.rows, state.sort.key, state.sort.dir, state.edits, state.targets); state.focus = { row: 0, col: 0 }; };

function render() {
  // An edit in progress owns the table: rendering under it would remove the input the person is typing
  // into, and the editor would never finish. See shouldDeferRender.
  if (shouldDeferRender(state)) return;
  $("#im-filters").innerHTML = STATUS_FILTERS.map(([id, label]) => `<button type="button" data-filter="${id}" aria-pressed="${state.filter === id}">${label}</button>`).join("");
  const heading = (key, label, extra = "") => `<th ${extra} data-col-id="${key}" aria-sort="${state.sort.key === key ? (state.sort.dir === "desc" ? "descending" : "ascending") : "none"}"><button type="button" class="ed-sort" data-sort="${key}">${label}<span class="ed-arrow" aria-hidden="true"></span></button></th>`;
  $("#im-head").innerHTML = '<th class="ed-check"><input type="checkbox" id="im-all" aria-label="Select all shown"></th>' + heading("status", "Status") + heading("title", "Title") + heading("authors", "Author") + heading("series", "Series") + heading("series_index", "#", 'class="ed-num"') + '<th data-col-id="library">Library</th>' + heading("target", "Will be saved as") + heading("source", "From the drop folder");
  const where = state.managed.directory;
  const chosen = state.destinations.find((d) => d.id === state.into);
  $("#im-into").innerHTML = `<option value="">${state.managed.library ? escapeHTML(state.managed.library.name) : "Don't sort yet — just copy them"}</option>` + state.destinations.map((d) => `<option value="${escapeHTML(d.id)}">${escapeHTML(d.name)}</option>`).join("") + '<option value="__new">New library…</option>';
  $("#im-into").value = state.into;
  $("#im-destination").textContent = !where ? "" : `By default, copied books go to ${chosen ? `“${chosen.name}”` : state.managed.library ? `“${state.managed.library.name}”` : "nowhere in particular — just the managed folder, so they won't show up in any library until you pick one"}. Pick a different library for some books with the Library column or Send to library…. Nothing is copied until you review and confirm.`;
  const selectable = state.rows.filter((row) => !state.copied.has(row.source_path));
  $("#im-all").checked = selectable.length > 0 && selectable.every((row) => state.selected.has(row.source_path));
  $("#im-all").onchange = (event) => { state.selected = event.target.checked ? new Set(selectable.map((row) => row.source_path)) : new Set(); render(); };
  $("#im-hide-copied").checked = state.hideCopied;
  $("#im-body").innerHTML = state.rows.map((row, index) => {
    const book = view(row, state.edits);
    const done = state.copied.has(row.source_path);
    if (done && state.hideCopied) return "";
    const target = fullTarget(row);
    const conflicts = (row.proposal.conflicts || []).join("\n");
    const lowAuthor = (row.proposal.evidence || []).some((e) => e.field === "authors" && e.confidence === "low") && !("authors" in (state.edits[row.source_path] || {}));
    const cells = FIELDS.map((field, col) => {
      const text = fieldText(book, field);
      const changed = field in (state.edits[row.source_path] || {});
      const focus = state.focus.row === index && state.focus.col === col;
      return `<td data-field="${field}" data-row="${index}" data-col="${col}" data-changed="${changed}" data-focus="${focus}" class="${field === "series_index" ? "ed-num" : ""}" title="${escapeHTML(text)}">${escapeHTML(text)}${field === "authors" && lowAuthor && text ? '<span class="im-warn" title="Guessed from a folder name">guess</span>' : ""}${field === "title" && conflicts ? `<span class="im-warn" title="${escapeHTML(conflicts)}">check</span>` : ""}</td>`;
    }).join("");
    const label = done ? "Copied" : STATUS_TEXT[row.status] || row.status;
    const existing = row.existing && !done ? `<span class="im-note" title="${escapeHTML(row.existing.title)}">${escapeHTML(row.existing.title)}</span>` : "";
    return `<tr data-source="${escapeHTML(row.source_path)}" data-copied="${done}" data-selected="${state.selected.has(row.source_path)}"><td><input type="checkbox" data-select="${escapeHTML(row.source_path)}" ${done ? "disabled" : ""} ${state.selected.has(row.source_path) ? "checked" : ""} aria-label="Select ${escapeHTML(book.title)}"></td>` +
      `<td><span class="im-badge" data-status="${done ? "in_library" : row.status}">${label}</span>${existing}</td>${cells}<td data-library="true">${escapeHTML(destinationName(destinationId(row)))}</td><td title="${escapeHTML(target)}">${target ? escapeHTML(target) : '<span class="im-warn">needs a title</span>'}</td><td title="${escapeHTML(row.source_path)}">${escapeHTML(row.source_path)}</td></tr>`;
  }).join("");
  $("#im-more").hidden = !state.cursor;
  $("#im-everything").hidden = !state.cursor;
  $("#im-more").textContent = "Load more";
  if (!state.loading) status(state.rows.length ? `${state.rows.length} books shown${state.cursor ? " so far. Sorting only covers these; use Load everything to include the rest." : ""}` : "Nothing here.");
  const count = state.selected.size;
  $("#im-bar").hidden = count === 0;
  $("#im-bar-text").textContent = `${count} selected to copy`;
  renderSetup();
}

// Picking a drop folder is the whole setup step: no incoming/managed_ingest
// config to hand-edit, just a folder a reader already has mounted. Shown
// whenever Import has not been pointed at one yet, or the reader asks to
// change it, and otherwise collapsed to a one-line "Importing from …".
function renderSetup() {
  const needsSetup = !state.enabled || state.setupOpen;
  $("#im-setup").hidden = !needsSetup;
  $("#im-source").hidden = needsSetup || !state.incoming;
  $("#im-content").hidden = needsSetup;
  $("#im-filters").hidden = needsSetup;
  $("#im-destination").hidden = needsSetup;
  $("#im-bar").hidden = needsSetup || state.selected.size === 0;
  if (needsSetup) $("#im-setup-incoming").value = $("#im-setup-incoming").value || state.incoming;
  else $("#im-source-path").textContent = state.incoming;
}

// ---- editing a cell ------------------------------------------------------------------------------------------
const cellAt = (row, col) => document.querySelector(`td[data-row="${row}"][data-col="${col}"]`);
function moveFocus(dRow, dCol) {
  const row = Math.min(state.rows.length - 1, Math.max(0, state.focus.row + dRow));
  const col = Math.min(FIELDS.length - 1, Math.max(0, state.focus.col + dCol));
  state.focus = { row, col };
  document.querySelectorAll('td[data-focus="true"]').forEach((cell) => { cell.dataset.focus = "false"; });
  const cell = cellAt(row, col);
  if (cell) { cell.dataset.focus = "true"; cell.scrollIntoView({ block: "nearest", inline: "nearest" }); }
}

async function refreshTarget(row) {
  const book = view(row, state.edits);
  try {
    const reply = await api("/api/metadata/editor/import-names", { method: "POST", body: { naming: state.naming, items: [{ source_path: row.source_path, title: book.title, authors: book.authors, series: book.series, series_index: book.series_index }] } });
    state.targets[row.source_path] = reply.items[0].target;
  } catch (error) { status(`Could not work out the new name: ${error.message}`); }
  render();
}

function startEdit(initial) {
  const row = rowAt(state.focus.row), cell = cellAt(state.focus.row, state.focus.col);
  if (!row || !cell || state.editing) return;
  const field = FIELDS[state.focus.col];
  const input = document.createElement("input");
  input.className = "ed-cell";
  input.type = "text";
  input.value = initial ?? fieldText(view(row, state.edits), field);
  cell.textContent = "";
  cell.append(input);
  state.editing = true;
  input.focus();
  if (initial === undefined) input.select();
  let finished = false;
  const finish = (commit, dRow = 0, dCol = 0) => {
    if (finished) return true;
    if (commit) {
      const problem = validateEdit(field, input.value);
      if (problem) { status(problem); input.focus(); return false; }
      const value = parseEdit(field, input.value);
      if (JSON.stringify(value) !== JSON.stringify(view(row)[field])) {
        state.edits = { ...state.edits, [row.source_path]: { ...(state.edits[row.source_path] || {}), [field]: value } };
        void refreshTarget(row);
      } else if (state.edits[row.source_path]) {
        const rest = { ...state.edits[row.source_path] }; delete rest[field];
        state.edits = { ...state.edits }; if (Object.keys(rest).length) state.edits[row.source_path] = rest; else delete state.edits[row.source_path];
        void refreshTarget(row);
      }
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
  else if (event.key === " " && rowAt(state.focus.row) && !state.copied.has(rowAt(state.focus.row).source_path)) {
    event.preventDefault();
    const path = rowAt(state.focus.row).source_path;
    if (event.shiftKey && state.anchor) {
      const ids = state.rows.filter((row) => !state.copied.has(row.source_path)).map((row) => row.source_path);
      for (const id of rangeBetween(ids, state.anchor, path)) state.selected.add(id);
    } else if (state.selected.has(path)) state.selected.delete(path);
    else state.selected.add(path);
    state.anchor = path;
    render();
  }
  else if (event.key === "Enter" || event.key === "F2") { event.preventDefault(); startEdit(); }
  else if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey && event.key !== " ") { event.preventDefault(); startEdit(event.key); }
});
$("#im-body").addEventListener("click", (event) => {
  const box = event.target.closest("[data-select]");
  if (box) {
    const path = box.dataset.select;
    if (event.shiftKey && state.anchor) {
      // shift-click takes everything from the last click to this one, as it was left: ticked, or unticked
      const ids = state.rows.filter((row) => !state.copied.has(row.source_path)).map((row) => row.source_path);
      for (const id of rangeBetween(ids, state.anchor, path)) { if (box.checked) state.selected.add(id); else state.selected.delete(id); }
    } else if (state.selected.has(path)) state.selected.delete(path);
    else state.selected.add(path);
    state.anchor = path;
    render();
    return;
  }
  const cell = event.target.closest("td[data-field]");
  if (!cell || state.editing) return;
  state.focus = { row: Number(cell.dataset.row), col: Number(cell.dataset.col) };
  moveFocus(0, 0);
});
$("#im-body").addEventListener("dblclick", (event) => { if (event.target.closest("td[data-field]")) startEdit(); });

// ---- copying -------------------------------------------------------------------------------------------------
const dialog = $("#ed-dialog");
// show opens the dialog; onOk may keep it open (returning "keep") while it reports progress, then closes it on Close.
function show(title, bodyHTML, okLabel, onOk) {
  return new Promise((resolve) => {
    $("#ed-dialog-title").textContent = title;
    $("#ed-dialog-body").innerHTML = bodyHTML;
    $("#ed-dialog-error").textContent = "";
    $("#ed-dialog-ok").textContent = okLabel;
    $("#ed-dialog-ok").hidden = !onOk;
    $("#ed-dialog-cancel").textContent = onOk ? "Cancel" : "Close";
    $("#ed-dialog-cancel").disabled = false;
    $("#ed-dialog-ok").onclick = async (event) => {
      event.preventDefault();
      $("#ed-dialog-ok").disabled = true; $("#ed-dialog-cancel").disabled = true;
      const result = await onOk();
      $("#ed-dialog-ok").disabled = false; $("#ed-dialog-cancel").disabled = false;
      if (result === "keep") { $("#ed-dialog-ok").hidden = true; $("#ed-dialog-cancel").textContent = "Close"; return; }
      if (result) { $("#ed-dialog-error").textContent = result; return; }
      dialog.close("ok");
    };
    dialog.addEventListener("close", () => resolve(dialog.returnValue === "ok"), { once: true });
    dialog.returnValue = "";
    dialog.showModal();
  });
}
const list = (items, render) => `<ul class="im-list">${items.slice(0, 40).map((item) => `<li>${render(item)}</li>`).join("")}${items.length > 40 ? `<li>…and ${items.length - 40} more</li>` : ""}</ul>`;

async function review() {
  const chosen = state.rows.filter((row) => state.selected.has(row.source_path) && !state.copied.has(row.source_path));
  const { entries, unnamed } = entriesFor(chosen, fullTarget);
  let ready = [], blocked = [];
  try {
    const rowsByPath = new Map(chosen.map((row) => [row.source_path, row]));
    const unique = withoutRepeatedTargets(entries.map((entry) => ({ ...entry, destination: destinationId(rowsByPath.get(entry.source_path)) })));
    blocked = [...unique.blocked];
    for (const [destination, group] of groupByDestination(unique.ready)) {
      for (const part of chunk(group, 100)) {
        const split = splitPreview(await api(`/api/metadata/managed-ingest/preview${destinationQuery(destination)}`, { method: "POST", body: { entries: part } }));
        ready.push(...split.ready.map((entry) => ({ ...entry, destination }))); blocked.push(...split.blocked);
      }
    }
  } catch (error) { status(`Could not check the books: ${error.message}`); return; }
  const readyChosen = readyRows(chosen, ready);
  const readyPaths = new Set(readyChosen.map((row) => row.source_path));
  const inLibrary = chosen.filter((row) => row.status === "in_library" && readyPaths.has(row.source_path)).length;
  const sidecar = $("#im-sidecar").checked;
  const body = `<p><strong>${ready.length}</strong> ${ready.length === 1 ? "book is" : "books are"} ready to copy${sidecar ? ", each with a metadata.opf beside it" : ""}${$("#im-embed").checked ? ", and the details written into each EPUB" : ""}.</p>` +
    `<ul>${tally(readyChosen, destinationId, destinationName).map((t) => `<li>${t.count} to <strong>${escapeHTML(t.name)}</strong></li>`).join("")}</ul>` +
    (inLibrary ? `<p class="ed-error">${inLibrary} of them look like books you already have. They will be copied again.</p>` : "") +
    (blocked.length ? `<p>${blocked.length} will be skipped:</p>${list(blocked, (b) => `${escapeHTML(b.source_path)}<span class="im-note">${escapeHTML(b.reasons.join("; "))}</span>`)}` : "") +
    (unnamed.length ? `<p>${unnamed.length} have no title yet, so they cannot be named. Give them one and try again:</p>${list(unnamed, (p) => escapeHTML(p))}` : "") +
    '<p>Your originals in the drop folder are not changed or removed.</p>';
  if (!ready.length) { await show("Nothing to copy", body, "", null); return; }
  await show(`Copy ${ready.length} ${ready.length === 1 ? "book" : "books"}?`, body, `Copy ${ready.length}`, () => copyAll(ready, blocked));
}

async function waitFor(id) {
  for (let i = 0; i < 600; i++) {
    const transaction = await api(`/api/metadata/managed-ingest/${encodeURIComponent(id)}`);
    if (["committed", "failed", "rolled_back"].includes(transaction.state)) return transaction;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error("The copy is taking too long. Check the import journal before trying again.");
}

async function copyAll(ready, blocked) {
  const total = ready.length;
  const rowsByPath = new Map(state.rows.map((row) => [row.source_path, row]));
  const body = $("#ed-dialog-body");
  const progress = (done, note) => { body.innerHTML = `<p>${escapeHTML(note)}</p><div class="im-progress"><span style="width:${Math.round((done / total) * 100)}%"></span></div><p>${done} of ${total}</p>`; };
  let copied = 0, written = 0, kept = 0, failedSidecars = [], embedded = 0, failedEmbeds = [];
  const failures = [];
  progress(0, "Copying…");
  const batches = groupByDestination(ready).flatMap(([destination, group]) => chunk(group, 100).map((part) => ({ destination, part })));
  for (const { destination, part } of batches) {
    const query = destinationQuery(destination);
    try {
      const planned = await api(`/api/metadata/managed-ingest/plan${query}`, { method: "POST", body: { entries: part } });
      await api(`/api/metadata/managed-ingest/${encodeURIComponent(planned.id)}/execute`, { method: "POST", body: { confirmation: planned.id } });
      const done = await waitFor(planned.id);
      if (done.state !== "committed") { failures.push(`${part.length} books were not copied (${done.error || done.state}). Nothing was left half-copied.`); continue; }
      copied += part.length;
      for (const entry of part) state.copied.add(entry.source_path);
      state.selected = new Set([...state.selected].filter((path) => !state.copied.has(path)));
      if ($("#im-sidecar").checked) {
        const sidecars = await api(`/api/metadata/managed-ingest/sidecars${query}`, { method: "POST", body: { overwrite: false, entries: part.map((entry) => ({ target_path: entry.target_path, metadata: sidecarFor(rowsByPath.get(entry.source_path), state.edits) })) } });
        for (const result of sidecars.results) {
          if (result.status === "written") written++;
          else if (result.status === "exists") kept++;
          else failedSidecars.push(`${result.target_path}: ${result.message}`);
        }
      }
      if ($("#im-embed").checked) {
        const epubs = part.filter((entry) => /\.epub$/i.test(entry.target_path));
        if (epubs.length) {
          const reply = await api(`/api/metadata/managed-ingest/embed${query}`, { method: "POST", body: { entries: epubs.map((entry) => ({ target_path: entry.target_path, metadata: sidecarFor(rowsByPath.get(entry.source_path), state.edits) })) } });
          for (const result of reply.results) {
            if (result.status === "embedded") embedded++;
            else failedEmbeds.push(`${result.target_path}: ${result.message}`);
          }
        }
      }
    } catch (error) { failures.push(error.message); }
    progress(copied, "Copying…");
  }
  const copiedRows = ready.map((entry) => rowsByPath.get(entry.source_path)).filter((row) => row && state.copied.has(row.source_path));
  const touched = [...new Set(copiedRows.map(destinationId).filter(Boolean))];
  for (const id of touched) { try { await api("/api/library-scan", { method: "POST", body: { library_id: id } }); } catch (_) {} }
  const where = tally(copiedRows, destinationId, destinationName);
  $("#ed-dialog-title").textContent = copied === total ? "Copied" : copied ? "Partly copied" : "Nothing was copied";
  body.innerHTML = `<p><strong>${copied}</strong> of ${total} ${total === 1 ? "book" : "books"} copied.</p>${where.length ? `<ul>${where.map((t) => `<li>${t.count} into <strong>${escapeHTML(t.name)}</strong></li>`).join("")}</ul>` : ""}` +
    ($("#im-sidecar").checked ? `<p>${written} metadata.opf ${written === 1 ? "file" : "files"} written${kept ? `, ${kept} already there and left alone` : ""}${failedSidecars.length ? `, ${failedSidecars.length} failed` : ""}.</p>` : "") +
    ($("#im-embed").checked ? `<p>${embedded} ${embedded === 1 ? "EPUB" : "EPUBs"} rewritten with the metadata${failedEmbeds.length ? `, ${failedEmbeds.length} left as copied` : ""}.</p>${failedEmbeds.length ? list(failedEmbeds, escapeHTML) : ""}` : "") +
    (failures.length ? `<p class="ed-error">${failures.map(escapeHTML).join("<br>")}</p>` : "") +
    (failedSidecars.length ? list(failedSidecars, escapeHTML) : "") +
    (blocked.length ? `<p>${blocked.length} were skipped.</p>` : "") +
    (touched.length ? `<p>${touched.length === 1 ? "That library is" : "Those libraries are"} rescanning, so the books appear there in a moment.</p>` : state.managed.library ? "" : "<p>These books were copied but not sorted into any library yet, so they won't show up anywhere until you choose one. Use Send to library… above, or add the managed folder itself as a library in Settings → Libraries.</p>");
  return "keep";
}

// ---- controls -------------------------------------------------------------------------------------------------
$("#im-filters").addEventListener("click", (event) => {
  const filter = event.target.closest("[data-filter]")?.dataset.filter;
  if (filter && filter !== state.filter) {
    // The rows are replaced, so a selection made under the old filter can no longer
    // be seen or copied; keeping it made the bar promise books review() could not find.
    state.filter = filter; state.rows = []; state.cursor = ""; state.selected = new Set(); state.anchor = ""; load();
  }
});
$("#im-more").addEventListener("click", () => load(true));
$("#im-everything").addEventListener("click", async () => {
  for (let guard = 0; state.cursor && guard < 80; guard++) await load(true);
});
$("#im-head").addEventListener("click", (event) => {
  const key = event.target.closest("[data-sort]")?.dataset.sort;
  if (!key) return;
  state.sort = nextSort(state.sort, key);
  applySort();
  render();
});
$("#im-copy").addEventListener("click", review);
$("#im-hide-copied").addEventListener("change", (event) => { state.hideCopied = event.target.checked; remember("mochil-import-hide-copied", state.hideCopied ? "1" : "0"); render(); });
$("#im-clear").addEventListener("click", () => { state.selected = new Set(); render(); });
$("#im-library").addEventListener("change", (event) => { state.library = event.target.value; remember("mochil-import-library", state.library); state.rows = []; state.cursor = ""; state.selected = new Set(); load(); });
$("#im-into").addEventListener("change", async (event) => {
  if (event.target.value === "__new") { event.target.value = state.into; await newLibrary(); return; }
  state.into = event.target.value; remember("mochil-import-into", state.into);
  render();
});
$("#im-send").addEventListener("click", async () => {
  const chosen = state.rows.filter((row) => state.selected.has(row.source_path) && !state.copied.has(row.source_path));
  if (!chosen.length) return;
  const options = [["", state.managed.library ? state.managed.library.name : "Don't sort yet — just copy them"], ...state.destinations.map((d) => [d.id, d.name])];
  const body = `<p>Send ${chosen.length} ${chosen.length === 1 ? "book" : "books"} to:</p><div class="im-choices">${options.map(([id, name], i) => `<label><input type="radio" name="im-destination" value="${escapeHTML(id)}" ${i === 0 ? "checked" : ""}> ${escapeHTML(name)}</label>`).join("")}</div>`;
  const picked = await show("Send to library", body, "Send", async () => (document.querySelector('input[name="im-destination"]:checked') ? "" : "Pick a library."));
  if (!picked) return;
  const id = document.querySelector('input[name="im-destination"]:checked').value;
  state.dest = { ...state.dest, ...Object.fromEntries(chosen.map((row) => [row.source_path, id])) };
  render();
});
async function newLibrary() {
  let created = null;
  await show("New library", '<label>Name<input type="text" id="im-new-name" maxlength="60" autocomplete="off" placeholder="Books 2"></label><p>An empty library is made inside the managed folder and added to your settings. It is ready straight away.</p>', "Create", async () => {
    try { created = await api("/api/metadata/editor/import-library", { method: "POST", body: { name: $("#im-new-name").value } }); return ""; }
    catch (error) {
      // The server reports this as a 409 when the drop folder isn't set up (or was
      // unset between visits); say what to actually do, not just that it failed.
      return /managed ingest|config file is not available/i.test(error.message)
        ? "Set a drop folder first — use Change… next to “Importing from” above, then try New library again."
        : error.message;
    }
  });
  if (!created) return;
  remember("mochil-import-into", created.id);
  if (!created.restart_required) {
    // the server applied it at once: use the new library as the default and go on
    state.into = created.id; state.rows = []; state.cursor = ""; state.selected = new Set();
    await load();
    status(`“${created.name}” is ready and is now the default library.`);
    return;
  }
  await show(`“${created.name}” is ready`, "<p>The library needs a restart before it appears. After that, choose it as the default library and copy your books.</p>", "Restart now", async () => {
    try { await api("/api/restart", { method: "POST", body: {} }); } catch (error) { return error.message; }
    $("#ed-dialog-body").innerHTML = "<p>Restarting…</p>";
    for (let i = 0; i < 60; i++) {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      try { if ((await fetch("/api/libraries")).ok) { location.reload(); return "keep"; } } catch (_) {}
    }
    return "It is taking long. Reload the page in a moment.";
  });
}
$("#im-naming").addEventListener("change", (event) => {
  if (Object.keys(state.edits).length && !confirm("Changing the layout recalculates every name. Your edits to titles and authors stay.")) { event.target.value = state.naming; return; }
  state.naming = event.target.value; remember("mochil-import-naming", state.naming); state.targets = {}; state.rows = []; state.cursor = ""; load();
});
window.addEventListener("beforeunload", (event) => { if (Object.keys(state.edits).length) event.preventDefault(); });

const folderPicker = createFolderPicker({ document, requestJSON });
attachFolderBrowse($("#im-setup-incoming"), folderPicker);

$("#im-source-change").addEventListener("click", () => {
  state.setupOpen = true;
  $("#im-setup-incoming").value = state.incoming;
  $("#im-setup-status").textContent = "";
  renderSetup();
  $("#im-setup-incoming").focus();
});

$("#im-setup-save").addEventListener("click", async () => {
  const path = $("#im-setup-incoming").value.trim();
  const button = $("#im-setup-save");
  const setupStatus = $("#im-setup-status");
  if (!path) { setupStatus.textContent = "Choose a folder first."; return; }
  button.disabled = true;
  setupStatus.textContent = "Saving…";
  try {
    // The managed folder is never asked for: it is derived under the server's
    // own data directory the moment a drop folder is saved without one.
    const { data: settings } = await requestJSON("/api/settings");
    settings.managed_ingest = { ...(settings.managed_ingest || {}), incoming_directory: path };
    await requestJSON("/api/settings", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(settings) });
    state.setupOpen = false;
    setupStatus.textContent = "";
    state.rows = []; state.cursor = "";
    await load();
  } catch (error) {
    setupStatus.textContent = error.message;
  } finally {
    button.disabled = false;
  }
});

async function start() {
  $("#im-naming").value = state.naming;
  try {
    state.libraries = (await api("/api/libraries")).filter((library) => PUBLICATION_KINDS.has(library.kind));
  } catch (error) { status(`Could not load libraries: ${error.message}`); return; }
  const wanted = recall("mochil-import-library");
  state.library = state.libraries.some((library) => library.id === wanted) ? wanted : (state.libraries.find((library) => library.id === "light-novels") || state.libraries[0] || {}).id || "";
  $("#im-library").innerHTML = '<option value="">(nothing)</option>' + state.libraries.map((library) => `<option value="${escapeHTML(library.id)}">${escapeHTML(library.name)}</option>`).join("");
  $("#im-library").value = state.library;
  await load();
}
$("#im-options").open = matchMedia("(min-width:900px)").matches;
start();
