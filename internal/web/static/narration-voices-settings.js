// Settings → Modules → Narration voices: the voices the reader can read with.
// A voice is made from a short recording of someone speaking, either uploaded
// or recorded here. Everything stays on this server.
import { languageName, languageChoices, filterVoices } from './narration-languages.js?v=1'
export { languageChoices, filterVoices }
const escapeHTML = value => String(value).replace(/[&<>"']/g, ch => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[ch])
const SAMPLE = { english: 'Hello! This is how I sound when I read your books aloud.', dutch: 'Hallo! Zo klink ik wanneer ik je boeken voorlees.' }
const MAX_RECORDING_SECONDS = 30

export const describeVoice = voice =>
  [languageName(voice.language || 'english'), voice.seconds ? `${Math.round(voice.seconds)} s` : ''].filter(Boolean).join(' · ')

export function voiceCardsHTML(voices, manage = true) {
  return voices.map(voice => `<div class="nv-card" data-voice="${escapeHTML(voice.id)}" data-language="${escapeHTML(voice.language || 'english')}" title="${escapeHTML(voice.name)}"><div><strong>${escapeHTML(voice.name)}</strong><small>${escapeHTML(describeVoice(voice))}</small></div><button type="button" data-action="try" aria-label="Hear ${escapeHTML(voice.name)}">Hear</button>${voice.custom ? `<details class="nv-more"><summary aria-label="More for ${escapeHTML(voice.name)}">⋯</summary><div class="nv-actions"><button type="button" data-action="listen">Original recording</button>${manage ? '<button type="button" data-action="rename">Rename</button><button type="button" data-action="delete">Delete</button>' : ''}</div></details>` : ''}</div>`).join('')
}

export function chipsHTML(choices, selected) {
  return choices.map(choice => `<button type="button" class="nv-chip" data-language="${escapeHTML(choice.id)}" aria-pressed="${choice.id === selected}">${escapeHTML(choice.label)} <span>${choice.count}</span></button>`).join('')
}

export function defaultVoiceOptionsHTML(voices, selected) {
  const mine = voices.filter(voice => voice.custom)
  const group = (label, list) => list.length ? `<optgroup label="${label}">${list.map(voice => `<option value="${escapeHTML(voice.id)}"${voice.id === selected ? ' selected' : ''}>${escapeHTML(voice.name)}</option>`).join('')}</optgroup>` : ''
  return group('My voices', mine) + group('Built-in voices', voices.filter(voice => !voice.custom))
}

const formHTML = `<details class="nv-new"><summary>Make a new voice</summary>
  <form id="nv-form" class="nv-form">
    <label>Name<input id="nv-name" maxlength="80" autocomplete="off" required></label>
    <label>Language<select id="nv-language"><option value="english">English</option><option value="dutch">Dutch</option></select></label>
    <div class="nv-source">
      <label class="nv-file">Upload a recording<input id="nv-file" type="file" accept="audio/*,video/webm"></label>
      <button type="button" id="nv-record">Record</button><span id="nv-timer" aria-live="off"></span>
    </div>
    <audio id="nv-preview" controls hidden></audio>
    <p class="nv-hint">Use 3 to 30 seconds of clear, continuous speech; 10 to 20 is a good start. Read it the way you want the voice to sound.</p>
    <button type="submit" class="primary" id="nv-create">Create voice</button>
  </form></details>`

export function createNarrationVoices({ document, $, fetchImpl = fetch, mediaDevices = globalThis.navigator?.mediaDevices }) {
  let root
  let hermesOn = false
  let recorded = null
  let recorder = null
  let timer = null
  const player = new Audio()

  function mount() {
    if (root) return root
    // Narration has its own Settings tab; the voices sit right under its heading.
    const list = document.querySelector('[data-settings-panel="narration"] > header')
    if (!list) return null
    root = document.createElement('details')
    root.className = 'settings-accordion nv'
    root.id = 'narration-voices'
    root.hidden = true
    root.open = true
    list.after(root)
    root.addEventListener('click', onClick)
    root.addEventListener('submit', onSubmit)
    root.addEventListener('change', event => {
      if (event.target.id !== 'nv-default') return
      root.querySelector('#nv-default-save').disabled = false
      root.querySelector('#nv-default-status').textContent = 'Not saved yet.'
    })
    root.addEventListener('input', event => { if (event.target.id === 'nv-filter') { view.query = event.target.value; paint() } })
    return root
  }

  let voices = []
  let manage = true // only administrators change the shared voices; the server says so in the status
  const view = { language: 'all', query: '' }

  /** Redraws only the buttons and the list, so typing in the search box or the new-voice form is never interrupted. */
  function paint() {
    if (!root.querySelector('.nv-grid')) return
    const choices = languageChoices(voices)
    if (!choices.some(choice => choice.id === view.language)) view.language = 'all'
    root.querySelector('.nv-chips').innerHTML = chipsHTML(choices, view.language)
    const shown = filterVoices(voices, view)
    root.querySelector('.nv-grid').innerHTML = voiceCardsHTML(shown, manage)
    root.querySelector('.nv-none').hidden = shown.length > 0
  }

  const say = message => { const line = root?.querySelector('#nv-status'); if (line) line.textContent = message }
  const busy = on => root?.querySelectorAll('button').forEach(button => { button.disabled = on })

  async function readError(response) {
    return (await response.text().catch(() => '')).trim() || `The server answered ${response.status}.`
  }

  async function refresh() {
    const response = await fetchImpl('/api/narration/status', { credentials: 'same-origin' }).catch(() => null)
    if (!response?.ok) { root.hidden = true; return }
    const status = await response.json()
    root.hidden = false
    if (!status.available) {
      root.innerHTML = '<summary><span><strong>Narration voices</strong><small>Not available</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary><div class="settings-accordion-body"><p class="nv-empty">The voice runtime is not installed on this server yet, so there is nothing to read with. See docs/NARRATION.md.</p></div>'
      return
    }
    voices = status.voices || []
    manage = status.manage !== false
    const languages = languageChoices(voices).filter(choice => choice.id !== 'all' && choice.id !== 'mine')
    const summary = `${voices.length} ${voices.length === 1 ? 'voice' : 'voices'}${languages.length > 1 ? ' · ' + languages.map(choice => `${choice.label} ${choice.count}`).join(', ') : ''}`
    root.innerHTML = `<summary><span><strong>Narration voices</strong><small>${escapeHTML(summary)}</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary><div class="settings-accordion-body">
      ${manage ? `<div class="nv-default" style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;margin:4px 0 0"><label style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;font-weight:600">Default voice<select id="nv-default" aria-label="Default voice">${defaultVoiceOptionsHTML(voices, status.voice)}</select></label><button type="button" id="nv-default-save" class="primary" disabled>Save</button><span id="nv-default-status" role="status" style="color:var(--muted);font-size:var(--fs-13)"></span></div><p class="nv-note">${hermesOn ? 'Used by the Hermes chat and as the starting voice in the reader.' : 'Used as the starting voice in the reader.'}</p>` : ''}
      <div class="nv-toolbar"><div class="nv-chips" role="group" aria-label="Filter by language"></div><input id="nv-filter" type="search" placeholder="Find a voice" aria-label="Find a voice" autocomplete="off" value="${escapeHTML(view.query)}"></div>
      <div class="nv-grid"></div><p class="nv-none nv-empty" hidden>No voice matches. Clear the search or pick another language.</p>
      <p class="nv-note">These also appear in the reader's Read aloud menu. ${manage ? 'Voices you make from a recording can be renamed or deleted.' : 'Voices are managed by an administrator.'}</p>
      ${manage ? formHTML : ''}<p id="nv-status" role="status"></p></div>`
    paint()
    root.querySelector('#nv-file')?.addEventListener('change', event => {
      recorded = null
      const file = event.target.files[0]
      const preview = root.querySelector('#nv-preview')
      preview.hidden = !file
      if (file) preview.src = URL.createObjectURL(file)
    })
  }

  async function play(url, options) {
    try {
      player.pause()
      const response = await fetchImpl(url, { credentials: 'same-origin', ...options })
      if (!response.ok) throw new Error(await readError(response))
      player.src = URL.createObjectURL(await response.blob())
      await player.play()
      say('')
    } catch (error) { say(error.message) }
  }

  async function saveDefaultVoice(button) {
    const select = root.querySelector('#nv-default'), line = root.querySelector('#nv-default-status')
    button.disabled = true
    line.textContent = 'Saving…'
    const response = await fetchImpl('/api/narration/default-voice', { method: 'PUT', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ voice: select.value }) }).catch(() => null)
    line.textContent = response?.ok ? 'Saved.' : 'Could not save the default voice.'
    button.disabled = Boolean(response?.ok)
  }

  async function onClick(event) {
    const button = event.target.closest('button')
    if (!button) return
    if (button.id === 'nv-default-save') return saveDefaultVoice(button)
    if (button.id === 'nv-record') return toggleRecording(button)
    if (button.classList.contains('nv-chip')) { view.language = button.dataset.language; return paint() }
    const row = button.closest('.nv-card')
    if (!row) return
    row.querySelector('.nv-more')?.removeAttribute('open')
    const id = row.dataset.voice
    const name = row.querySelector('strong').textContent
    const language = row.dataset.language
    if (button.dataset.action === 'listen') { say('Playing the original recording…'); await play(`/api/narration/voices/${id}/sample`) }
    if (button.dataset.action === 'try') {
      say('Preparing a sample… the first one can take a few seconds.')
      busy(true)
      await play('/api/narration/speak', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ text: SAMPLE[language] || SAMPLE.english, voice: id }) })
      busy(false)
    }
    if (button.dataset.action === 'rename') {
      const next = globalThis.prompt?.('Name for this voice', name)
      if (!next || next.trim() === name) return
      const response = await fetchImpl(`/api/narration/voices/${id}`, { method: 'PATCH', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: next }) })
      if (!response.ok) return say(await readError(response))
      await refresh()
    }
    if (button.dataset.action === 'delete') {
      if (!globalThis.confirm?.(`Delete the voice “${name}”? Books it was reading will use another voice.`)) return
      const response = await fetchImpl(`/api/narration/voices/${id}`, { method: 'DELETE', credentials: 'same-origin' })
      if (!response.ok) return say(await readError(response))
      await refresh()
    }
  }

  async function toggleRecording(button) {
    if (recorder) { recorder.stop(); return }
    if (!mediaDevices?.getUserMedia || !globalThis.MediaRecorder) return say('This browser cannot record audio. Upload a recording instead.')
    let stream
    try { stream = await mediaDevices.getUserMedia({ audio: true }) } catch { return say('Microphone access was not allowed.') }
    const chunks = []
    recorder = new MediaRecorder(stream)
    recorder.ondataavailable = event => chunks.push(event.data)
    recorder.onstop = () => {
      stream.getTracks().forEach(track => track.stop())
      clearInterval(timer)
      recorded = new Blob(chunks, { type: recorder.mimeType || 'audio/webm' })
      recorder = null
      button.textContent = 'Record again'
      root.querySelector('#nv-file').value = ''
      const preview = root.querySelector('#nv-preview')
      preview.src = URL.createObjectURL(recorded)
      preview.hidden = false
    }
    recorder.start()
    const began = Date.now()
    button.textContent = 'Stop'
    timer = setInterval(() => {
      const seconds = Math.floor((Date.now() - began) / 1000)
      root.querySelector('#nv-timer').textContent = `${seconds} s`
      if (seconds >= MAX_RECORDING_SECONDS) recorder?.stop()
    }, 250)
  }

  async function onSubmit(event) {
    event.preventDefault()
    const file = root.querySelector('#nv-file').files[0] || recorded
    if (!file) return say('Upload or record a sample first.')
    const form = new FormData()
    form.set('name', root.querySelector('#nv-name').value)
    form.set('language', root.querySelector('#nv-language').value)
    form.set('file', file, file.name || 'recording.webm')
    busy(true)
    say('Making the voice… this takes up to half a minute the first time.')
    try {
      const response = await fetchImpl('/api/narration/voices', { method: 'POST', credentials: 'same-origin', body: form })
      if (!response.ok) throw new Error(await readError(response))
      recorded = null
      await refresh()
      say('Voice saved. Use “Try voice” to hear it.')
    } catch (error) { busy(false); say(error.message) }
  }

  return {
    /** Shows the panel when the Narration module is running on the server. */
    async load(modules) {
      const narration = (modules || []).find(module => module.id === 'narration')
      hermesOn = Boolean((modules || []).find(module => module.id === 'hermes')?.enabled)
      if (!mount()) return
      if (!narration?.enabled) { root.hidden = true; return }
      await refresh()
    },
  }
}
