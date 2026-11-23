import { mountLibraryHub } from "./library-hub.js?v=1";
import { mountLibraryPills } from "./library-pills.js?v=1";
import { captureCataloguePlace, restoreCataloguePlace } from "./catalogue-place.js?v=1";

export function createCatalogueToolbar({
  $,
  escapeHTML,
  storage = localStorage,
  libraryState,
  getCollectionMode,
  setCollectionMode,
  // Called when the reader moves to another shelf or view, so a selection that points at the old one can go.
  onLibraryChanged = () => {},
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
    const buttons = $("#view-menu").querySelectorAll("[data-layout]");
    // One switch: the button names the layout it would give you, and shows the glyph of the one you are in. The
    // two-button group this replaced is kept working for anything still mounting that shape.
    if (buttons.length === 1) {
      const [toggle] = buttons;
      const target = layout === "list" ? "covers" : "list";
      toggle.dataset.layout = target;
      toggle.setAttribute("aria-pressed", String(layout === "list"));
      toggle.setAttribute("aria-label", target === "list" ? "Show as list" : "Show covers");
      return;
    }
    buttons.forEach((button) =>
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
    storage.setItem("mochil-series-layout", value);
    storage.setItem("mochil-item-layout", value);
    renderBrowseControls();
    if (getCollectionMode() && reflowCollection()) return;
    const place = capturePlace();
    loadCurrentView();
    restorePlace(place);
  }


  // The page names itself on every browse mode, not only when the flat book catalogue happens to render: the
  // Series and Authors views showed a bare page with no title at all.
  // The design's library switcher is a row of pills, not a menu: the same list the dock and the shelf menu use.
  const libraryPills = mountLibraryPills({
    document: globalThis.document,
    escapeHTML,
    libraries: () => libraryState.visible().filter((library) => !/wiki|documents|vault|notes/i.test(library.id || '')),
    selected: () => libraryState.selected(),
    onPick: (id) => { if (id && id !== libraryState.selected()) openShelf(id); },
  });

  const libraryHub = mountLibraryHub({
    document: globalThis.document,
    escapeHTML,
    libraries: () => libraryState.visible(),
    onPick: (id) => openShelf(id),
  });

  /** Opening a shelf goes through the shelf menu's own change handler, which also rewrites the route. */
  function openShelf(id) {
    libraryHub.hide();
    const select = $("#filters");
    if (select) {
      select.value = id;
      select.dispatchEvent(new Event("change", { bubbles: true }));
      return;
    }
    libraryState.select(id);
    loadCurrentView();
  }

  // The dock's Library button asks for the hub rather than the last shelf.
  globalThis.document?.addEventListener?.("mochil:library-hub", () => libraryHub.show());

  // The bar names where you are and offers the way back, as the design's sub-screens do.
  const backButton = $("#masthead-back");
  const mastheadTitle = $("#masthead-title");
  if (backButton) backButton.onclick = () => libraryHub.show();

  function syncMasthead() {
    // Home, Settings and Lyana are not shelves: the selected shelf is just whatever you looked at last, and it was
    // being used to name the bar ("Anime") and to show a back arrow on the dashboard.
    const view = getActiveView();
    const offShelf = view === "dashboard" || view === "settings" || view === "hermes";
    const inHub = libraryHub.isOpen() || !libraryState.selected();
    if (backButton) backButton.hidden = inHub || offShelf;
    // The pill row is the library switcher, so it belongs to shelves only: on Home, Settings and Lyana there is
    // nothing to switch between and the row was just a band of buttons floating over unrelated pages.
    const pillsNav = globalThis.document?.getElementById?.("library-pills");
    if (pillsNav) {
      const tooFew = (pillsNav.querySelectorAll?.(".library-pill")?.length || 0) < 2;
      pillsNav.hidden = offShelf || tooFew;
    }
    if (mastheadTitle) {
      const shelf = libraryState.visible().find((library) => library.id === libraryState.selected());
      mastheadTitle.textContent = offShelf ? "" : inHub ? "Library" : String(shelf?.name || "").trim();
      mastheadTitle.hidden = offShelf;
    }
  }

  function renderShelfHead() {
    // The toolbar is exercised headless in the tests, where there is no document: go through the injected $.
    // Both of these belong to the bar and the pill row, not to this heading. Rendering them after the guard meant
    // any view without #shelf-title (video, audio, the hub) kept whatever the pills said last time.
    libraryPills.render();
    syncMasthead();
    const title = $("#shelf-title");
    const meta = $("#shelf-meta");
    if (!title || !meta) return;
    const selector = $("#filters");
    const option = selector?.selectedOptions?.[0];
    const view = getActiveView();
    const name = view === "dashboard" ? "Home" : view === "settings" ? "Settings" : (option?.textContent || "").trim();
    title.textContent = name;
    // The menu knows how many items a shelf holds; the catalogue replaces this with the exact total once loaded.
    const count = Number(option?.dataset?.count);
    meta.textContent = Number.isFinite(count) && count > 0 ? `${count} ${count === 1 ? "item" : "items"}` : "";
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
    renderShelfHead();
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
      onLibraryChanged();
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
