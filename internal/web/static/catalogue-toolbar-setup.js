// Builds the catalogue toolbar, which needs a dozen answers about the current
// view. Keeping the wiring here leaves the app shell readable and the toolbar's
// inputs in one place.
import { createCatalogueToolbar } from "./catalogue-toolbar.js?v=16";
import { createModuleMenus } from "./hermes-link.js?v=3";
import { captureCataloguePlace, restoreCataloguePlace } from "./catalogue-place.js?v=1";

export function createCatalogueToolbarFor({
  document,
  window,
  $,
  $$,
  escapeHTML,
  fetchImpl,
  createEditorMenu,
  libraryState,
  collectionsController,
  viewController,
  getToolbar,
  publicationDestinations,
  state,
  helpers,
}) {
  // Where the visible catalogue starts: under the sticky masthead/toolbars.
  const catalogueTop = () =>
    Number.parseFloat(
      globalThis.getComputedStyle?.(document.documentElement)?.getPropertyValue("--catalog-sticky-top") || "",
    ) || 0;

  const toolbar = createCatalogueToolbar({
    ...createModuleMenus({
      createEditorMenu,
      document,
      fetchImpl,
      onChange: () => getToolbar()?.render(),
    }),
    openSettings: () => viewController.show("settings"),
    reflowCollection: () => collectionsController.reflow(),
    capturePlace: () => captureCataloguePlace(document, catalogueTop()),
    restorePlace: (place) => restoreCataloguePlace(place, { document, window }),
    $,
    $$,
    escapeHTML,
    libraryState,
    ...state,
    ...helpers,
    openHome: () => viewController.show("dashboard"),
    openLibrary: () => viewController.show("library"),
    onRendered: () => publicationDestinations.sync(),
  });
  toolbar.bind();
  return toolbar;
}
