export function createLibraryState({ storage = localStorage } = {}) {
  let allLibraries = [];
  let visibleLibraries = [];
  let selectedLibrary = "";
  let selectionReady = false;

  const persistSelection = () =>
    storage.setItem("mochil-library", selectedLibrary);

  function select(id) {
    selectedLibrary = id || "";
    persistSelection();
  }

  function initializeSelection() {
    if (selectionReady) return selectedLibrary;
    const saved = storage.getItem("mochil-library");
    // Library is the landing page, so the first thing opened must be a library of books -- not the notes vault you
    // happened to visit last. Notes stay reachable from the dock.
    const readable = visibleLibraries.filter((library) => !/wiki|documents|vault|notes/i.test(library.id || ""));
    const pool = readable.length ? readable : visibleLibraries;
    selectedLibrary = pool.some((library) => library.id === saved)
      ? saved
      : pool[0]?.id || "";
    selectionReady = true;
    persistSelection();
    return selectedLibrary;
  }

  function ensureVisible() {
    if (
      (!selectedLibrary && visibleLibraries.length > 0) ||
      (selectedLibrary &&
        !visibleLibraries.some((library) => library.id === selectedLibrary))
    ) {
      select(visibleLibraries[0]?.id || "");
    }
    return selectedLibrary;
  }

  return {
    all: () => allLibraries,
    visible: () => visibleLibraries,
    selected: () => selectedLibrary,
    selectedRecord: () =>
      visibleLibraries.find((library) => library.id === selectedLibrary),
    setAll: (libraries) => (allLibraries = libraries),
    setVisible: (libraries) => (visibleLibraries = libraries),
    select,
    initializeSelection,
    ensureVisible,
  };
}
