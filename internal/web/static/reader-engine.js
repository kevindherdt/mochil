// Which engine opens EPUBs: Readium (the default) or the classic reader, kept as a hidden escape
// hatch: there is no Settings row for it, but ?engine=classic (or a previously saved choice) still works.
import { highlightColor, highlightColors, highlightKey } from "./read-along-colors.js?v=1";
export const engineKey = "mochil-reader-engine";
const engines = new Set(["classic", "readium"]);

export function readerEngine(storage, search = "") {
  const asked = new URLSearchParams(search).get("engine");
  if (engines.has(asked)) return asked;
  try {
    const saved = storage.getItem(engineKey);
    return engines.has(saved) ? saved : "readium";
  } catch {
    return "readium";
  }
}

export const readerEngineHTML = `
      <label>Read-along highlight<select id="read-along-color">${Object.entries(highlightColors).map(([key, [name]]) => `<option value="${key}">${name}</option>`).join("")}</select></label>
      <small>The colour of the sentence being read aloud.</small>`;

export function mountReaderEngine(document, storage) {
  const card = document.querySelector(".appearance-settings");
  if (!card || card.querySelector("#read-along-color")) return;
  card.insertAdjacentHTML("beforeend", readerEngineHTML);
  const color = card.querySelector("#read-along-color");
  color.value = highlightColor(storage);
  color.addEventListener("change", () => {
    try { storage.setItem(highlightKey, color.value); } catch {}
  });
}
