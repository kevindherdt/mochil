const storageKey = "mochil-hidden-libraries-v1";
const maximumLibraries = 512;

function parseHiddenLibraries(value) {
  if (!value) return [];
  try {
    const parsed = JSON.parse(value);
    if (!Array.isArray(parsed)) return [];
    return [
      ...new Set(
        parsed.filter(
          (id) => typeof id === "string" && id.length > 0 && id.length <= 256,
        ),
      ),
    ].slice(0, maximumLibraries);
  } catch {
    return [];
  }
}

export function createLibraryVisibility({
  storage = localStorage,
  onChange = () => {},
} = {}) {
  let libraries = [];
  let hidden = new Set(parseHiddenLibraries(storage.getItem(storageKey)));
  const visible = () => libraries.filter((library) => !hidden.has(library.id));
  const isVisible = (libraryID) => !hidden.has(libraryID);
  const save = () => storage.setItem(storageKey, JSON.stringify([...hidden]));

  function setLibraries(nextLibraries) {
    libraries = Array.isArray(nextLibraries) ? nextLibraries : [];
    const known = new Set(libraries.map((library) => library.id));
    const cleaned = new Set([...hidden].filter((id) => known.has(id)));
    if (cleaned.size !== hidden.size) {
      hidden = cleaned;
      save();
    }
    return visible();
  }

  function setVisible(libraryID, shown) {
    if (shown === isVisible(libraryID)) return;
    if (shown) hidden.delete(libraryID);
    else hidden.add(libraryID);
    save();
    onChange(visible());
  }
  function show(ids) {
    let changed = false;
    for (const id of ids || []) changed = hidden.delete(id) || changed;
    if (changed) {
      save();
      onChange(visible());
    }
  }
  const count = (libraryID) => libraries.find((library) => library.id === libraryID)?.count;
  return { isVisible, setVisible, count, setLibraries, visible, show };
}

export { parseHiddenLibraries };
