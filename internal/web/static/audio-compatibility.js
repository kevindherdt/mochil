import { loadHLSClient } from './hls-client.js?v=1'
import { qualityHLSURL } from './playback-quality.js?v=2'

export function createAudioCompatibility({ loadClient = loadHLSClient, setStatus = () => {} } = {}) {
  let player
  function reset() { player?.destroy(); player = undefined }
  async function attach({ item, audio, quality = 'high', resumeAt = 0, onReady = () => {}, onFatal = () => {} }) {
    if (!item?.compatibility_url || !audio) return false
    reset()
    const qualityURL = qualityHLSURL(item.compatibility_url, quality)
    const source = `${qualityURL}${qualityURL.includes('?')?'&':'?'}v=${encodeURIComponent(item.revision || 'current')}`
    setStatus('Preparing compatibility playback…')
    try {
      const Hls = await loadClient()
      if (Hls?.isSupported()) {
        audio.removeAttribute('src'); audio.load()
        player = new Hls({ maxBufferLength: 30, backBufferLength: 30, maxBufferSize: 48 * 1024 * 1024, startPosition: Number(resumeAt || -1) })
        player.attachMedia(audio)
        player.on(Hls.Events.MEDIA_ATTACHED, () => player?.loadSource(source))
        player.on(Hls.Events.MANIFEST_PARSED, onReady)
        player.on(Hls.Events.ERROR, (_event, data) => { if (data.fatal) onFatal() })
        return true
      }
    } catch (_) { /* Native HLS remains available without hls.js. */ }
    if (audio.canPlayType('application/vnd.apple.mpegurl')) {
      audio.src = source; audio.load()
      return true
    }
    setStatus('This audio format cannot be played on this device.')
    return false
  }
  return { attach, reset }
}
