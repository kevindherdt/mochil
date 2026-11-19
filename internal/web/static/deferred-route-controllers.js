import { mountDocumentLibrary } from "./document-library-template.js?v=1";
import { loadFeatureStyles } from "./feature-style-loader.js?v=1";

// Owns feature controllers that are unnecessary on Home. Their small DOM
// contracts remain mounted so the shell can switch routes synchronously, while
// implementation graphs load only when the corresponding route is opened.
export function createDeferredRouteControllers({
  document,
  $,
  $$,
  escapeHTML,
  requestJSON,
  libraryVisibility,
  setDocumentNavigation,
  onLibraryActivated,
  onSettingsSaved,
}) {
  mountDocumentLibrary(document);
  let documentController;
  let documentLoading;
  let settingsController;
  let settingsLoading;

  function ensureDocument() {
    if (documentController) return Promise.resolve(documentController);
    if (!documentLoading) {
      documentLoading = Promise.all([
        loadFeatureStyles(document, ["/assets/document-library.css?v=2", "/assets/vault.css?v=16", "/assets/vault-nav.css?v=19", "/assets/vault-live.css?v=14", "/assets/vault-editor.css?v=17", "/assets/vault-home.css?v=2"]),
        import("./document-library.js?v=84"),
      ]).then(([, { createDocumentLibrary }]) => {
        documentController = createDocumentLibrary({ document, $, escapeHTML, requestJSON, setNavigation: setDocumentNavigation });
        return documentController;
      }).catch(error => {
        documentLoading = null; // a dropped connection must not break documents until reload
        throw error;
      });
    }
    return documentLoading;
  }

  function openSettings() {
    const view = $("#settings-view");
    if (settingsController) {
      if (!settingsController.isLoaded()) void loadSettings(settingsController, view);
      return;
    }
    view.inert = true;
    view.setAttribute("aria-busy", "true");
    $("#settings-status").textContent = "Opening settings…";
    if (!settingsLoading) {
      settingsLoading = Promise.all([
        loadFeatureStyles(document, ["/assets/metadata-review.css?v=8", "/assets/settings-controls.css?v=3", "/assets/settings-navigation.css?v=1", "/assets/settings-unified.css?v=28", "/assets/settings-modern.css?v=3", "/assets/settings-shell.css?v=2", "/assets/settings-cards.css?v=13", "/assets/settings-index.css?v=10"]),
        import("./settings.js?v=99"),
        import("./metadata-review.js?v=19"),
      ]).then(([, { createSettings }, { createMetadataReview }]) => {
        const metadataReview = createMetadataReview({ document, $, $$, escapeHTML, requestJSON });
        metadataReview.bind();
        settingsController = createSettings({
          document, $, $$, requestJSON, libraryVisibility,
          onLibraries: libraries => metadataReview.setLibraries(libraries),
          onLibraryActivated,
          onSaved: onSettingsSaved,
        });
        settingsController.bind();
        return settingsController;
      }).catch(error => {
        settingsLoading = null;
        throw error;
      });
    }
    void settingsLoading.then(async controller => {
      if (!controller.isLoaded()) await controller.open();
    }).catch(error => {
      $("#settings-status").textContent = `Could not open Settings: ${error.message}`;
    }).finally(() => {
      view.inert = false;
      view.removeAttribute("aria-busy");
    });
  }

  async function loadSettings(controller, view) {
    view.inert = true;
    view.setAttribute("aria-busy", "true");
    try {
      await controller.open();
    } finally {
      view.inert = false;
      view.removeAttribute("aria-busy");
    }
  }

  return {
    ensureDocument,
    abortDocument: () => documentController?.abort(),
    openSettings,
  };
}
