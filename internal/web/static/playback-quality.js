const qualities = new Set(['maximum', 'high', 'medium', 'low'])

export function normalizePlaybackQuality(value, fallback = 'maximum') {
  return qualities.has(value) ? value : fallback
}

export function effectivePlaybackQuality(navigatorObject = globalThis.navigator, storage = globalThis.localStorage) {
  const connection = navigatorObject?.connection
  const mobile = connection?.type === 'cellular' || connection?.saveData === true
  const key = mobile ? 'bobobooks-playback-mobile-quality' : 'bobobooks-playback-wifi-quality'
  return normalizePlaybackQuality(storage?.getItem(key), mobile ? 'high' : 'maximum')
}

export function qualityHLSURL(path, quality) {
  const normalized = normalizePlaybackQuality(quality, 'high')
  const url = new URL(path, globalThis.location?.origin || 'http://localhost')
  const parts = url.pathname.split('/').filter(Boolean)
  if (parts.at(-1) !== 'index.m3u8') return path
  parts.splice(parts.length - 1, 0, normalized)
  url.pathname = `/${parts.join('/')}`
  return `${url.pathname}${url.search}${url.hash}`
}

export async function loadPlaybackQualityPolicy(requestJSON, storage = globalThis.localStorage) {
  try {
    const { data } = await requestJSON('/api/playback-policy')
    storage?.setItem('bobobooks-playback-wifi-quality', normalizePlaybackQuality(data?.wifi_quality, 'maximum'))
    storage?.setItem('bobobooks-playback-mobile-quality', normalizePlaybackQuality(data?.mobile_quality, 'high'))
  } catch (_) {
    // Defaults remain useful while offline or during a rolling server update.
  }
}

const targetKbps = { high: 192, medium: 128, low: 80 }

// worthConverting says whether a converted stream would actually be smaller. A file that is already close to or below
// the target gains nothing from being re-encoded, and re-encoding costs the server CPU and the listener a delay, so it
// is played as it is. A file whose bitrate is unknown is converted, as before.
export function worthConverting(item, quality) {
  const target = targetKbps[quality]
  const bitrate = Number(item?.bitrate)
  if (!target || !(bitrate >= 1000)) return Boolean(target)
  return bitrate > target * 1000 * 1.1
}
