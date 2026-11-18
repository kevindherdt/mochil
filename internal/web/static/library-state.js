export function createLibraryState({ storage = localStorage } = {}) {
  let allLibraries = [];
  let visibleLibraries = [];
  let selectedLibrary = "";
  let selectionReady = false;

  const persistSelection = () =>
    storage.setItem("bobobooks-library", selectedLibrary);

  function select(id) {
    selectedLibrary = id || "";
    persistSelection();
  }

  function initializeSelection() {
    if (selectionReady) return selectedLibrary;
    const saved = storage.getItem("bobobooks-library");
    selectedLibrary = visibleLibraries.some((library) => library.id === saved)
      ? saved
      : visibleLibraries[0]?.id || "";
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
