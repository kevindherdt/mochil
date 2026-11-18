let modulePromise

export async function loadSpokenMediaView() {
  modulePromise ||= import('./audio-media-views.js?v=5')
  return modulePromise
}
