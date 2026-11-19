import test from 'node:test'
import assert from 'node:assert/strict'
import { createNarrationPlayer } from '../static/narration-player.js'

function fakeAudio() {
  const target = new EventTarget()
  Object.assign(target, {
    src: '', playbackRate: 1, played: [], paused: true,
    async play() { this.paused = false; if (this.src.startsWith('speech:')) this.played.push(this.src) },
    pause() { this.paused = true },
    end() { this.dispatchEvent(new Event('ended')) },
  })
  return target
}
const tick = () => new Promise(resolve => setTimeout(resolve, 5))
async function until(check, label) {
  for (let i = 0; i < 200; i++) { if (check()) return; await tick() }
  assert.fail('timed out waiting for ' + label)
}

function setup(texts, extra = {}) {
  const audio = fakeAudio()
  const log = { fetched: [], marked: [], visible: [], states: [], errors: [] }
  let sentences = texts.map(text => ({ text }))
  const player = createNarrationPlayer({
    audio,
    loadSentences: async () => sentences,
    fetchAudio: async (text, voice) => { log.fetched.push(`${voice}|${text}`); return `speech:${text}` },
    mark: sentence => log.marked.push(sentence.text),
    unmark: () => log.marked.push('-'),
    ensureVisible: async sentence => { log.visible.push(sentence.text) },
    firstOnPage: () => 0,
    nextChapter: async () => false,
    onState: state => log.states.push(state),
    onError: error => log.errors.push(error.message),
    ...extra,
  })
  return { audio, log, player, set sentences(value) { sentences = value } }
}

test('reads sentences in order, marking each one and fetching a few ahead', async () => {
  const { audio, log, player } = setup(['One.', 'Two.', 'Three.', 'Four.', 'Five.'])
  player.toggle()
  await until(() => audio.played.length === 1, 'first sentence')
  assert.deepEqual(log.marked, ['One.'])
  assert.deepEqual(log.fetched.slice(0, 4), ['|One.', '|Two.', '|Three.', '|Four.'], 'three ahead of the current one')
  audio.end()
  await until(() => audio.played.length === 2, 'second sentence')
  assert.deepEqual(audio.played, ['speech:One.', 'speech:Two.'])
  assert.deepEqual(log.visible.slice(0, 2), ['One.', 'Two.'])
})

test('pausing holds the place and resuming carries on', async () => {
  const { audio, log, player } = setup(['One.', 'Two.'])
  player.toggle()
  await until(() => player.state === 'playing', 'playing')
  player.pause()
  assert.equal(player.state, 'paused')
  assert.equal(audio.paused, true)
  player.resume()
  await until(() => player.state === 'playing', 'playing again')
  audio.end()
  await until(() => audio.played.at(-1) === 'speech:Two.', 'second sentence')
  assert.equal(log.errors.length, 0)
})

test('running out of text with no next chapter stops cleanly', async () => {
  const { audio, log, player } = setup(['Only one.'])
  player.toggle()
  await until(() => audio.played.length === 1, 'sentence')
  audio.end()
  await until(() => player.state === 'idle', 'idle')
  assert.equal(log.marked.at(-1), '-', 'the highlight is removed')
})

test('carries on into the next chapter', async () => {
  let hasNext = true
  const ctx = setup(['Chapter one ends.'], { nextChapter: async () => { const had = hasNext; hasNext = false; return had } })
  ctx.player.toggle()
  await until(() => ctx.audio.played.length === 1, 'first')
  ctx.sentences = ['Chapter two starts.', 'It goes on.'].map(text => ({ text }))
  ctx.audio.end()
  await until(() => ctx.audio.played.length === 2, 'second chapter')
  assert.equal(ctx.audio.played[1], 'speech:Chapter two starts.')
})

test('skipping moves by sentences', async () => {
  const { audio, player } = setup(['A one.', 'B two.', 'C three.', 'D four.'])
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  player.skip(2)
  await until(() => audio.played.length === 2, 'after skip')
  assert.equal(audio.played[1], 'speech:C three.')
  player.skip(-1)
  await until(() => audio.played.length === 3, 'after back')
  assert.equal(audio.played[2], 'speech:B two.')
})

test('stopping while a sentence is still being prepared plays nothing', async () => {
  let release
  const slow = new Promise(resolve => { release = resolve })
  const { audio, player } = setup(['One.'], { fetchAudio: async () => { await slow; return 'speech:One.' } })
  player.toggle()
  await until(() => player.state === 'loading', 'loading')
  player.toggle()
  assert.equal(player.state, 'idle')
  release()
  await tick(); await tick()
  assert.deepEqual(audio.played, [])
})

test('a failed fetch stops narration and says why', async () => {
  const { player, log } = setup(['One.'], { fetchAudio: async () => { throw new Error('Narration is not set up on this server.') } })
  player.toggle()
  await until(() => log.errors.length === 1, 'error')
  assert.equal(log.errors[0], 'Narration is not set up on this server.')
  assert.equal(player.state, 'idle')
})

test('changing the voice re-reads the current sentence in the new voice', async () => {
  const { audio, log, player } = setup(['One.', 'Two.'])
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  player.setVoice('cosette')
  await until(() => audio.played.length === 2, 'again')
  assert.ok(log.fetched.includes('cosette|One.'))
})

test('speed is kept and applied; unknown speeds are refused', () => {
  const store = new Map()
  const storage = { getItem: key => store.get(key) ?? null, setItem: (key, value) => store.set(key, value) }
  const { audio, player } = setup(['One.'], { storage })
  player.setRate(1.5)
  assert.equal(audio.playbackRate, 1.5)
  assert.equal(store.get('mochil-narration-rate'), '1.5')
  player.setRate(9)
  assert.equal(player.rate, 1.5)
})

test('nothing to read reports it instead of hanging', async () => {
  const { player, log } = setup([])
  player.toggle()
  await until(() => log.errors.length === 1, 'error')
  assert.match(log.errors[0], /no text/i)
  assert.equal(player.state, 'idle')
})

test('the sleep timer pauses narration where it stopped, and resuming carries on', async () => {
  const { audio, player } = setup(['One.', 'Two.', 'Three.'])
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  player.setSleep(0.001) // about 60 ms
  await until(() => player.state === 'paused', 'paused by the timer')
  assert.equal(audio.paused, true)
  assert.equal(audio.volume ?? 1, 1, 'volume is restored for the next time')
  player.resume()
  await until(() => player.state === 'playing', 'playing again')
  audio.end()
  await until(() => audio.played.at(-1) === 'speech:Two.', 'continues with the next sentence')
})

test('"end of chapter" stops after the chapter, positioned at the start of the next one', async () => {
  let moved = 0
  const { audio, player } = setup(['Last sentence.'], { nextChapter: async () => { moved++; return true } })
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  player.setSleep('chapter')
  assert.equal(player.sleepAtChapterEnd, true)
  audio.end()
  await until(() => player.state === 'idle', 'stopped')
  assert.equal(moved, 1, 'the next chapter is opened so resuming starts there')
  assert.equal(audio.played.length, 1, 'nothing more is spoken')
  assert.equal(player.sleepAtChapterEnd, false)
})

test('stopping cancels a running sleep timer', async () => {
  const { audio, player } = setup(['One.', 'Two.'])
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  player.setSleep(0.001)
  player.stop()
  await new Promise(resolve => setTimeout(resolve, 700))
  assert.equal(player.state, 'idle')
  assert.equal(audio.volume ?? 1, 1)
})

test('the first sentence is spoken clause by clause so audio starts sooner', async () => {
  const whole = { text: 'When the old captain came ashore, he counted every rope.', lead: { text: 'When the old captain came ashore,', endNode: {}, endOffset: 33 } }
  const { audio, log, player } = setup([], { loadSentences: async () => [whole, { text: 'Next.' }] })
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  assert.deepEqual(log.marked, ['When the old captain came ashore,'])
  audio.end()
  await until(() => audio.played.length === 2, 'second')
  assert.equal(log.marked.at(-1), 'he counted every rope.')
})

test('the next sentence starts inside the "ended" event, with nothing awaited in between', async () => {
  // A locked phone stops the page once nothing is playing and holds back page work,
  // so the hand-over to the next sentence must not wait for either.
  let release
  const slowScroll = new Promise(resolve => { release = resolve })
  const { audio, log, player } = setup(['One.', 'Two.', 'Three.'], { ensureVisible: async sentence => { log.visible.push(sentence.text); if (sentence.text !== 'One.') await slowScroll } })
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  await until(() => log.fetched.length === 3, 'next sentences prepared') // prefetch must be done
  await tick()
  audio.end()
  assert.equal(audio.played.at(-1), 'speech:Two.', 'started synchronously, before any other task ran')
  await until(() => player.state === 'playing', 'still playing')
  audio.end() // the page work for sentence two never finished, and narration goes on anyway
  assert.equal(audio.played.at(-1), 'speech:Three.')
  release()
})

test('a sentence that is not ready when the last one ends is waited for', async () => {
  let ready
  const late = new Promise(resolve => { ready = resolve })
  const { audio, player } = setup(['One.', 'Two.'], { fetchAudio: async text => (text === 'Two.' ? late : `speech:${text}`) })
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  audio.end()
  assert.ok(audio.src.startsWith('data:audio/wav'), 'silence plays while waiting, so a locked phone keeps the page alive')
  assert.equal(audio.loop, true)
  await tick()
  assert.equal(audio.played.length, 1)
  ready('speech:Two.')
  await until(() => audio.played.at(-1) === 'speech:Two.', 'plays once it arrives')
  assert.equal(audio.loop, false)
})

test('silence also plays while the next chapter is opened', async () => {
  let open
  const gate = new Promise(resolve => { open = resolve })
  const { audio, player } = setup(['Last.'], { nextChapter: async () => { await gate; return false } })
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  audio.end()
  assert.equal(audio.loop, true)
  assert.ok(audio.src.startsWith('data:audio/wav'))
  open()
  await until(() => player.state === 'idle', 'finished')
  assert.equal(audio.loop, false)
})

test('stopping ends the silence', async () => {
  const { audio, player } = setup(['One.', 'Two.'], { fetchAudio: async text => (text === 'Two.' ? new Promise(() => {}) : `speech:${text}`) })
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  audio.end()
  assert.equal(audio.loop, true)
  player.stop()
  assert.equal(audio.loop, false)
  assert.equal(audio.paused, true)
})

test('after skipping, the end of a sentence moves on by exactly one sentence and the mark follows the audio', async () => {
  const { audio, log, player } = setup(['One.', 'Two.', 'Three.', 'Four.', 'Five.', 'Six.', 'Seven.'])
  player.toggle()
  await until(() => audio.played.length === 1, 'first sentence')
  await tick()
  player.skip(1)
  await until(() => audio.played.length === 2, 'skipped to Two')
  player.skip(1)
  await until(() => audio.played.length === 3, 'skipped to Three')
  player.skip(1)
  await until(() => audio.played.length === 4, 'skipped to Four')
  audio.end() // one ending: Four is over, so Five must be the one that plays, once
  await until(() => audio.played.length === 5, 'Five after Four')
  await tick(); await tick()
  assert.deepEqual(audio.played, ['speech:One.', 'speech:Two.', 'speech:Three.', 'speech:Four.', 'speech:Five.'], 'no sentence is played twice or out of turn')
  assert.equal(log.marked.at(-1), 'Five.', 'the highlight is on the sentence that is playing')
  player.stop()
})

test('skipping while the next sentence is still being fetched is not undone when that audio arrives', async () => {
  let release
  const gate = new Promise(resolve => { release = resolve })
  const { audio, log, player } = setup(['One.', 'Two.', 'Three.', 'Four.'], {
    fetchAudio: async text => { if (text === 'Two.') await gate; return `speech:${text}` },
  })
  player.toggle()
  await until(() => audio.played.length === 1, 'first sentence')
  audio.end() // Two is not ready, so narration waits for it
  await until(() => log.states.at(-1) === 'loading', 'waiting for Two')
  player.skip(2) // the reader asks for Four
  release()
  await until(() => audio.played.includes('speech:Four.'), 'Four')
  assert.equal(log.marked.at(-1), 'Four.')
  assert.ok(!audio.played.includes('speech:Two.'), 'the abandoned sentence is not read')
  player.stop()
})

test('seeking jumps to the sentence that was picked, and does nothing when nothing is chosen or narration is off', async () => {
  const { audio, player } = setup(['A one.', 'B two.', 'C three.', 'D four.'])
  assert.equal(player.seek(() => 2), false, 'nothing to jump in while idle')
  player.toggle()
  await until(() => audio.played.length === 1, 'first')
  assert.equal(player.seek(() => -1), false)
  assert.equal(player.seek(sentences => sentences.findIndex(sentence => sentence.text.startsWith('D'))), true)
  await until(() => audio.played.length === 2, 'after seek')
  assert.equal(audio.played[1], 'speech:D four.')
})
