// The library editor is an optional module for administrators. It is offered in the Home/library menu only when it is
// on and this person may use it; any failure leaves it out. It is a desktop tool (wide table, keyboard editing), so a
// phone or touch tablet never offers it.
import { moduleEnabled } from "./module-enabled.js?v=1";

export async function editorAvailable(document, fetchImpl) {
  const view = document.defaultView;
  if (typeof view?.matchMedia === "function" && !view.matchMedia("(min-width: 900px) and (hover: hover) and (pointer: fine)").matches) return false;
  try {
    if (!(await moduleEnabled("editor", fetchImpl))) return false;
    const probe = await fetchImpl("/api/metadata/editor/books?library=-&limit=1");
    return !(probe.status === 403 || probe.status === 401 || probe.status >= 500);
  } catch (_) {
    return false;
  }
}

const storageKey = "mochil-editor-in-menu-v1";

/** Whether the editor is a page in the Home/library menu: offered by the server, and not hidden on this device. */
export function createEditorMenu({ storage = globalThis.localStorage, onChange = () => {}, document = null, fetchImpl = null, open = () => globalThis.location?.assign("/editor") } = {}) {
  let available = false;
  const read = () => {
    try { return storage?.getItem(storageKey) !== "0"; } catch (_) { return true; }
  };
  let shown = read();
  const entries = () => (available && shown ? [{ id: "__editor__", name: "Library editor" }] : []);
  const menu = {
    available: () => available,
    shown: () => shown,
    entries,
    // What the Home/library menu needs: the extra entry, and what choosing it does (it opens in this tab, like Hermes,
    // so the app stays one place; the editor has its own Back arrow).
    toolbar: { extraDestinations: entries, openEditor: open },
    setAvailable(value) {
      if (available === value) return;
      available = value;
      onChange();
    },
    setShown(value) {
      shown = Boolean(value);
      try { storage?.setItem(storageKey, shown ? "1" : "0"); } catch (_) { /* a blocked store only forgets the choice */ }
      onChange();
    },
  };
  if (document && fetchImpl) void startEditorMenu({ document, fetchImpl, menu });
  return menu;
}

/** Finds out whether the editor is offered, and wires the Settings checkbox that hides its menu entry on this device. */
export function startEditorMenu({ document, fetchImpl, menu }) {
  const toggle = document.querySelector("#editor-menu-toggle");
  const input = toggle?.querySelector("input");
  if (input) input.onchange = () => menu.setShown(input.checked);
  return editorAvailable(document, fetchImpl).then((available) => {
    if (toggle) toggle.hidden = !available;
    if (input) input.checked = menu.shown();
    menu.setAvailable(available);
  });
}
