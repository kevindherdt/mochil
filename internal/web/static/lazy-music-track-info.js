import { loadFeatureStyles } from './feature-style-loader.js?v=1'

export function createLazyMusicTrackInfo({ document, escapeHTML, loadArtwork }) {
  let controller
  let loading

  async function ensure() {
    if (controller) return controller
    loading ||= Promise.all([
      loadFeatureStyles(document, ['/assets/music-track-info.css?v=2']),
      import('./music-track-info.js?v=4'),
    ]).then(([, { createMusicTrackInfo }]) => {
      controller = createMusicTrackInfo({ document, escapeHTML, loadArtwork })
      return controller
    })
    return loading
  }

  return {
    async open(item) {
      const value = await ensure()
      value.open(item)
    },
  }
}
