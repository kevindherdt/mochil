const modes = new Set(['off', 'track', 'album'])

export function replayGainFactor(item, mode = 'track') {
  if (!item || mode === 'off') return 1
  const album = mode === 'album'
  const gain = finite(album ? item.replaygain_album_db : item.replaygain_track_db)
    ?? finite(item.replaygain_track_db)
    ?? finite(item.replaygain_album_db)
  if (gain === null) return 1
  const peak = positive(album ? item.replaygain_album_peak : item.replaygain_track_peak)
    ?? positive(item.replaygain_track_peak)
    ?? positive(item.replaygain_album_peak)
  let factor = 10 ** (Math.max(-60, Math.min(12, gain)) / 20)
  if (peak !== null) factor = Math.min(factor, 1 / peak)
  return Math.max(0.001, Math.min(4, factor))
}

export function createReplayGain({ audio, audios = audio ? [audio] : [], select, storage = window.localStorage, AudioContextClass = window.AudioContext || window.webkitAudioContext }) {
  let mode = modes.has(storage.getItem('mochil-replaygain')) ? storage.getItem('mochil-replaygain') : 'track'
  const graphs = new Map()
  let context
  let unavailable = false
  select.value = mode

  function ensureGraph(target) {
    if (graphs.has(target) || unavailable || !AudioContextClass) return graphs.get(target)
    try {
	  context ||= new AudioContextClass()
      const source = context.createMediaElementSource(target)
      const gainNode = context.createGain()
      source.connect(gainNode).connect(context.destination)
	  const graph = { gainNode }
      graphs.set(target, graph)
      return graph
    } catch {
      unavailable = true
    }
  }

  function apply(item, target = audios[0]) {
    if (!target) return 1
    const factor = replayGainFactor(item, mode)
    const graph = ensureGraph(target)
    if (graph) {
      target.volume = 1
	  graph.gainNode.gain.setValueAtTime(factor, context.currentTime)
    } else {
      target.volume = Math.min(1, factor)
    }
    return factor
  }

  function setMode(value, item, target = audios[0]) {
    mode = modes.has(value) ? value : 'track'
    storage.setItem('mochil-replaygain', mode)
    select.value = mode
    return apply(item, target)
  }

  async function resume() {
	if (context?.state === 'suspended') {
	  try { await context.resume() } catch {}
	}
  }

  return { apply, resume, setMode, mode: () => mode }
}

function finite(value) {
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function positive(value) {
  const number = finite(value)
  return number !== null && number > 0 ? number : null
}
