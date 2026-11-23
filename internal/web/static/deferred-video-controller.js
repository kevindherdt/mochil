import { createVideoContinueRail, watchedLabel } from "./video-continue-rail.js?v=1";
import { mountHero } from "./home-hero.js?v=1";
import { mountVideoLibrary } from "./video-library-template.js?v=2";
import { lazyOnce, reportLoadFailure } from "./lazy-once.js?v=1";
import { loadFeatureStyles } from "./feature-style-loader.js?v=1";

// Home renders only its bounded Continue watching rail. Catalogue grouping,
// metadata review, HLS playback, and player chrome load on first video use.
export function createDeferredVideoController({ document, $, escapeHTML, requestJSON, isLibraryVisible, setNavigation }) {
  mountVideoLibrary(document);
  let controller;
  let generation = 0;

  const ensure = lazyOnce(() =>
    Promise.all([
      loadFeatureStyles(document, ["/assets/video-player.css?v=7", "/assets/video-controls.css?v=4"]),
      import("./video-library.js?v=67"),
    ]).then(([, { createVideoLibrary }]) => {
      controller = createVideoLibrary({ document, $, escapeHTML, requestJSON, isLibraryVisible, setNavigation });
      return controller;
    }),
  );
  const failed = (error) => reportLoadFailure(document, "Video", error);

  const addArtwork = (element, url) => {
    if (!element || !url) return;
    const image = new Image();
    image.alt = "";
    image.loading = "lazy";
    image.decoding = "async";
    image.fetchPriority = "low";
    image.onerror = () => image.remove();
    image.src = url;
    element.append(image);
  };
  const continueRail = createVideoContinueRail({
    $,
    requestJSON,
    escapeHTML,
    isLibraryVisible,
    playerController: { open: (item, title) => {
      const requestedGeneration = generation;
      void ensure().then(value => { if (requestedGeneration === generation) value.open(item, title); }).catch(failed);
    } },
    addArtwork,
    hero: mountHero({
      document,
      escaped: escapeHTML,
      watchedLabel: progress => watchedLabel(progress),
      onOpen: (item, title) => playerController.open(item, title),
    }),
  });

  return {
    load(...args) {
      const requestedGeneration = ++generation;
      void ensure().then(value => {
        if (requestedGeneration === generation) value.load(...args);
      }).catch(failed);
    },
    loadContinue: continueRail.load,
    abort() {
      generation += 1;
      controller?.abort();
    },
  };
}
