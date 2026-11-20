import test from 'node:test'
import assert from 'node:assert/strict'
import { readerLanguageChoices, readerVoiceOptionsHTML, languageName } from '../static/narration-languages.js'

const voices = [
  { id: 'alba', name: 'Alba', language: 'english' },
  { id: 'cosette', name: 'Cosette', language: 'english' },
  { id: 'daan', name: 'Daan', language: 'dutch' },
  { id: 'mine', name: 'Kevin <b>', language: 'english', custom: true },
]

test('the reader offers a language chooser only when voices speak more than one language', () => {
  assert.deepEqual(readerLanguageChoices(voices).map(choice => choice.id), ['all', 'dutch', 'english'])
  assert.deepEqual(readerLanguageChoices(voices.filter(voice => voice.language === 'english')), [], 'one language: nothing to choose')
  assert.deepEqual(readerLanguageChoices([]), [])
})

test('the chooser never lists "My voices": the reader groups those already', () => {
  assert.ok(!readerLanguageChoices(voices).some(choice => choice.id === 'mine'))
})

test('all languages: your voices first, then the rest, with the language noted on Dutch ones', () => {
  const html = readerVoiceOptionsHTML(voices, 'all', 'alba')
  assert.ok(html.indexOf('Your voices') < html.indexOf('Built in'))
  assert.match(html, /Daan \(Dutch\)/)
  assert.ok(!html.includes('Choose a voice'))
})

test('one language shows just its voices, and does not repeat the language in each name', () => {
  const html = readerVoiceOptionsHTML(voices, 'dutch', 'daan')
  assert.match(html, /value="daan">Daan</)
  assert.ok(!html.includes('Alba') && !html.includes('(Dutch)'))
})

test('a voice outside the chosen language is not switched silently: the list asks for a choice', () => {
  const html = readerVoiceOptionsHTML(voices, 'dutch', 'alba')
  assert.match(html, /<option value="" selected disabled>Choose a voice<\/option>/)
})

test('voice names are escaped in the reader menu', () => {
  const html = readerVoiceOptionsHTML(voices, 'all', 'alba')
  assert.ok(!html.includes('<b>'))
  assert.match(html, /Kevin &lt;b&gt;/)
})

test('language names are readable, including ones the server has not heard of yet', () => {
  assert.equal(languageName('dutch'), 'Dutch')
  assert.equal(languageName('french'), 'French')
  assert.equal(languageName(undefined), 'English')
})
