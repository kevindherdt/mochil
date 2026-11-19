// Narration for the Readium reader: finds out whether the server can speak,
// adds the play button, and connects the narration player to the chapter that
// is on screen (which sentence is showing, marking it, turning pages, moving
// on to the next chapter).
import { chapterSentences } from './narration-text.js?v=2'
import { readerLanguageChoices, readerVoiceOptionsHTML } from './narration-languages.js?v=1'
import { createNarrationPlayer, narrationRates } from './narration-player.js?v=7'
import { isMatterTitle } from './narration-clean.js?v=1'
import { placeOf, readPlace, rememberedIndex, savePlace, startIndex } from './narration-resume.js?v=2'

const MARK = 'bobo-narration'

export async function mountReadiumNarration({ $, navigator, order, step, Locator, title, publication, storage = localStorage }) {
  let status
  try {
    const response = await fetch('/api/narration/status', { credentials: 'same-origin' })
    if (!response.ok) return null // the module is off
    status = await response.json()
  } catch { return null }
  if (!status?.available) return null

  const button = $('#narrate'), section = $('#narration-settings'), dialog = $('#reader-settings')
  const frame = () => navigator._cframes?.[0]?.window
  const paint = () => { button.dataset.state = player.state; button.setAttribute('aria-pressed', String(player.state === 'playing')) }
  const notify = message => {
    const line = $('#location'), before = line.textContent
    line.textContent = message
    setTimeout(() => { if (line.textContent === message) line.textContent = before }, 6000)
  }

  const rangeOf = sentence => {
    const doc = frame()?.document
    if (!doc || sentence.startNode.ownerDocument !== doc) return null
    const range = doc.createRange()
    range.setStart(sentence.startNode, sentence.startOffset)
    range.setEnd(sentence.endNode, sentence.endOffset)
    return range
  }
  const inView = range => {
    const win = frame()
    const box = range.getClientRects()[0] || range.getBoundingClientRect()
    // Pages sit side by side (so left says where a sentence is), or one long page scrolls (so top does).
    const wide = (win?.innerWidth || 0) - 8, tall = (win?.innerHeight || 0) - 8
    return { box, visible: box.left >= -1 && box.left < wide && box.top >= -20 && box.top < tall, after: box.left >= wide || box.top >= tall }
  }

  // Chapter titles from the table of contents, to tell the story from the pages around it.
  const titles = new Map()
  const collect = links => { for (const link of links || []) { titles.set(String(link.href).split('#')[0], titles.get(String(link.href).split('#')[0]) ?? link.title); collect(link.children?.items ?? link.children) } }
  collect(publication?.manifest?.toc?.items ?? publication?.manifest?.toc)
  let preparing = null, wasActive = false, visibleRun = 0
  // Where narration stopped is kept for this book, so it carries on from there and shows it when the book opens.
  const placeName = `bobo.narration.place.${location.pathname}`
  const hrefNow = () => navigator.currentLocator?.href ?? ''
  let chapter = []
  // The sentence a person has selected in the page, if any: narration starts there.
  const selectedSentence = sentences => {
    const selection = frame()?.getSelection?.()
    if (!selection || selection.isCollapsed || !selection.rangeCount) return -1
    const { startContainer, startOffset } = selection.getRangeAt(0)
    const at = sentences.findIndex(sentence => { const range = rangeOf(sentence); return range && range.comparePoint(startContainer, startOffset) <= 0 })
    if (at >= 0) selection.removeAllRanges() // the marked sentence takes its place
    return at
  }
  const audio = new Audio()
  audio.preload = 'auto'
  const mark = sentence => {
    const win = frame(), range = rangeOf(sentence)
    if (!win?.CSS?.highlights || !range) return
    savePlace(storage, placeName, placeOf(hrefNow(), sentence, Math.max(0, chapter.indexOf(sentence))))
    const doc = win.document
    if (!doc.getElementById('bobo-narration-style')) {
      const style = doc.createElement('style')
      style.id = 'bobo-narration-style'
      style.textContent = `::highlight(${MARK}){background-color:rgba(240,154,114,.45)}`
      doc.head.append(style)
    }
    win.CSS.highlights.set(MARK, new win.Highlight(range))
  }

  const player = createNarrationPlayer({
    audio, storage, mediaSession: globalThis.navigator?.mediaSession,
    loadSentences: async () => (chapter = chapterSentences(frame()?.document)),
    firstOnPage: sentences => {
      const visibleAt = index => { const range = rangeOf(sentences[index]); return Boolean(range) && inView(range).visible }
      const first = sentences.findIndex((_, index) => visibleAt(index))
      return startIndex({ selected: selectedSentence(sentences), remembered: rememberedIndex(sentences, readPlace(storage, placeName), hrefNow()), visibleAt, first: first < 0 ? 0 : first })
    },
    fetchAudio: async (text, voice, signal) => {
      const response = await fetch('/api/narration/speak', {
        method: 'POST', credentials: 'same-origin', signal,
        headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ text, voice }),
      })
      if (!response.ok) throw new Error((await response.text().catch(() => '')).trim() || `Narration returned ${response.status}.`)
      return URL.createObjectURL(await response.blob())
    },
    mark,
    unmark: () => {}, // the sentence narration stopped at stays marked, so it is clear where it will carry on
    ensureVisible: async sentence => {
      // Only the newest request turns pages: skipping quickly would otherwise leave several loops each paging on.
      const mine = ++visibleRun
      for (let tries = 0; tries < 8 && mine === visibleRun; tries++) {
        const win = frame(), range = rangeOf(sentence)
        if (!win || !range) return
        const { visible, after } = inView(range)
        if (visible) return
        await step(after ? 1 : -1)
        await new Promise(done => setTimeout(done, 200))
      }
    },
    nextChapter: async () => {
      // Copyright pages, "also by" lists and the like are skipped; if only those are left, the book is done.
      const here = navigator.currentLocator?.href?.split('#')[0]
      let at = order.findIndex(link => link.href === here) + 1
      while (order[at] && isMatterTitle(titles.get(order[at].href))) at++
      const next = order[at]
      const before = frame()?.document
      if (!here || !next) return false
      await new Promise(resolve => navigator.go(Locator.deserialize({ href: next.href, type: next.type || 'application/xhtml+xml', locations: { progression: 0 } }), false, resolve))
      for (let waited = 0; waited < 6000; waited += 50) {
        const doc = frame()?.document
        if (doc && doc !== before && doc.body?.childElementCount) return true
        await new Promise(done => setTimeout(done, 50))
      }
      return false
    },
    onState: state => {
      paint()
      // Starting takes a few seconds while the voice wakes up; say so instead of seeming stuck.
      const line = $('#location')
      if (state === 'loading' && preparing === null && !wasActive) { preparing = line.textContent; line.textContent = 'Preparing narration…' }
      if (state !== 'loading' && preparing !== null) { if (line.textContent === 'Preparing narration…') line.textContent = preparing; preparing = null }
      wasActive = state !== 'idle'
      if (state === 'playing') { try { storage.setItem(usedKey, '1') } catch {} }
      if (player.state === 'playing' && globalThis.MediaMetadata && globalThis.navigator?.mediaSession) {
        globalThis.navigator.mediaSession.metadata = new MediaMetadata({ title: title(), artist: 'Mochil narration' })
      }
    },
    onError: error => { paint(); notify(error?.message || 'Narration stopped.') },
  })

  // Loading the voice takes seconds, so it starts before play is pressed: when the book
  // opens (for readers who have used narration), and when the menu or the voice changes.
  const warm = () => { fetch('/api/narration/warm', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ voice: player.voice }) }).catch(() => {}) }
  const usedKey = 'bobo.narration.used'
  let used = false
  try { used = storage.getItem(usedKey) === '1' } catch {}
  if (used && !status.loaded) warm()
  // When the book opens, mark the sentence narration stopped at (the chapter takes a moment to appear).
  const showPlace = async () => {
    const place = readPlace(storage, placeName)
    if (!place) return
    for (let waited = 0; waited < 10000; waited += 300) {
      const doc = frame()?.document
      if (doc?.body?.childElementCount && hrefNow().split('#')[0] === place.href) {
        const sentences = chapterSentences(doc), at = rememberedIndex(sentences, place, hrefNow())
        if (at >= 0 && player.state === 'idle') { chapter = sentences; mark(sentences[at]) }
        return
      }
      await new Promise(done => setTimeout(done, 300))
    }
  }
  button.hidden = false
  const tab = $('#rs-tab-narration')
  if (tab) tab.hidden = false
  // Narration becomes available asynchronously, well after the settings dialog
  // may already be open on another tab. Only reveal this panel if its tab is
  // the one currently selected; otherwise stay hidden like any other inactive
  // panel, so at most one panel is ever visible at a time.
  section.hidden = !(tab && tab.getAttribute('aria-selected') === 'true')
  showPlace().catch(() => {})
  button.onclick = () => player.toggle()
  const voiceSelect = $('#narration-voice'), speedSelect = $('#narration-speed')
  // A language chooser narrows the list when the voices speak more than one; voices you made come first.
  const allVoices = status.voices || []
  const languageSelect = $('#narration-language'), languages = readerLanguageChoices(allVoices)
  const showVoices = (language, current = player.voice) => { voiceSelect.innerHTML = readerVoiceOptionsHTML(allVoices, language, current) }
  if (languageSelect) {
    languageSelect.innerHTML = languages.map(choice => `<option value="${choice.id}">${choice.id === 'all' ? 'All languages' : choice.label} (${choice.count})</option>`).join('')
    languageSelect.closest('label').hidden = !languages.length
    languageSelect.onchange = () => {
      showVoices(languageSelect.value)
      // Keep showing the voice in use if it speaks this language; otherwise the list asks for a choice.
      if ([...voiceSelect.options].some(option => option.value === player.voice)) voiceSelect.value = player.voice
    }
  }
  const known = id => allVoices.some(voice => voice.id === id)
  const inUse = known(player.voice) ? player.voice : status.voice
  showVoices('all', inUse)
  voiceSelect.value = inUse
  if (player.voice !== voiceSelect.value) player.setVoice(voiceSelect.value)
  speedSelect.innerHTML = narrationRates.map(rate => `<option value="${rate}">${rate}×</option>`).join('')
  speedSelect.value = String(player.rate)
  voiceSelect.onchange = () => { player.setVoice(voiceSelect.value); warm() }
  speedSelect.onchange = () => player.setRate(Number(speedSelect.value))
  const sleepSelect = $('#narration-sleep'), sleepLeft = $('#narration-sleep-left')
  const showSleep = () => {
    const ms = player.sleepRemainingMs()
    sleepLeft.textContent = ms ? `Pauses in ${Math.ceil(ms / 60000)} min` : player.sleepAtChapterEnd ? 'Pauses at the end of this chapter' : ''
    if (!ms && !player.sleepAtChapterEnd) sleepSelect.value = '0'
  }
  sleepSelect.onchange = () => { player.setSleep(sleepSelect.value === 'chapter' ? 'chapter' : Number(sleepSelect.value)); showSleep() }
  const sleepPoll = setInterval(showSleep, 15000)
  globalThis.window?.addEventListener('pagehide', () => clearInterval(sleepPoll))
  $('#settings-open').addEventListener('click', () => { showSleep(); warm() }) // the reader opens the dialog; this prepares the voice
  paint()
  // Double-click a word, or triple-click a paragraph, to carry on reading from there: it starts narration, or jumps to it.
  const carryOn = () => { if (player.state === 'idle') player.toggle(); else player.seek(selectedSentence) }
  const frameLoaded = win => {
    if (win.document.boboCarryOn) return // already listening in this chapter
    win.document.boboCarryOn = true
    win.document.addEventListener('dblclick', carryOn)
    win.document.addEventListener('click', event => { if (event.detail === 3) carryOn() })
  }
  return { player, dialog, frameLoaded }
}
