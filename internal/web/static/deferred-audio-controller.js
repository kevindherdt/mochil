import { createAudioContinueRail } from "./audio-continue-rail.js?v=2";
import { createAudioItemView } from "./audio-item-view.js?v=13";
import { createAudioArtworkLoader } from "./audio-artwork-loader.js?v=8";
import { relativePublishedDate } from "./date-format.js?v=1";
import { mountAudioPlayer } from "./audio-player-template.js?v=8";
import { mountAudioLibrary } from "./audio-library-template.js?v=12";
import { lazyOnce, reportLoadFailure } from "./lazy-once.js?v=1";
import { createServerSave } from "./podcast-server-save.js?v=1";
import { createDeviceSaved } from "./podcast-device-saved.js?v=1";
import { createPlayerTint } from "./audio-player-tint.js?v=3";
import { watchAlbumTint } from "./album-tint.js?v=3";
import { loadFeatureStyles } from "./feature-style-loader.js?v=1";
import { hasSavedPlayer } from "./audio-player-restore.js?v=2";

// Home needs a bounded spoken-media rail, not the complete music catalogue and
// playback graph. Mount stable shell contracts up front and load the full
// controller only when playback or an audio shelf is actually requested.
export function createDeferredAudioController({ document, $, escapeHTML, requestJSON, isLibraryVisible, setNavigation }) {
  mountAudioPlayer(document);
  createPlayerTint({ player: document.getElementById("audio-player"), art: document.getElementById("audio-player-art"), document });
  mountAudioLibrary(document);
  watchAlbumTint($("#audio-items"));
  createServerSave({ container: $("#audio-items"), requestJSON, status: $("#podcast-status") });
  createDeviceSaved({ container: $("#audio-items") });
  let controller;
  let generation = 0;
  const artworkLoader = createAudioArtworkLoader();
  const { row } = createAudioItemView({
    escapeHTML,
    relativePublishedDate,
    playlists: () => [],
    isFavorite: () => false,
    browse: () => "",
  });
  const artwork = (element, url) => {
    if (!element || !url) return;
    element.dataset.artwork = url;
    artworkLoader.load(element, url);
  };

  const ensure = lazyOnce(() =>
    Promise.all([
      loadFeatureStyles(document, ["/assets/audio-library.css?v=2", "/assets/audio-player.css?v=28", "/assets/audio-player-wide.css?v=3", "/assets/audio-table.css?v=5", "/assets/music-discovery.css?v=6", "/assets/music-library.css?v=42", "/assets/podcast-badges.css?v=1"]),
      import("./audio-library.js?v=164"),
    ]).then(([, { createAudioLibrary }]) => {
      controller = createAudioLibrary({ document, $, escapeHTML, requestJSON, isLibraryVisible, setNavigation });
      return controller;
    }),
  );
  const failed = (error) => reportLoadFailure(document, "Audio", error);

  // A reload (back from the editor, a refreshed tab) brings the player bar back, paused, where it was.
  if (hasSavedPlayer()) void ensure().then(value => value.restore()).catch(failed);

  const continueRail = createAudioContinueRail({
    rail: $("#audio-reading-rail"),
    track: $("#audio-reading-track"),
    requestJSON,
    isLibraryVisible,
    row,
    artwork,
    onOpen: item => { void ensure().then(value => value.open(item)).catch(failed); },
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
