// The Hermes chat is an optional module. It is offered in the Home/library menu only when the module is on; any
// failure to find out leaves it out.
import { moduleEnabled } from "./module-enabled.js?v=1";

export async function hermesAvailable(fetchImpl) {
  try {
    if (!(await moduleEnabled("hermes", fetchImpl))) return false;
    // Only administrators may chat with Hermes; anyone else is told no and never sees the entry.
    return (await fetchImpl("/api/hermes/status")).ok;
  } catch (_) {
    return false;
  }
}

export function createHermesMenu({ onChange = () => {}, fetchImpl = null, open = () => globalThis.location?.assign("/hermes") } = {}) {
  let available = false;
  const menu = {
    available: () => available,
    entries: () => (available ? [{ id: "__hermes__", name: "Hermes chat" }] : []),
    open,
    setAvailable(value) {
      if (available === value) return;
      available = value;
      onChange();
    },
  };
  if (fetchImpl) void hermesAvailable(fetchImpl).then((value) => menu.setAvailable(value));
  return menu;
}

/** The optional-module entries of the Home/library menu (Library editor, Hermes chat) as props for the toolbar. */
export function createModuleMenus({ createEditorMenu, document, fetchImpl, onChange }) {
  const editor = createEditorMenu({ document, fetchImpl, onChange });
  const hermes = createHermesMenu({ fetchImpl, onChange });
  return { ...editor.toolbar, extraDestinations: () => [...editor.entries(), ...hermes.entries()], openHermes: hermes.open };
}
