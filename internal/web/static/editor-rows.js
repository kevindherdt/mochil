// What a row of the editor's table looks like, as markup. Only the rows in view are drawn (see virtual-rows.js).
import { badgeHTML, okBadgeHTML } from "./editor-badges.js?v=2";
import { countText, fieldValue, folderOf, issueText, progressText, scoreText, shown as cellText } from "./editor-model.js?v=9";

export const escapeHTML = (value) => String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

// How each place with the Hardcover sync reads in the table; the sentence behind it is the cell's tooltip.
const HARDCOVER_TEXT = { synced: "Synced", doubt: "Check match", linked: "Linked", nomatch: "Not on Hardcover", ignored: "Left out" };

// A spacer stands in for the rows that are not drawn. It is as tall as they would be.
export const spacerHTML = (height, columns) => (height > 0 ? `<tr class="ed-spacer" aria-hidden="true" style="height:${height}px"><td colspan="${columns}"></td></tr>` : "");

// rowHTML is one book. `view` is the row with the person's unsaved edits applied; `columns` and `editable` are the shown
// columns and the ones that can be edited. In the comfortable view every cell is clipped to the row's height, so all rows
// are exactly as tall as the table expects.
export function rowHTML(row, view, index, { columns, editable, state }) {
  const clip = state.density === "comfortable" ? (inner) => `<div class="ed-clip">${inner}</div>` : (inner) => inner;
  const changed = state.pending[row.id] || {};
  const cells = columns.map((column) => {
    if (column.field) {
      const col = editable.indexOf(column);
      const focus = state.focus.row === index && state.focus.col === col;
      return `<td data-field="${column.field}" data-row="${index}" data-col="${col}" data-changed="${column.field in changed}" data-focus="${focus}" class="${column.num ? "ed-num" : ""}" title="${escapeHTML(fieldValue(view, column.field))}">${clip(escapeHTML(cellText(view, column.field)))}</td>`;
    }
    const plain = (text, title = "", num = false) => `<td${num ? ' class="ed-num"' : ""}${title ? ` title="${escapeHTML(title)}"` : ""}>${clip(escapeHTML(text))}</td>`;
    switch (column.id) {
      case "progress": return plain(progressText(row), "", true);
      case "words": case "pages": return plain(countText(row[column.id]), "", true);
      case "match": return plain(row.match ? `${row.match.title} · ${scoreText(row.match.score)}${row.match_clear ? "" : " ?"}` : "", row.match_note);
      case "isbn_found": return plain(row.isbn_found || "", row.isbn_note);
      case "format": return plain((row.format || "").toUpperCase());
      case "hardcover": return plain(HARDCOVER_TEXT[row.hardcover] || "", row.hardcover_note)
      case "collections": return plain((row.collections || []).join(", "), (row.collections || []).join(", "));
      case "folder": return plain(folderOf(row), folderOf(row));
      case "library_folder": return plain(state.libraryFolder, state.libraryFolder);
      case "path": return plain(row.path, row.path);
    }
    const marked = state.resets.has(row.id) ? '<span class="ed-badge edited">Will reset</span>' : row.edited ? '<span class="ed-badge edited">Edited</span>' : "";
    const found = (row.issues || []).map((issue) => badgeHTML(issue, issueText(issue), row.id, escapeHTML)).join("");
    const badges = found + marked || okBadgeHTML();
    return `<td>${clip(badges)}</td>`;
  }).join("");
  const cover = state.density === "comfortable" ? `<img loading="lazy" alt="" src="/covers/${encodeURIComponent(state.library)}/${encodeURIComponent(row.id)}">` : "";
  return `<tr data-id="${escapeHTML(row.id)}" data-selected="${state.selected.has(row.id)}"><td><input type="checkbox" data-select="${escapeHTML(row.id)}" aria-label="Select ${escapeHTML(row.title)}" ${state.selected.has(row.id) ? "checked" : ""}></td><td>${cover}</td>${cells}<td class="ed-fill"></td></tr>`;
}
