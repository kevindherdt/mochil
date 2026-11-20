import { createBookCards } from "./book-cards.js?v=16";
import { openSettingsWhenEmpty } from "./first-run.js?v=1";
import { createCollections } from "./collections.js?v=22";
import { createJumpBackIn } from "./jump-back-in.js?v=24";
import { createDeferredVideoController } from "./deferred-video-controller.js?v=20";
import { createDeferredAudioController } from "./deferred-audio-controller.js?v=70";
import { createDeferredRouteControllers } from "./deferred-route-controllers.js?v=122";
import { createContinuousPager } from "./continuous-pager.js?v=4";
import { createLibraryViewRouter } from "./library-view-router.js?v=6";
import { createBookSelection } from "./book-selection.js?v=5";
import { createCatalogAlphabet } from "./catalog-alphabet.js?v=4";
import { configureAlphabetRail } from "./catalog-alphabet-rail.js?v=1";
import { createMediaSectionNavigator } from "./media-section-navigator.js?v=2";
import { createBoundedWindow } from "./bounded-window.js?v=1";
import { requestJSON } from "./http-json.js?v=2";
import { createHomeRails } from "./home-rails.js?v=2";
import { createCardTint } from "./card-tint.js?v=5";
import { createRequestCache } from "./request-cache.js?v=2";
import { loadPlaybackQualityPolicy } from "./playback-quality.js?v=2";
import { createLibraryVisibility } from "./library-visibility.js?v=4";
import { createActivityMonitor } from "./activity-monitor.js?v=5";
import { createHeldRail } from "./held-rail.js?v=6";
import { createThemeController } from "./theme-controller.js?v=17";
import { createSearchController } from "./search-controller.js?v=3";
import { bindPopupDismissal } from "./popup-controller.js?v=1";
import { createShellLayout } from "./shell-layout.js?v=2";
import { createAppViewController } from "./app-view-controller.js?v=2";
import { createViewRoute } from "./view-route.js?v=6";
import { registerServiceWorker } from "./register-sw.js?v=2";
import { errorSpec, renderStatePanel } from "./state-panel.js?v=2";
import { createCatalogueToolbarFor } from "./catalogue-toolbar-setup.js?v=1";
import { createLibraryLoader } from "./library-loader.js?v=5";
import { createLibraryState } from "./library-state.js?v=2";
import { createPublicationCatalogues } from "./publication-catalogues.js?v=29";
import { createSpatialNavigation } from "./spatial-navigation.js?v=1";
import { createStatusToast } from "./status-toast.js?v=1";
import { createLibraryMenu } from "./library-menu.js?v=7";
import { createAlphabetPeek } from "./alphabet-peek.js?v=1";
import { createPublicationDestinations } from "./publication-destinations.js?v=9";
import { escapeHTML } from "./html.js?v=1";
import { installViewportActionMenus } from "./viewport-action-menus.js?v=2";
import { createEditorMenu } from "./editor-link.js?v=7";
import { followKeyboard } from "./keyboard-viewport.js?v=1";

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];
$("#shell-context").append($(".library-tools"));

const libraryState = createLibraryState();
let browseMode = localStorage.getItem("mochil-browse-by") || "series";
if (!["discover", "series", "authors", "books"].includes(browseMode)) browseMode = "series";
let seriesLayout = localStorage.getItem("mochil-series-layout") || "covers";
let itemLayout = localStorage.getItem("mochil-item-layout") || "covers";
let collectionMode = false;
let activeBucket = "";
let bookCatalogue;
let seriesCatalogue, authorCatalogue, continueCatalogue;
createActivityMonitor({ document, $, escapeHTML, requestJSON });
installViewportActionMenus({ document, window }); followKeyboard(window);

const {
  author,
  bookmarkIcon,
  bookMarkup,
  installCoverFallbacks,
  releaseCovers,
} = createBookCards({ escapeHTML });
const collectionsController = createCollections({
  document, $, $$, escapeHTML, requestJSON, bookMarkup, installCoverFallbacks, releaseCovers,
  createBoundedWindow, createContinuousPager,
  getSelectedLibrary: libraryState.selected,
  getItemLayout: () => itemLayout,
  isActive: () => collectionMode && !$("#library-view").hidden,
  abortBrowse: () => {
    bookCatalogue?.abort(); seriesCatalogue?.abort(); authorCatalogue?.abort(); continueCatalogue?.abort();
  },
});
collectionsController.bind();
// Kept so a shelf change can drop a selection pointing at books that are no longer on screen.
const bookSelection = createBookSelection({ document, collections: collectionsController, reload: () => loadCurrentView() });
loadPlaybackQualityPolicy(requestJSON);

let libraryLoader;
const libraryVisibility = createLibraryVisibility({
  onChange: (visibleLibraries) => libraryLoader.applyVisible(visibleLibraries),
});
// A rolling deploy can leave an older dependency loaded; saving must still work.
const activateLibraries = (ids) =>
  typeof libraryVisibility.show === "function" ? libraryVisibility.show(ids) : libraryLoader?.load();

// Every Home shelf reads the same lists; one in-flight GET per URL is enough.
const homeRequests = createRequestCache(requestJSON), homeRails = createHomeRails({
  requests: homeRequests, document, $, $$, escapeHTML, releaseCovers, installCoverFallbacks,
  isLibraryVisible: libraryVisibility.isVisible, createJumpBackIn, createHeldRail,
});
// Every card on every shelf takes its glow from its own artwork; one observer serves them all.
createCardTint({ root: document.body });
const jumpBackIn = homeRails.jumpBackIn;
const mediaNavigator = createMediaSectionNavigator($("#media-alphabet"));
const videoLibrary = createDeferredVideoController({
  document,
  $,
  escapeHTML,
  requestJSON: homeRequests.json,
  isLibraryVisible: libraryVisibility.isVisible,
  setNavigation: (...args) => {
    if (!$("#video-index").hidden) mediaNavigator.show(...args);
  },
});
const audioLibrary = createDeferredAudioController({
  document,
  $,
  escapeHTML,
  requestJSON: homeRequests.json,
  isLibraryVisible: libraryVisibility.isVisible,
  setNavigation: (...args) => {
    if (!$("#audio-index").hidden) mediaNavigator.show(...args);
  },
});
const deferredRoutes = createDeferredRouteControllers({
  document, $, $$, escapeHTML, requestJSON, libraryVisibility,
  setDocumentNavigation: (...args) => {
    if (!$("#document-index").hidden) mediaNavigator.show(...args);
  },
  onLibraryActivated: activateLibraries,
  onSettingsSaved: () => libraryLoader.load(),
});
createSpatialNavigation({container:$("#books"),selector:".book-open"});
createSpatialNavigation({container:$("#series-index"),selector:".series-card>button:first-child,.series-row>summary"});
createSpatialNavigation({container:$("#videos"),selector:".video-open,.video-show>summary,.video-season>summary"});
const selectedLibraryKind = () => libraryState.selectedRecord()?.kind || "";
const selectedMetadataProvider = () =>
  libraryState.selectedRecord()?.metadata_provider || "";
const isVideoLibrary = () =>
  ["movies", "tv", "anime"].includes(selectedLibraryKind());
const isPlannedAudioLibrary = () =>
  ["audiobooks", "podcasts", "music"].includes(selectedLibraryKind());
const isDocumentLibrary = () =>
  ["documents", "vault"].includes(selectedLibraryKind());

// Under a non-title sort the rail filters by letter instead of jumping to offsets.
function renderAlphabet() {
  configureAlphabetRail(catalogAlphabet, {
    collectionMode,
    browseMode,
    sortedBooks: publicationCatalogues?.sortBar?.get(),
    buckets: alphabetBuckets.map((bucket) => ({ bucket })),
    activeBucket,
    navigation: () => bookCatalogue.navigation(),
  });
}

const alphabetBuckets = [..."#ABCDEFGHIJKLMNOPQRSTUVWXYZ"];
const catalogAlphabet = createCatalogAlphabet({
  element: $("#alphabet"),
  onFilter: (bucket) => {
    activeBucket = bucket;
    renderAlphabet();
    $(`.letter[data-bucket="${CSS.escape(activeBucket)}"]`)?.focus();
    loadCurrentView();
  },
  onJump: (bucket) => {
    if (browseMode === "books") {
      bookCatalogue.jump(bucket);
      return;
    }
    activeBucket = bucket;
    renderAlphabet();
    loadCurrentView();
  },
});



const publicationCatalogues = createPublicationCatalogues({
  $, document, escapeHTML, requestJSON, createBoundedWindow, createContinuousPager,
  cards: { markup: bookMarkup, bookmarkIcon, installCoverFallbacks, releaseCovers },
  collections: collectionsController, libraryState,
  getBrowseMode: () => browseMode, getSeriesLayout: () => seriesLayout,
  getItemLayout: () => itemLayout, getActiveBucket: () => activeBucket,
  onNavigationChange: renderAlphabet,
  onSortChange: (sort) => {
    // Leaving a sorted view for Title (the only jump order) drops the letter
    // filter that only exists for the other sorts.
    if (!sort) activeBucket = "";
    renderAlphabet();
  },
  setCurrentBucket: bucket => catalogAlphabet.setCurrent(bucket),
});
({ books: bookCatalogue, series: seriesCatalogue, authors: authorCatalogue, continuing: continueCatalogue } = publicationCatalogues);

const loadCurrentView = createLibraryViewRouter({
  $, libraryState, mediaNavigator, videoLibrary, audioLibrary, deferredRoutes, collections: collectionsController,
  catalogues: publicationCatalogues,
  getBrowseMode: () => browseMode, isCollectionMode: () => collectionMode,
  isVideoLibrary, isPlannedAudioLibrary, isDocumentLibrary, selectedLibraryKind, selectedMetadataProvider,
});

createAlphabetPeek({ document, window });
createStatusToast({ element: $("#status"), window });
createLibraryMenu({ document, select: $("#filters") });
const searchToggle = $("#search-toggle");
const searchController = createSearchController({
  masthead: $(".masthead"),
  panel: $("#search-panel"),
  toggle: searchToggle,
  input: $("#search"),
  label: () => (isDocumentLibrary() ? "Search documents" : "Search books"),
  onSearch: loadCurrentView,
});
let shellLayout, catalogueToolbar;
const publicationDestinations = createPublicationDestinations({
  element: $("#publication-destinations"), gestureSurface: $("#library-view"),
  getBrowseMode: () => browseMode, getCollectionMode: () => collectionMode,
  onSelect: destination => {
    collectionMode = destination === "collections";
    if (!collectionMode) { browseMode = destination; activeBucket = ""; localStorage.setItem("mochil-browse-by", destination); }
    renderAlphabet(); catalogueToolbar?.render(); loadCurrentView(); window.dispatchEvent(new Event("mochil:place"));
  },
});
createThemeController({
  document,
  root: document.documentElement,
  toggle: $("#theme-toggle"),
  storage: localStorage,
  media: matchMedia("(prefers-color-scheme: dark)"),
});
let viewRoute;
const viewController = createAppViewController({
  $,
  mediaNavigator,
  searchController,
  searchToggle,
  hasLibraries: () => libraryState.visible().length > 0,
  openSettings: deferredRoutes.openSettings,
  loadLibrary: loadCurrentView,
  loadDashboard: () => {
    homeRails.load();
    videoLibrary.loadContinue?.();
    audioLibrary.loadContinue?.();
  },
  onViewChanged: () => { catalogueToolbar?.render(); viewRoute?.write(); },
  updateLayout: () => shellLayout.updateStickyMetrics(),
});
viewController.bind();
viewRoute = createViewRoute({
  window,
  getView: () => viewController.current(),
  getLibrary: libraryState.selected,
  hasLibrary: (id) => libraryState.visible().some((library) => library.id === id),
  showHome: () => viewController.show("dashboard"),
  showSettings: () => viewController.show("settings"),
  getPlace: () => ({ tab: collectionMode ? "collections" : browseMode, collection: collectionMode ? collectionsController.openId() : "", detail: collectionMode ? "" : (browseMode === "series" ? seriesCatalogue : browseMode === "authors" ? authorCatalogue : null)?.opened() || "" }),
  showLibrary: (id, place) => {
    collectionMode = place?.tab === "collections";
    if (place?.tab && !collectionMode) { browseMode = place.tab; localStorage.setItem("mochil-browse-by", place.tab); }
    collectionsController.openOnLoad(place?.collection); (place?.tab === "series" ? seriesCatalogue : authorCatalogue)?.openOnLoad(place?.tab === "series" || place?.tab === "authors" ? place.detail : "");
    libraryState.select(id);
    viewController.show("library");
  },
});

catalogueToolbar = createCatalogueToolbarFor({
  onLibraryChanged: () => bookSelection.clear(),
  document,
  window: globalThis,
  $,
  $$,
  escapeHTML,
  fetchImpl: fetch,
  createEditorMenu,
  libraryState,
  collectionsController,
  viewController,
  getToolbar: () => catalogueToolbar,
  publicationDestinations,
  state: {
    getCollectionMode: () => collectionMode,
    setCollectionMode: (value) => (collectionMode = value),
    getBrowseMode: () => browseMode,
    getSeriesLayout: () => seriesLayout,
    setSeriesLayout: (value) => (seriesLayout = value),
    getItemLayout: () => itemLayout,
    setItemLayout: (value) => (itemLayout = value),
  },
  helpers: {
    isVideoLibrary,
    isAudioLibrary: isPlannedAudioLibrary,
    isDocumentLibrary,
    isVaultLibrary: () => selectedLibraryKind() === "vault",
    renderAlphabet,
    loadCurrentView,
    getActiveView: () => viewController.current(),
  },
});
libraryLoader = createLibraryLoader({
  requestJSON,
  libraryVisibility,
  videoLibrary,
  audioLibrary,
  libraryState,
  renderToolbar: () => catalogueToolbar.render(),
  renderAlphabet,
  isLibraryView: () => viewController.current() === "library",
  loadCurrentView,
  setStatus: (message) => ($("#status").textContent = message),
  onFirstRun: () => viewController.show("settings"),
  onReady: () => viewRoute.restoreInitial(),
  showProblem: (error, retry) => {
    const target = $("#app-problem");
    renderStatePanel(target, errorSpec(error, "your library"), { retry });
    target.hidden = false;
  },
  clearProblem: () => { $("#app-problem").hidden = true; },
});

bindPopupDismissal({
  document,
  podcastPanel: $("#podcast-manage"),
  podcastToggle: $("#podcast-manage-toggle"),
});
shellLayout = createShellLayout({
  window,
  root: document.documentElement,
  body: document.body,
  masthead: $(".masthead"),
  tools: $(".library-tools"),
  libraryView: $("#library-view"),
  ResizeObserver,
  onScroll: (top) => {
    if (collectionMode) return;
    if (browseMode === "books" && !$("#book-catalog").hidden) bookCatalogue.updateCurrentFromScroll(top);
    else if (browseMode === "authors" && !$("#series-index").hidden) authorCatalogue.updateCurrentFromScroll(top, bucket => catalogAlphabet.setCurrent(bucket));
    else if (browseMode === "series" && !$("#series-index").hidden) seriesCatalogue.updateCurrentFromScroll(top, bucket => catalogAlphabet.setCurrent(bucket));
  },
});
libraryLoader.load();
registerServiceWorker();

openSettingsWhenEmpty({ location });
