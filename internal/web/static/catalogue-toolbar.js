import { captureCataloguePlace, restoreCataloguePlace } from "./catalogue-place.js?v=1";

export function createCatalogueToolbar({
  $,
  escapeHTML,
  storage = localStorage,
  libraryState,
  getCollectionMode,
  setCollectionMode,
  getBrowseMode,
  getSeriesLayout,
  setSeriesLayout,
  getItemLayout,
  setItemLayout,
  isVideoLibrary,
  isAudioLibrary,
  isDocumentLibrary,
  isVaultLibrary = () => false,
  renderAlphabet,
  loadCurrentView,
  reflowCollection = () => false,
  capturePlace = () => captureCataloguePlace(),
  restorePlace = (place) => restoreCataloguePlace(place),
  getActiveView = () => 'library',
  openHome = () => {},
  openLibrary = () => {},
  openSettings = () => {},
  openEditor = () => {},
  openHermes = () => {},
  extraDestinations = () => [],
  onRendered = () => {},
}) {
  // One Covers/List switch for whatever shelf is open. The per-shelf selects
  // stay in the page (hidden) as the state video and document shelves read.
  function activeLayout() {
    if (isVideoLibrary()) return $("#video-layout").value;
    if (isDocumentLibrary()) return $("#document-layout").value;
    return getItemLayout();
  }

  function renderBrowseControls() {
    $("#series-layout").value = getSeriesLayout();
    $("#item-layout").value = getItemLayout();
    const layout = activeLayout();
    $("#view-menu")
      .querySelectorAll("[data-layout]")
      .forEach((button) =>
        button.setAttribute("aria-pressed", String(button.dataset.layout === layout)),
      );
  }

  function chooseLayout(value) {
    for (const id of isVideoLibrary() ? ["#video-layout"] : isDocumentLibrary() ? ["#document-layout"] : []) {
      $(id).value = value;
      $(id).onchange?.({ target: $(id) });
      renderBrowseControls();
      return;
    }
    setSeriesLayout(value);
    setItemLayout(value);
    storage.setItem("bobobooks-series-layout", value);
    storage.setItem("bobobooks-item-layout", value);
    renderBrowseControls();
    if (getCollectionMode() && reflowCollection()) return;
    const place = capturePlace();
    loadCurrentView();
    restorePlace(place);
  }

  function render() {
    const libraries = libraryState.visible();
    // Settings and the editor are pages in this menu rather than icons in the top bar.
    const options = [{ id: "__home__", name: "Home" }, ...libraries, ...extraDestinations(), { id: "__settings__", name: "Settings" }];
    const selector = $("#filters");
    selector.innerHTML = options
      .map(
        (item) =>
          `<option value="${escapeHTML(item.id)}" data-kind="${escapeHTML(item.kind || ({ __home__: "home", __settings__: "settings", __editor__: "editor", __hermes__: "hermes" })[item.id] || "")}"${Number.isFinite(item.count) ? ` data-count="${item.count}"` : ""}>${escapeHTML(item.name)}</option>`,
      )
      .join("");
    selector.value = getActiveView() === 'dashboard' ? '__home__' : getActiveView() === 'settings' ? '__settings__' : libraryState.selected();
    const videoShelf = isVideoLibrary();
    const audioShelf = isAudioLibrary();
    const documentShelf = isDocumentLibrary();
    const vaultShelf = isVaultLibrary();
    const collectionMode = getCollectionMode();
    const publicationDestinations = $("#publication-destinations");
    if (publicationDestinations)
      publicationDestinations.hidden = videoShelf || audioShelf || documentShelf;
    $("#view-menu").hidden = audioShelf || vaultShelf;
    $(".browse-controls").hidden = audioShelf;
    const audioBrowseHead = $(".audio-browse-head");
    if (audioBrowseHead && !audioShelf) audioBrowseHead.hidden = true;
    $("#search").placeholder = documentShelf
      ? "Document title or folder"
      : "Titles, authors, series";
    $("#search-panel .sr-only").textContent = documentShelf
      ? "Search documents"
      : "Search books";
    renderBrowseControls();
    onRendered();
  }

  function bind() {
    $("#filters").onchange = (event) => {
      if(event.target.value==='__home__'){openHome();event.target.focus();return}
      if (event.target.value === "__settings__") { openSettings(); event.target.focus(); return; }
      if (event.target.value === "__hermes__") { openHermes(); render(); event.target.focus(); return; }
      if (event.target.value === "__editor__") { openEditor(); render(); event.target.focus(); return; }
      // Re-picking the library already on screen is a no-op; from Settings or
      // Home it still has to take you there.
      if (event.target.value === libraryState.selected() && getActiveView() === "library") return;
      setCollectionMode(false);
      libraryState.select(event.target.value);
      render();
      event.target.focus();
      openLibrary();
    };
    $("#view-menu")
      .querySelectorAll("[data-layout]")
      .forEach((button) => {
        button.onclick = () => chooseLayout(button.dataset.layout);
      });
    renderBrowseControls();
  }

  return { bind, render, renderBrowseControls };
}
