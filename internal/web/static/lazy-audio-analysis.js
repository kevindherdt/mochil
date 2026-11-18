const supportedKinds = new Set(['music', 'audiobooks'])

// Loudness analysis is an administrative tool. Keep its controller, polling,
// and first API request outside the ordinary listening path until requested.
export function createLazyAudioAnalysis({ container, button, status, requestJSON }) {
  let controller
  let loading
  let library = ''
  let kind = ''

  const enabled = () => Boolean(library) && supportedKinds.has(kind)

  async function ensure() {
    if (!controller) {
      loading ||= import('./audio-analysis-controller.js?v=2').then(({ createAudioAnalysisController }) =>
        createAudioAnalysisController({ container, button, status, requestJSON }))
      controller = await loading
    }
    controller.setContext(library, kind)
    return controller
  }

  button.addEventListener('click', () => {
    if (enabled() && !controller) void ensure()
  })

  return {
    setContext(nextLibrary, nextKind) {
      library = nextLibrary || ''
      kind = nextKind || ''
      if (controller) {
        controller.setContext(library, kind)
        return
      }
      container.hidden = !enabled()
      button.disabled = false
      button.textContent = 'Check loudness'
      status.textContent = ''
    },
    dispose() {
      controller?.dispose()
    },
  }
}
