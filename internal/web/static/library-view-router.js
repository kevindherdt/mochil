// Chooses which feature module fills the library view. Each kind of library
// is one branch with the same shape: stop what the others were doing, show your
// own index, load. Keeping the choice here means a new module is one more
// branch, and a failing module cannot reach into the others.
export function createLibraryViewRouter({
  $, libraryState, mediaNavigator, videoLibrary, audioLibrary, deferredRoutes, catalogues, collections,
  getBrowseMode, isCollectionMode, isVideoLibrary, isPlannedAudioLibrary, isDocumentLibrary, selectedLibraryKind, selectedMetadataProvider,
}) {
  return function loadCurrentView() {
    mediaNavigator.hide();
    $("#collection-index").hidden = true;
    $("#video-index").hidden = true;
    $("#audio-index").hidden = true;
    $("#document-index").hidden = true;
    catalogues.sortBar?.hide();
    $("#alphabet").hidden = isCollectionMode() || ["continue", "recent", "discover"].includes(getBrowseMode());
    if (!libraryState.visible().length) {
      catalogues.abortAll();
      videoLibrary.abort();
      audioLibrary.abort();
      deferredRoutes.abortDocument();
      $("#series-index").hidden = true;
      $("#book-catalog").hidden = true;
      $("#alphabet").hidden = true;
      $("#status").textContent =
        "No libraries are visible on this device. You can restore them in Settings.";
      return;
    }
    if (isVideoLibrary()) {
      catalogues.abortAll();
      deferredRoutes.abortDocument();
      $("#series-index").hidden = true;
      $("#book-catalog").hidden = true;
      $("#alphabet").hidden = true;
      $("#video-index").hidden = false;
      videoLibrary.load(
        libraryState.selected(),
        $("#search").value.trim(),
        true,
        selectedLibraryKind(),
        selectedMetadataProvider(),
      );
      return;
    }
    if (isPlannedAudioLibrary()) {
      catalogues.abortAll();
      videoLibrary.abort();
      deferredRoutes.abortDocument();
      $("#series-index").hidden = true;
      $("#book-catalog").hidden = true;
      $("#alphabet").hidden = true;
      $("#audio-index").hidden = false;
      audioLibrary.load(
        libraryState.selected(),
        $("#search").value.trim(),
        true,
        selectedLibraryKind(),
      );
      return;
    }
    if (isDocumentLibrary()) {
      catalogues.abortAll();
      videoLibrary.abort();
      audioLibrary.abort();
      $("#series-index").hidden = true;
      $("#book-catalog").hidden = true;
      $("#alphabet").hidden = true;
      $("#document-index").hidden = false;
      const selected = libraryState.selected();
      const kind = selectedLibraryKind();
      $("#status").textContent = "Opening documents…";
      void deferredRoutes.ensureDocument().then((library) => {
        if (!isDocumentLibrary() || libraryState.selected() !== selected) return;
        library.load(selected, $("#search").value.trim(), true, kind);
      }).catch((error) => {
        if (isDocumentLibrary() && libraryState.selected() === selected) {
          $("#status").textContent = `Could not open documents: ${error.message}`;
        }
      });
      return;
    }
    videoLibrary.abort();
    audioLibrary.abort();
    deferredRoutes.abortDocument();
    catalogues.authors.abort();
    if (isCollectionMode()) {
      collections.load();
      return;
    }
    if (getBrowseMode() === "series") catalogues.series.load();
    else if (getBrowseMode() === "authors") catalogues.authors.load();
    else if (getBrowseMode() === "continue") catalogues.continuing.load();
    else {
      $("#series-index").hidden = true;
      $("#book-catalog").hidden = false;
      // Discover is what is in progress, then what is new: the shelves in newest-first order, without the sort bar.
      if (getBrowseMode() !== "discover") catalogues.sortBar?.show();
      catalogues.books.load(true);
    }
  }
}
