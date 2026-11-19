// Putting the editor's columns in the order a person wants, by dragging them in the Columns list.
export const applyOrder = (columns, order) => {
  const rank = (column) => { const at = order.indexOf(column.id); return at < 0 ? order.length : at; };
  return [...columns].sort((a, b) => rank(a) - rank(b));
};

// moveBefore puts `id` just before `beforeId` (or last when beforeId is empty) in the full list of ids.
export function moveBefore(all, id, beforeId) {
  const rest = all.filter((other) => other !== id);
  const at = beforeId ? rest.indexOf(beforeId) : -1;
  rest.splice(at < 0 ? rest.length : at, 0, id);
  return rest;
}

export function mountColumnDrag({ list, ids, onMove }) {
  let dragged = "";
  list.addEventListener("dragstart", (event) => {
    const row = event.target.closest("[data-column-row]");
    if (!row) return;
    dragged = row.dataset.columnRow;
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", dragged);
  });
  list.addEventListener("dragover", (event) => { if (dragged && event.target.closest("[data-column-row]")) event.preventDefault(); });
  list.addEventListener("drop", (event) => {
    const target = event.target.closest("[data-column-row]");
    if (!dragged || !target) return;
    event.preventDefault();
    const id = dragged; dragged = "";
    if (target.dataset.columnRow !== id) onMove(moveBefore(ids(), id, target.dataset.columnRow));
  });
  list.addEventListener("dragend", () => { dragged = ""; });
  // The same without a mouse: Alt+Arrow moves the focused column up or down.
  list.addEventListener("keydown", (event) => {
    const row = event.target.closest("[data-column-row]");
    if (!row || !event.altKey || !/^Arrow(Up|Down)$/.test(event.key)) return;
    event.preventDefault();
    const all = ids(), at = all.indexOf(row.dataset.columnRow), to = at + (event.key === "ArrowUp" ? -1 : 1);
    if (to < 0 || to >= all.length) return;
    onMove(moveBefore(all, row.dataset.columnRow, event.key === "ArrowUp" ? all[to] : all[to + 1] || ""), row.dataset.columnRow);
  });
}
